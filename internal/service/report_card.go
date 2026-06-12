package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/lark"
)

// BuildReportCardJSON renders a report run as a Card 2.0 JSON string:
// stat columns + trend line chart + severity pie + top-alerts bar + AI summary.
// All chart numbers come from platform-computed stats (never the LLM).
func BuildReportCardJSON(taskName string, run *model.ReportRun, stats *ReportAlertStats, detailURL string) (string, error) {
	template := "blue"
	if run.Status == "failed" {
		template = "red"
	}

	builder := lark.NewCardV2Builder().
		Config(&lark.CardV2Config{
			WideScreenMode: true,
			WidthMode:      "fill",
			Summary:        &lark.CardV2Text{Tag: "plain_text", Content: "📊 " + taskName},
		}).
		Header("📊 "+taskName, template)

	if run.Status == "failed" {
		builder.AddMarkdown(fmt.Sprintf("❌ **报告生成失败**\n%s", run.ErrorMsg))
		return builder.BuildJSON()
	}

	if stats != nil {
		// Headline stat columns.
		delta := "—"
		if stats.PrevTotal > 0 {
			pct := float64(stats.Total-stats.PrevTotal) / float64(stats.PrevTotal) * 100
			arrow := "↑"
			if pct < 0 {
				arrow = "↓"
				pct = -pct
			}
			delta = fmt.Sprintf("%s%.0f%%", arrow, pct)
		}
		left := fmt.Sprintf("**告警总数:** %d\n**环比上期:** %s", stats.Total, delta)
		mid := fmt.Sprintf("**Critical:** %d\n**Warning:** %d",
			stats.BySeverity["critical"], stats.BySeverity["warning"])
		right := fmt.Sprintf("**MTTA:** %s\n**MTTR:** %s",
			formatMinutes(stats.MTTAMinutes), formatMinutes(stats.MTTRMinutes))
		builder.AddColumnSet(
			lark.NewColumn(1, lark.NewMarkdown(left)),
			lark.NewColumn(1, lark.NewMarkdown(mid)),
			lark.NewColumn(1, lark.NewMarkdown(right)),
		)

		// Trend line (skip when the window produced no events).
		if len(stats.Hourly) > 1 {
			values := make([]map[string]interface{}, 0, len(stats.Hourly))
			for _, h := range stats.Hourly {
				values = append(values, map[string]interface{}{"time": h.Hour, "count": h.Count})
			}
			builder.AddChart("16:9", lark.NewLineChartSpec(
				fmt.Sprintf("近 %d 小时告警趋势", stats.WindowHours), values, "time", "count"))
		}

		// Severity distribution pie.
		if len(stats.BySeverity) > 0 {
			values := make([]map[string]interface{}, 0, len(stats.BySeverity))
			for sev, count := range stats.BySeverity {
				values = append(values, map[string]interface{}{"type": sev, "value": count})
			}
			builder.AddChart("4:3", lark.NewPieChartSpec("告警等级分布", values, "value", "type"))
		}

		// Top alert sources bar.
		if len(stats.TopAlerts) > 0 {
			values := make([]map[string]interface{}, 0, len(stats.TopAlerts))
			for _, tc := range stats.TopAlerts {
				values = append(values, map[string]interface{}{"alert": tc.Name, "count": tc.Count})
			}
			builder.AddChart("16:9", lark.NewBarChartSpec("Top 告警来源", values, "alert", "count"))
		}

		if stats.Truncated {
			builder.AddMarkdown("*⚠️ 事件量超过扫描上限，统计基于前 5000 条*")
		}
	}

	// AI interpretation.
	if run.ReportSummary != "" {
		builder.AddMarkdown("---\n**🤖 AI 摘要:** " + run.ReportSummary)
	}
	if findings := renderFindings(run.FindingsJSON); findings != "" {
		builder.AddCollapsiblePanel("发现项明细", false, lark.NewMarkdown(findings))
	}

	if detailURL != "" {
		builder.AddActions(lark.NewButton("查看完整报告", "open_url", "primary",
			map[string]interface{}{"default_url": detailURL}))
	}

	return builder.BuildJSON()
}

// renderFindings converts the findings JSON into markdown lines.
func renderFindings(findingsJSON string) string {
	if findingsJSON == "" {
		return ""
	}
	var findings []ReportFinding
	if err := json.Unmarshal([]byte(findingsJSON), &findings); err != nil || len(findings) == 0 {
		return ""
	}
	var b strings.Builder
	for _, f := range findings {
		fmt.Fprintf(&b, "- **[%s] %s** / %s：%s\n", f.Severity, f.Category, f.Object, f.Detail)
	}
	return b.String()
}

// formatMinutes renders a minute count as a compact human duration.
func formatMinutes(min float64) string {
	if min <= 0 {
		return "—"
	}
	if min < 60 {
		return fmt.Sprintf("%.0f 分钟", min)
	}
	return fmt.Sprintf("%.1f 小时", min/60)
}
