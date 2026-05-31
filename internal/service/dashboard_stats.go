package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// recoverPanic captures panics in goroutines and converts them to errors.
func recoverPanic(logger *zap.Logger, label string, err *error) {
	if r := recover(); r != nil {
		logger.Error("panic recovered in dashboard goroutine",
			zap.String("metric", label),
			zap.Any("panic", r))
		*err = fmt.Errorf("panic in %s: %v", label, r)
	}
}

// DashboardStatsService provides aggregated statistics for the dashboard.
// Unlike other services it takes *gorm.DB directly because it performs
// complex reporting queries that don't map to a single repository.
type DashboardStatsService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewDashboardStatsService(db *gorm.DB, logger *zap.Logger) *DashboardStatsService {
	return &DashboardStatsService{db: db, logger: logger}
}

// ──────────────────────────── Shared types ────────────────────────────

// DashboardStats represents the aggregated dashboard statistics.
type DashboardStats struct {
	TotalDatasources  int64            `json:"total_datasources"`
	TotalRules        int64            `json:"total_rules"`
	ActiveAlerts      int64            `json:"active_alerts"`
	ResolvedToday     int64            `json:"resolved_today"`
	TotalUsers        int64            `json:"total_users"`
	TotalTeams        int64            `json:"total_teams"`
	SeverityBreakdown map[string]int64 `json:"severity_breakdown"`
}

// MTTRMetric holds the mean, P50, and P95 of a latency distribution.
// All values are seconds; -1 means "no data in window".
type MTTRMetric struct {
	Mean  float64 `json:"mean"`
	P50   float64 `json:"p50"`
	P95   float64 `json:"p95"`
	Count int64   `json:"count"`
}

// SeverityMTTR holds MTTA/MTTR for a single severity level.
type SeverityMTTR struct {
	Severity string     `json:"severity"`
	MTTA     MTTRMetric `json:"mtta"`
	MTTR     MTTRMetric `json:"mttr"`
}

// MTTRStats holds time-to-acknowledge and time-to-resolve statistics over a
// configurable window. Percentiles are computed in application code rather than
// with SQL percentile functions so we stay portable across MySQL versions.
type MTTRStats struct {
	WindowHours int `json:"window_hours"`

	// Overall (all severities combined).
	MTTA MTTRMetric `json:"mtta"`
	MTTR MTTRMetric `json:"mttr"`

	// Per-severity breakdown. Order is critical → warning → info.
	BySeverity []SeverityMTTR `json:"by_severity"`

	// Legacy fields retained for older dashboard builds.
	MTTASeconds   float64 `json:"mtta_seconds"`
	MTTRSeconds   float64 `json:"mttr_seconds"`
	AckedCount    int64   `json:"acked_count"`
	ResolvedCount int64   `json:"resolved_count"`
}

// MTTRTrendPoint is one day of MTTA/MTTR means used to render trend lines.
type MTTRTrendPoint struct {
	Date          string  `json:"date"`
	MTTASeconds   float64 `json:"mtta_seconds"` // -1 if no data that day
	MTTRSeconds   float64 `json:"mttr_seconds"` // -1 if no data that day
	AckedCount    int64   `json:"acked_count"`
	ResolvedCount int64   `json:"resolved_count"`
}

// AlertTrendPoint represents a data point for the alert trend chart.
type AlertTrendPoint struct {
	Date          string `json:"date"`
	FiredCount    int64  `json:"fired_count"`
	ResolvedCount int64  `json:"resolved_count"`
}

// TopRuleItem represents a rule with its alert count for the top-rules endpoint.
type TopRuleItem struct {
	RuleID    *uint  `json:"rule_id"`
	AlertName string `json:"alert_name"`
	Count     int64  `json:"count"`
}

// SeverityHistoryPoint represents per-severity alert counts for a single day.
type SeverityHistoryPoint struct {
	Date   string           `json:"date"`
	Counts map[string]int64 `json:"counts"`
}

// ───────────────── Export report types ─────────────────

// ExportDaySummary holds one day of aggregated export data.
type ExportDaySummary struct {
	Critical, Warning, Info, Resolved int64
	AvgMTTA, AvgMTTR                  float64
}

