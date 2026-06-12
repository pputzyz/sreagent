package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/labelmatch"
	"github.com/sreagent/sreagent/internal/repository"
)

// ReportScope narrows which alert events a report task covers.
// Stored as JSON in report_tasks.scope.
type ReportScope struct {
	// MatchLabels filters events by labels; supports the platform's standard
	// matcher syntax (exact, !=, =~, !~). Empty = all events.
	MatchLabels map[string]string `json:"match_labels,omitempty"`
	// TimeRangeHours overrides the window derived from report_type.
	TimeRangeHours int `json:"time_range_hours,omitempty"`
}

// parseReportScope decodes the scope JSON; empty/invalid input yields a zero scope.
func parseReportScope(raw string) ReportScope {
	var scope ReportScope
	if raw == "" {
		return scope
	}
	_ = json.Unmarshal([]byte(raw), &scope) // invalid scope degrades to "no filter"
	return scope
}

// windowHours resolves the analysis window: scope override > report type default.
func (s ReportScope) windowHours(reportType string) int {
	if s.TimeRangeHours > 0 {
		return s.TimeRangeHours
	}
	if reportType == "weekly" {
		return 7 * 24
	}
	return 24
}

// ReportAlertStats holds platform-computed alert statistics for one window.
// Every number here comes from DB queries — the LLM only interprets them.
type ReportAlertStats struct {
	WindowHours int       `json:"window_hours"`
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`

	Total      int            `json:"total"`
	PrevTotal  int            `json:"prev_total"` // same-length window immediately before
	BySeverity map[string]int `json:"by_severity"`
	ByStatus   map[string]int `json:"by_status"`

	TopAlerts []NameCount  `json:"top_alerts"` // top 5 by alert_name
	Hourly    []HourlyStat `json:"hourly"`     // event counts bucketed by hour

	MTTAMinutes float64 `json:"mtta_minutes"` // mean time to acknowledge (acked events)
	MTTRMinutes float64 `json:"mttr_minutes"` // mean time to resolve (resolved events)

	Truncated bool `json:"truncated"` // true if the event scan hit the cap
}

// NameCount is a (name, count) aggregation entry.
type NameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// HourlyStat is an hourly event-count bucket.
type HourlyStat struct {
	Hour  string `json:"hour"` // "01-02 15:00"
	Count int    `json:"count"`
}

// reportStatsScanCap bounds how many events one report scans per window.
const reportStatsScanCap = 5000

// GatherReportAlertStats computes the statistics for a report window directly
// from the alert_events table (label filtering applied in memory using the
// platform matcher). This is the "numbers come from the platform, not the
// LLM" hard boundary of the report pipeline.
func GatherReportAlertStats(ctx context.Context, eventRepo *repository.AlertEventRepository, scope ReportScope, reportType string) (*ReportAlertStats, error) {
	hours := scope.windowHours(reportType)
	to := time.Now()
	from := to.Add(-time.Duration(hours) * time.Hour)
	prevFrom := from.Add(-time.Duration(hours) * time.Hour)

	events, truncated, err := scanEvents(ctx, eventRepo, from, to, scope.MatchLabels)
	if err != nil {
		return nil, fmt.Errorf("scan events: %w", err)
	}
	prevEvents, _, err := scanEvents(ctx, eventRepo, prevFrom, from, scope.MatchLabels)
	if err != nil {
		return nil, fmt.Errorf("scan previous window: %w", err)
	}

	stats := &ReportAlertStats{
		WindowHours: hours,
		From:        from,
		To:          to,
		Total:       len(events),
		PrevTotal:   len(prevEvents),
		BySeverity:  make(map[string]int),
		ByStatus:    make(map[string]int),
		Truncated:   truncated,
	}

	nameCounts := make(map[string]int)
	hourly := make(map[string]int)
	var mttaSum, mttrSum time.Duration
	var mttaN, mttrN int

	for i := range events {
		e := &events[i]
		stats.BySeverity[string(e.Severity)]++
		stats.ByStatus[string(e.Status)]++
		nameCounts[e.AlertName]++
		hourly[e.FiredAt.Truncate(time.Hour).Format("01-02 15:00")]++
		if e.AckedAt != nil && e.AckedAt.After(e.FiredAt) {
			mttaSum += e.AckedAt.Sub(e.FiredAt)
			mttaN++
		}
		if e.ResolvedAt != nil && e.ResolvedAt.After(e.FiredAt) {
			mttrSum += e.ResolvedAt.Sub(e.FiredAt)
			mttrN++
		}
	}
	if mttaN > 0 {
		stats.MTTAMinutes = mttaSum.Minutes() / float64(mttaN)
	}
	if mttrN > 0 {
		stats.MTTRMinutes = mttrSum.Minutes() / float64(mttrN)
	}

	// Top 5 alert names (deterministic: count desc, then name asc).
	for name, count := range nameCounts {
		stats.TopAlerts = append(stats.TopAlerts, NameCount{Name: name, Count: count})
	}
	sort.Slice(stats.TopAlerts, func(i, j int) bool {
		if stats.TopAlerts[i].Count != stats.TopAlerts[j].Count {
			return stats.TopAlerts[i].Count > stats.TopAlerts[j].Count
		}
		return stats.TopAlerts[i].Name < stats.TopAlerts[j].Name
	})
	if len(stats.TopAlerts) > 5 {
		stats.TopAlerts = stats.TopAlerts[:5]
	}

	// Hourly buckets in chronological order.
	hourKeys := make([]string, 0, len(hourly))
	for h := range hourly {
		hourKeys = append(hourKeys, h)
	}
	sort.Strings(hourKeys)
	for _, h := range hourKeys {
		stats.Hourly = append(stats.Hourly, HourlyStat{Hour: h, Count: hourly[h]})
	}

	return stats, nil
}

// scanEvents pages through alert events in [from, to) and applies the label
// matcher in memory. Returns truncated=true when the scan cap was reached.
func scanEvents(ctx context.Context, eventRepo *repository.AlertEventRepository, from, to time.Time, matchLabels map[string]string) ([]model.AlertEvent, bool, error) {
	const pageSize = 500
	var matched []model.AlertEvent
	scanned := 0

	for page := 1; ; page++ {
		batch, _, err := eventRepo.ListWithFilter(ctx, repository.AlertEventFilter{
			StartTime: &from,
			EndTime:   &to,
			Page:      page,
			PageSize:  pageSize,
		})
		if err != nil {
			return nil, false, err
		}
		for i := range batch {
			// Match(target, pattern): target = the event's labels.
			if len(matchLabels) == 0 || labelmatch.Match(map[string]string(batch[i].Labels), matchLabels) {
				matched = append(matched, batch[i])
			}
		}
		scanned += len(batch)
		if len(batch) < pageSize {
			return matched, false, nil
		}
		if scanned >= reportStatsScanCap {
			return matched, true, nil
		}
	}
}
