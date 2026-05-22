package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/datasource"
	"github.com/sreagent/sreagent/internal/repository"
)

// AlertContext holds the enriched context for an alert, including metric data.
type AlertContext struct {
	AlertName   string
	Severity    string
	Labels      map[string]string
	Annotations map[string]string
	FiredAt     time.Time
	MetricData  string // formatted metric data (if available)
	ContextText string // full formatted context text for LLM
}

// AlertContextBuilder builds enriched context for alerts by pulling metrics from datasources.
type AlertContextBuilder struct {
	ruleRepo    *repository.AlertRuleRepository
	dsRepo      *repository.DataSourceRepository
	queryClient *datasource.QueryClient
	logger      *zap.Logger
}

// NewAlertContextBuilder creates a new AlertContextBuilder.
func NewAlertContextBuilder(
	ruleRepo *repository.AlertRuleRepository,
	dsRepo *repository.DataSourceRepository,
	queryClient *datasource.QueryClient,
	logger *zap.Logger,
) *AlertContextBuilder {
	return &AlertContextBuilder{
		ruleRepo:    ruleRepo,
		dsRepo:      dsRepo,
		queryClient: queryClient,
		logger:      logger,
	}
}

// BuildContext builds a full context for the given alert event, including metric data from the
// associated datasource when available.
func (b *AlertContextBuilder) BuildContext(ctx context.Context, event *model.AlertEvent) (*AlertContext, error) {
	alertCtx := &AlertContext{
		AlertName:   event.AlertName,
		Severity:    string(event.Severity),
		Labels:      event.Labels,
		Annotations: event.Annotations,
		FiredAt:     event.FiredAt,
	}

	// Attempt to pull real metric data from the alert rule's datasource.
	if event.RuleID != nil && *event.RuleID != 0 {
		b.enrichWithMetrics(ctx, alertCtx, *event.RuleID, event.FiredAt)
	}

	alertCtx.ContextText = formatBasicContext(alertCtx)
	return alertCtx, nil
}

// enrichWithMetrics fetches recent metric data for the alert rule expression and appends it to
// alertCtx.MetricData. Errors are logged but do not fail the overall context build.
func (b *AlertContextBuilder) enrichWithMetrics(ctx context.Context, alertCtx *AlertContext, ruleID uint, firedAt time.Time) {
	rule, err := b.ruleRepo.GetByID(ctx, ruleID)
	if err != nil {
		b.logger.Warn("alert context: failed to load alert rule",
			zap.Uint("rule_id", ruleID), zap.Error(err))
		return
	}
	if rule.Expression == "" {
		return
	}

	if rule.DataSourceID == nil {
		b.logger.Warn("alert context: rule has no datasource_id, skipping context query",
			zap.Uint("rule_id", ruleID))
		return
	}
	ds, err := b.dsRepo.GetByID(ctx, *rule.DataSourceID)
	if err != nil {
		b.logger.Warn("alert context: failed to load datasource",
			zap.Uint("datasource_id", *rule.DataSourceID), zap.Error(err))
		return
	}

	// Query the 30 minutes surrounding the alert fire time at 1-minute resolution.
	end := firedAt
	start := firedAt.Add(-30 * time.Minute)
	// Clamp end to now so we don't query the future if firedAt is very recent.
	if end.After(time.Now()) {
		end = time.Now()
	}

	results, err := b.queryClient.RangeQuery(ctx, ds.Endpoint, ds.AuthType, ds.AuthConfig, rule.Expression, start, end, "60s")
	if err != nil {
		b.logger.Warn("alert context: failed to query metric data",
			zap.String("expr", rule.Expression),
			zap.String("datasource", ds.Name),
			zap.Error(err))
		return
	}

	alertCtx.MetricData = formatQueryResults(results)
	b.logger.Debug("alert context: metric data enriched",
		zap.String("rule", rule.Name),
		zap.Int("series", len(results)),
	)
}

// formatQueryResults converts PromQL query results into a concise text block.
func formatQueryResults(results []datasource.QueryResult) string {
	if len(results) == 0 {
		return "(no metric data returned)"
	}

	var sb strings.Builder
	for _, r := range results {
		// Build a label string excluding __name__
		var labelParts []string
		for k, v := range r.Labels {
			if k == "__name__" {
				continue
			}
			labelParts = append(labelParts, fmt.Sprintf("%s=%q", k, v))
		}
		metricName := r.MetricName
		if metricName == "" {
			metricName = "(unnamed)"
		}
		if len(labelParts) > 0 {
			fmt.Fprintf(&sb, "%s{%s}:\n", metricName, strings.Join(labelParts, ", "))
		} else {
			fmt.Fprintf(&sb, "%s:\n", metricName)
		}

		// Show up to the last 10 data points
		values := r.Values
		if len(values) > 10 {
			values = values[len(values)-10:]
		}
		for _, dp := range values {
			fmt.Fprintf(&sb, "  %s  %.4g\n", dp.Timestamp.Format("15:04:05"), dp.Value)
		}
	}
	return sb.String()
}

// formatBasicContext formats the alert context into a text block suitable for LLM consumption.
func formatBasicContext(alertCtx *AlertContext) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "告警名称: %s\n", alertCtx.AlertName)
	fmt.Fprintf(&sb, "告警级别: %s\n", alertCtx.Severity)
	fmt.Fprintf(&sb, "触发时间: %s\n", alertCtx.FiredAt.Format("2006-01-02 15:04:05"))

	if len(alertCtx.Labels) > 0 {
		sb.WriteString("\n标签:\n")
		for k, v := range alertCtx.Labels {
			fmt.Fprintf(&sb, "  %s: %s\n", k, v)
		}
	}

	if len(alertCtx.Annotations) > 0 {
		sb.WriteString("\n描述信息:\n")
		for k, v := range alertCtx.Annotations {
			fmt.Fprintf(&sb, "  %s: %s\n", k, v)
		}
	}

	if alertCtx.MetricData != "" {
		sb.WriteString("\n指标数据:\n")
		sb.WriteString(alertCtx.MetricData)
		sb.WriteString("\n")
	}

	return sb.String()
}