// ExportTopRule holds a top-rule row in the export.
type ExportTopRule struct {
	AlertName        string
	Cnt              int64
	Critical, Warning, Info int64
}

// ExportData holds all data needed to render the export CSV.
type ExportData struct {
	Dates    []string
	DayMap   map[string]*ExportDaySummary
	TopRules []ExportTopRule
}

// ───────────────── Channel / Team / Incident stats types ─────────────

// ChannelStatsRow holds aggregated stats for a single channel.
type ChannelStatsRow struct {
	ChannelID   uint   `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Total       int64  `json:"total"`
	Triggered   int64  `json:"triggered"`
	Processing  int64  `json:"processing"`
	Closed      int64  `json:"closed"`
	Critical    int64  `json:"critical"`
}

// TeamStatsRow holds aggregated stats for a single team.
type TeamStatsRow struct {
	TeamID      uint    `json:"team_id"`
	TeamName    string  `json:"team_name"`
	Total       int64   `json:"total"`
	Closed      int64   `json:"closed"`
	Critical    int64   `json:"critical"`
	AvgMTTR     float64 `json:"avg_mttr_seconds"`
}

// IncidentTrendPoint holds one day of incident counts.
type IncidentTrendPoint struct {
	Date      string `json:"date"`
	Triggered int64  `json:"triggered"`
	Closed    int64  `json:"closed"`
}

// IncidentStatsResult holds overall incident statistics.
type IncidentStatsResult struct {
	TotalIncidents       int64   `json:"total_incidents"`
	ActiveIncidents      int64   `json:"active_incidents"`
	ClosedToday          int64   `json:"closed_today"`
	CriticalActive       int64   `json:"critical_active"`
	AvgMTTRSeconds       float64 `json:"avg_mttr_seconds"`
	TotalPostMortems     int64   `json:"total_post_mortems"`
	PublishedPostMortems int64   `json:"published_post_mortems"`
}

// ──────────────────────── Helpers ─────────────────────────

// percentile returns the `p` percentile (0-100) of a slice of durations in
// seconds using nearest-rank. The input MUST be sorted ascending.
// Returns -1 when the slice is empty.
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return -1
	}
	if n == 1 {
		return sorted[0]
	}
	// Nearest-rank: ceil(p/100 * n) — result index in [1, n].
	rank := int((p/100.0)*float64(n) + 0.9999999)
	if rank < 1 {
		rank = 1
	}
	if rank > n {
		rank = n
	}
	return sorted[rank-1]
}

// computeMetric builds an MTTRMetric from an unsorted []float64 of seconds.
func computeMetric(durations []float64) MTTRMetric {
	n := len(durations)
	if n == 0 {
		return MTTRMetric{Mean: -1, P50: -1, P95: -1, Count: 0}
	}
	var sum float64
	for _, d := range durations {
		sum += d
	}
	sort.Float64s(durations)
	return MTTRMetric{
		Mean:  sum / float64(n),
		P50:   percentile(durations, 50),
		P95:   percentile(durations, 95),
		Count: int64(n),
	}
}

// ──────────────────────── Service methods ─────────────────────────

// GetStats returns aggregated dashboard statistics.
func (s *DashboardStatsService) GetStats(ctx context.Context) (*DashboardStats, error) {
	var stats DashboardStats

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.db.WithContext(ctx).Model(&model.DataSource{}).Count(&stats.TotalDatasources).Error
	})
	g.Go(func() error {
		return s.db.WithContext(ctx).Model(&model.AlertRule{}).Count(&stats.TotalRules).Error
	})
	g.Go(func() error {
		return s.db.WithContext(ctx).Model(&model.AlertEvent{}).
			Where("status IN ?", []string{
				string(model.EventStatusFiring),
				string(model.EventStatusAcknowledged),
			}).
			Count(&stats.ActiveAlerts).Error
	})
	g.Go(func() error {
		todayStart := time.Now().Truncate(24 * time.Hour)
		return s.db.WithContext(ctx).Model(&model.AlertEvent{}).
			Where("status = ? AND resolved_at >= ?", string(model.EventStatusResolved), todayStart).
			Count(&stats.ResolvedToday).Error
	})
	g.Go(func() error {
		return s.db.WithContext(ctx).Model(&model.User{}).Count(&stats.TotalUsers).Error
	})
	g.Go(func() error {
		return s.db.WithContext(ctx).Model(&model.Team{}).Count(&stats.TotalTeams).Error
	})
	g.Go(func() error {
		type sevRow struct {
			Severity string
			Cnt      int64
		}
		var sevRows []sevRow
		if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
			Select("severity, COUNT(*) AS cnt").
			Where("status IN ?", []string{
				string(model.EventStatusFiring),
				string(model.EventStatusAcknowledged),
			}).
			Group("severity").
			Scan(&sevRows).Error; err != nil {
			return err
		}
		stats.SeverityBreakdown = map[string]int64{
			"critical": 0,
			"warning":  0,
			"info":     0,
		}
		for _, r := range sevRows {
			stats.SeverityBreakdown[r.Severity] = r.Cnt
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		s.logger.Error("dashboard stats query failed", zap.Error(err))
		return &stats, err
	}
	return &stats, nil
}

// GetMTTRStats returns MTTA and MTTR over a configurable window including
// percentiles and severity breakdown.
func (s *DashboardStatsService) GetMTTRStats(ctx context.Context, hours int) (*MTTRStats, error) {
	if hours <= 0 {
		hours = 24
	}
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	type row struct {
		Severity    string
		AckSeconds  *float64
		RespSeconds *float64
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var rows []row
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select(`severity,
			CASE WHEN acked_at    IS NOT NULL THEN TIMESTAMPDIFF(SECOND, fired_at, acked_at)    END AS ack_seconds,
			CASE WHEN resolved_at IS NOT NULL THEN TIMESTAMPDIFF(SECOND, fired_at, resolved_at) END AS resp_seconds`).
		Where("fired_at >= ? AND deleted_at IS NULL", since).
		Scan(&rows).Error; err != nil {
		s.logger.Error("mttr_stats query failed", zap.Error(err))
		return nil, err
	}

	var allAck, allResp []float64
	perSev := map[string]*struct{ ack, resp []float64 }{}

	for _, r := range rows {
		if r.AckSeconds != nil && *r.AckSeconds >= 0 {
			allAck = append(allAck, *r.AckSeconds)
		}
		if r.RespSeconds != nil && *r.RespSeconds >= 0 {
			allResp = append(allResp, *r.RespSeconds)
		}
		sev := r.Severity
		if sev == "" {
			continue
		}
		bucket, ok := perSev[sev]
		if !ok {
			bucket = &struct{ ack, resp []float64 }{}
			perSev[sev] = bucket
		}
		if r.AckSeconds != nil && *r.AckSeconds >= 0 {
			bucket.ack = append(bucket.ack, *r.AckSeconds)
		}
		if r.RespSeconds != nil && *r.RespSeconds >= 0 {
			bucket.resp = append(bucket.resp, *r.RespSeconds)
		}
	}

	stats := MTTRStats{
		WindowHours: hours,
		MTTA:        computeMetric(allAck),
		MTTR:        computeMetric(allResp),
	}

	for _, sev := range []string{"critical", "warning", "info"} {
		b, ok := perSev[sev]
		if !ok {
			b = &struct{ ack, resp []float64 }{}
		}
		stats.BySeverity = append(stats.BySeverity, SeverityMTTR{
			Severity: sev,
			MTTA:     computeMetric(b.ack),
			MTTR:     computeMetric(b.resp),
		})
	}

	// Legacy mirrors for older frontends.
	stats.MTTASeconds = stats.MTTA.Mean
	stats.MTTRSeconds = stats.MTTR.Mean
	stats.AckedCount = stats.MTTA.Count
	stats.ResolvedCount = stats.MTTR.Count

	return &stats, nil
}

// GetMTTRTrend returns day-by-day MTTA/MTTR means so operators can see
// whether response times are improving or regressing over time.
func (s *DashboardStatsService) GetMTTRTrend(ctx context.Context, days int) ([]MTTRTrendPoint, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type ackRow struct {
		Date   string
		AvgSec float64
		Cnt    int64
	}

	var mttaRows []ackRow
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select(`DATE(fired_at) AS date,
			AVG(TIMESTAMPDIFF(SECOND, fired_at, acked_at)) AS avg_sec,
			COUNT(acked_at) AS cnt`).
		Where("fired_at >= ? AND acked_at IS NOT NULL AND deleted_at IS NULL", since).
		Group("DATE(fired_at)").
		Order("date").
		Scan(&mttaRows).Error; err != nil {
		s.logger.Error("mttr_trend mtta query failed", zap.Error(err))
		return nil, err
	}

	var mttrRows []ackRow
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select(`DATE(fired_at) AS date,
			AVG(TIMESTAMPDIFF(SECOND, fired_at, resolved_at)) AS avg_sec,
			COUNT(resolved_at) AS cnt`).
		Where("fired_at >= ? AND resolved_at IS NOT NULL AND deleted_at IS NULL", since).
		Group("DATE(fired_at)").
		Order("date").
		Scan(&mttrRows).Error; err != nil {
		s.logger.Error("mttr_trend mttr query failed", zap.Error(err))
		return nil, err
	}

	points := map[string]*MTTRTrendPoint{}
	for _, r := range mttaRows {
		p, ok := points[r.Date]
		if !ok {
			p = &MTTRTrendPoint{Date: r.Date, MTTASeconds: -1, MTTRSeconds: -1}
			points[r.Date] = p
		}
		p.MTTASeconds = r.AvgSec
		p.AckedCount = r.Cnt
	}
	for _, r := range mttrRows {
		p, ok := points[r.Date]
		if !ok {
			p = &MTTRTrendPoint{Date: r.Date, MTTASeconds: -1, MTTRSeconds: -1}
			points[r.Date] = p
		}
		p.MTTRSeconds = r.AvgSec
		p.ResolvedCount = r.Cnt
	}

	dates := make([]string, 0, len(points))
	for d := range points {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	result := make([]MTTRTrendPoint, 0, len(dates))
	for _, d := range dates {
		result = append(result, *points[d])
	}
	return result, nil
}

// GetAlertTrend returns daily fired/resolved counts for trend charts.
func (s *DashboardStatsService) GetAlertTrend(ctx context.Context, days int) ([]AlertTrendPoint, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type dateCount struct {
		Date string
		Cnt  int64
	}

	var firedRows []dateCount
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("DATE(fired_at) AS date, COUNT(*) AS cnt").
		Where("fired_at >= ? AND deleted_at IS NULL", since).
		Group("DATE(fired_at)").Order("date").Scan(&firedRows).Error; err != nil {
		s.logger.Error("alert_trend fired query failed", zap.Error(err))
		return nil, err
	}

	var resolvedRows []dateCount
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("DATE(resolved_at) AS date, COUNT(*) AS cnt").
		Where("resolved_at >= ? AND resolved_at IS NOT NULL AND deleted_at IS NULL", since).
		Group("DATE(resolved_at)").Order("date").Scan(&resolvedRows).Error; err != nil {
		s.logger.Error("alert_trend resolved query failed", zap.Error(err))
		return nil, err
	}

	resolvedMap := map[string]int64{}
	for _, r := range resolvedRows {
		resolvedMap[r.Date] = r.Cnt
	}

	result := make([]AlertTrendPoint, 0, len(firedRows))
	for _, f := range firedRows {
		result = append(result, AlertTrendPoint{
			Date: f.Date, FiredCount: f.Cnt, ResolvedCount: resolvedMap[f.Date],
		})
	}
	return result, nil
}

// GetTopRules returns the most frequently firing alert rules.
func (s *DashboardStatsService) GetTopRules(ctx context.Context, days, limit int) ([]TopRuleItem, error) {
	if days <= 0 {
		days = 30
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	since := time.Now().AddDate(0, 0, -days)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var items []TopRuleItem
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("rule_id, alert_name, COUNT(*) AS count").
		Where("fired_at >= ? AND deleted_at IS NULL", since).
		Group("rule_id, alert_name").
		Order("count DESC").
		Limit(limit).
		Scan(&items).Error; err != nil {
		s.logger.Error("top_rules query failed", zap.Error(err))
		return nil, err
	}
	return items, nil
}

// GetSeverityHistory returns daily alert counts broken down by severity.
func (s *DashboardStatsService) GetSeverityHistory(ctx context.Context, days int) ([]SeverityHistoryPoint, error) {
	if days <= 0 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type row struct {
		Date     string
		Severity string
		Cnt      int64
	}
	var rows []row
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("DATE(fired_at) AS date, severity, COUNT(*) AS cnt").
		Where("fired_at >= ? AND deleted_at IS NULL", since).
		Group("DATE(fired_at), severity").
		Order("date").
		Scan(&rows).Error; err != nil {
		s.logger.Error("severity_history query failed", zap.Error(err))
		return nil, err
	}

	dateMap := map[string]map[string]int64{}
	for _, r := range rows {
		if dateMap[r.Date] == nil {
			dateMap[r.Date] = map[string]int64{"critical": 0, "warning": 0, "info": 0}
		}
		dateMap[r.Date][r.Severity] = r.Cnt
	}

	dates := make([]string, 0, len(dateMap))
	for d := range dateMap {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	result := make([]SeverityHistoryPoint, 0, len(dates))
	for _, d := range dates {
		result = append(result, SeverityHistoryPoint{Date: d, Counts: dateMap[d]})
	}
	return result, nil
}

// ExportReport gathers all data needed to produce the CSV export.
// The caller is responsible for writing CSV rows.
func (s *DashboardStatsService) ExportReport(ctx context.Context, startDate, endDate time.Time) (*ExportData, error) {
	const dateFmt = "2006-01-02"

	startTS := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.Local)
	endTS := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, time.Local)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Per-day fired counts by severity
	type sevDayRow struct {
		Date     string
		Severity string
		Cnt      int64
	}
	var sevRows []sevDayRow
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("DATE(fired_at) AS date, severity, COUNT(*) AS cnt").
		Where("fired_at BETWEEN ? AND ? AND deleted_at IS NULL", startTS, endTS).
		Group("DATE(fired_at), severity").
		Order("date").
		Scan(&sevRows).Error; err != nil {
		s.logger.Error("export_report sev_rows query failed", zap.Error(err))
		return nil, err
	}

	// Per-day resolved counts
	type dayCount struct {
		Date string
		Cnt  int64
	}
	var resolvedRows []dayCount
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("DATE(resolved_at) AS date, COUNT(*) AS cnt").
		Where("resolved_at BETWEEN ? AND ? AND deleted_at IS NULL", startTS, endTS).
		Group("DATE(resolved_at)").
		Scan(&resolvedRows).Error; err != nil {
		s.logger.Error("export_report resolved_rows query failed", zap.Error(err))
		return nil, err
	}

	// Per-day MTTA / MTTR (mean)
	type ttaRow struct {
		Date   string
		AvgSec *float64
	}
	var mttaRows, mttrRows []ttaRow
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("DATE(fired_at) AS date, AVG(TIMESTAMPDIFF(SECOND, fired_at, acked_at)) AS avg_sec").
		Where("fired_at BETWEEN ? AND ? AND acked_at IS NOT NULL AND deleted_at IS NULL", startTS, endTS).
		Group("DATE(fired_at)").Scan(&mttaRows).Error; err != nil {
		s.logger.Error("export_report mtta query failed", zap.Error(err))
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("DATE(fired_at) AS date, AVG(TIMESTAMPDIFF(SECOND, fired_at, resolved_at)) AS avg_sec").
		Where("fired_at BETWEEN ? AND ? AND resolved_at IS NOT NULL AND deleted_at IS NULL", startTS, endTS).
		Group("DATE(fired_at)").Scan(&mttrRows).Error; err != nil {
		s.logger.Error("export_report mttr query failed", zap.Error(err))
		return nil, err
	}

	// Top rules in range
	type topRuleRow struct {
		AlertName string
		Cnt       int64
		Critical  int64
		Warning   int64
		Info      int64
	}
	var topRows []topRuleRow
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select(`alert_name,
			COUNT(*) AS cnt,
			SUM(CASE WHEN severity='critical' THEN 1 ELSE 0 END) AS critical,
			SUM(CASE WHEN severity='warning'  THEN 1 ELSE 0 END) AS warning,
			SUM(CASE WHEN severity='info'     THEN 1 ELSE 0 END) AS info`).
		Where("fired_at BETWEEN ? AND ? AND deleted_at IS NULL", startTS, endTS).
		Group("alert_name").Order("cnt DESC").Limit(20).
		Scan(&topRows).Error; err != nil {
		s.logger.Error("export_report top_rules query failed", zap.Error(err))
		return nil, err
	}

	// Merge into day-keyed maps
	dayMap := map[string]*ExportDaySummary{}
	ensureDay := func(d string) *ExportDaySummary {
		if dayMap[d] == nil {
			dayMap[d] = &ExportDaySummary{AvgMTTA: -1, AvgMTTR: -1}
		}
		return dayMap[d]
	}
	for _, r := range sevRows {
		ds := ensureDay(r.Date)
		switch r.Severity {
		case "critical":
			ds.Critical = r.Cnt
		case "warning":
			ds.Warning = r.Cnt
		case "info":
			ds.Info = r.Cnt
		}
	}
	for _, r := range resolvedRows {
		ensureDay(r.Date).Resolved = r.Cnt
	}
	for _, r := range mttaRows {
		if r.AvgSec != nil {
			ensureDay(r.Date).AvgMTTA = *r.AvgSec / 60.0
		}
	}
	for _, r := range mttrRows {
		if r.AvgSec != nil {
			ensureDay(r.Date).AvgMTTR = *r.AvgSec / 60.0
		}
	}

	// Build sorted date list (fill every calendar day in range)
	var dates []string
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		key := d.Format(dateFmt)
		ensureDay(key)
		dates = append(dates, key)
	}
	sort.Strings(dates)

	// Convert top rules
	exportTop := make([]ExportTopRule, 0, len(topRows))
	for _, r := range topRows {
		exportTop = append(exportTop, ExportTopRule(r))
	}

	return &ExportData{
		Dates:    dates,
		DayMap:   dayMap,
		TopRules: exportTop,
	}, nil
}

// ChannelStats returns incident statistics grouped by channel.
func (s *DashboardStatsService) ChannelStats(ctx context.Context, days int) ([]ChannelStatsRow, error) {
	if days <= 0 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var rows []ChannelStatsRow
	err := s.db.WithContext(ctx).Table("incidents").
		Select(`incidents.channel_id,
			channels.name AS channel_name,
			COUNT(*) AS total,
			SUM(CASE WHEN incidents.status='triggered' THEN 1 ELSE 0 END) AS triggered,
			SUM(CASE WHEN incidents.status='processing' THEN 1 ELSE 0 END) AS processing,
			SUM(CASE WHEN incidents.status='closed' THEN 1 ELSE 0 END) AS closed,
			SUM(CASE WHEN incidents.severity='critical' THEN 1 ELSE 0 END) AS critical`).
		Joins("LEFT JOIN channels ON channels.id = incidents.channel_id AND channels.deleted_at IS NULL").
		Where("incidents.deleted_at IS NULL AND incidents.triggered_at >= ?", since).
		Group("incidents.channel_id, channels.name").
		Order("total DESC").
		Scan(&rows).Error
	if err != nil {
		s.logger.Error("channel_stats query failed", zap.Error(err))
		return nil, err
	}
	return rows, nil
}

// TeamStats returns incident statistics grouped by team.
func (s *DashboardStatsService) TeamStats(ctx context.Context, days int) ([]TeamStatsRow, error) {
	if days <= 0 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var rows []TeamStatsRow
	err := s.db.WithContext(ctx).Table("incidents").
		Select(`channels.team_id,
			teams.name AS team_name,
			COUNT(*) AS total,
			SUM(CASE WHEN incidents.status='closed' THEN 1 ELSE 0 END) AS closed,
			SUM(CASE WHEN incidents.severity='critical' THEN 1 ELSE 0 END) AS critical,
			COALESCE(AVG(CASE WHEN incidents.closed_at IS NOT NULL
				THEN TIMESTAMPDIFF(SECOND, incidents.triggered_at, incidents.closed_at)
				ELSE NULL END), 0) AS avg_mttr`).
		Joins("LEFT JOIN channels ON channels.id = incidents.channel_id AND channels.deleted_at IS NULL").
		Joins("LEFT JOIN teams ON teams.id = channels.team_id AND teams.deleted_at IS NULL").
		Where("incidents.deleted_at IS NULL AND incidents.triggered_at >= ?", since).
		Group("channels.team_id, teams.name").
		Order("total DESC").
		Scan(&rows).Error
	if err != nil {
		s.logger.Error("team_stats query failed", zap.Error(err))
		return nil, err
	}
	return rows, nil
}

// IncidentTrend returns daily incident counts for the last N days.
func (s *DashboardStatsService) IncidentTrend(ctx context.Context, days int) ([]IncidentTrendPoint, error) {
	if days <= 0 {
		days = 30
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type row struct {
		Day    string
		Status string
		Count  int64
	}

	var rows []row
	err := s.db.WithContext(ctx).Table("incidents").
		Select("DATE(triggered_at) AS day, status, COUNT(*) AS count").
		Where("deleted_at IS NULL AND triggered_at >= ?", time.Now().AddDate(0, 0, -days)).
		Group("DATE(triggered_at), status").
		Order("day ASC").
		Scan(&rows).Error
	if err != nil {
		s.logger.Error("incident_trend query failed", zap.Error(err))
		return nil, err
	}

	pointMap := make(map[string]*IncidentTrendPoint)
	for _, r := range rows {
		if _, ok := pointMap[r.Day]; !ok {
			pointMap[r.Day] = &IncidentTrendPoint{Date: r.Day}
		}
		switch r.Status {
		case "triggered", "processing":
			pointMap[r.Day].Triggered += r.Count
		case "closed":
			pointMap[r.Day].Closed += r.Count
		}
	}
	result := make([]IncidentTrendPoint, 0, len(pointMap))
	for _, p := range pointMap {
		result = append(result, *p)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Date < result[j].Date })
	return result, nil
}

// IncidentStats returns overall incident statistics.
func (s *DashboardStatsService) IncidentStats(ctx context.Context) (*IncidentStatsResult, error) {
	var stats IncidentStatsResult
	todayStart := time.Now().Truncate(24 * time.Hour)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() (err error) {
		defer recoverPanic(s.logger, "IncidentStats.TotalIncidents", &err)
		return s.db.WithContext(ctx).Model(&model.Incident{}).Count(&stats.TotalIncidents).Error
	})
	g.Go(func() (err error) {
		defer recoverPanic(s.logger, "IncidentStats.ActiveIncidents", &err)
		return s.db.WithContext(ctx).Model(&model.Incident{}).Where("status IN ?", []string{"triggered", "processing"}).Count(&stats.ActiveIncidents).Error
	})
	g.Go(func() (err error) {
		defer recoverPanic(s.logger, "IncidentStats.ClosedToday", &err)
		return s.db.WithContext(ctx).Model(&model.Incident{}).Where("status = 'closed' AND closed_at >= ?", todayStart).Count(&stats.ClosedToday).Error
	})
	g.Go(func() (err error) {
		defer recoverPanic(s.logger, "IncidentStats.CriticalActive", &err)
		return s.db.WithContext(ctx).Model(&model.Incident{}).Where("status IN ? AND severity = 'critical'", []string{"triggered", "processing"}).Count(&stats.CriticalActive).Error
	})
	g.Go(func() (err error) {
		defer recoverPanic(s.logger, "IncidentStats.AvgMTTR", &err)
		return s.db.WithContext(ctx).Table("incidents").
			Where("closed_at IS NOT NULL AND deleted_at IS NULL").
			Select("COALESCE(AVG(TIMESTAMPDIFF(SECOND, triggered_at, closed_at)), 0)").
			Scan(&stats.AvgMTTRSeconds).Error
	})
	g.Go(func() (err error) {
		defer recoverPanic(s.logger, "IncidentStats.TotalPostMortems", &err)
		return s.db.WithContext(ctx).Model(&model.PostMortem{}).Count(&stats.TotalPostMortems).Error
	})
	g.Go(func() (err error) {
		defer recoverPanic(s.logger, "IncidentStats.PublishedPostMortems", &err)
		return s.db.WithContext(ctx).Model(&model.PostMortem{}).Where("status = 'published'").Count(&stats.PublishedPostMortems).Error
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return &stats, nil
}

// GetExportFilename returns the default CSV filename for a date range.
func GetExportFilename(startDate, endDate time.Time) string {
	return fmt.Sprintf("alert-report-%s-to-%s.csv",
		startDate.Format("20060102"), endDate.Format("20060102"))
}
