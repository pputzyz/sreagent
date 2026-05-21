package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/sreagent/sreagent/internal/pkg/lark"
)

// BuildResolvedCard builds a card for a resolved alert.
func BuildResolvedCard(
	alertName string,
	severity string,
	status string,
	labels map[string]string,
	firedAt time.Time,
	resolvedAt time.Time,
	platformURL string,
) *lark.CardMessage {
	template := larkSeverityTemplate(severity)
	emoji := larkSeverityEmoji(severity)

	headerContent := fmt.Sprintf("%s [RESOLVED] %s", emoji, alertName)

	duration := resolvedAt.Sub(firedAt)
	durationText := formatDuration(duration)

	elements := []interface{}{
		lark.CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("**Status:** Resolved\n**Severity:** %s\n**Fired At:** %s\n**Resolved At:** %s\n**Duration:** %s", severity, firedAt.Format("2006-01-02 15:04:05"), resolvedAt.Format("2006-01-02 15:04:05"), durationText),
		},
		lark.CardDivider{Tag: "hr"},
	}

	// Labels
	var labelsBuilder strings.Builder
	for k, v := range labels {
		if k == "alertname" || k == "severity" {
			continue
		}
		labelsBuilder.WriteString(fmt.Sprintf("**%s:** %s\n", k, v))
	}
	labelsText := labelsBuilder.String()
	if labelsText == "" {
		labelsText = "_No additional labels_"
	}
	elements = append(elements,
		lark.CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("**Labels**\n%s", labelsText),
		},
	)

	// Platform link
	if platformURL != "" {
		elements = append(elements,
			lark.CardDivider{Tag: "hr"},
			lark.CardAction{
				Tag: "action",
				Actions: []interface{}{
					lark.CardButton{
						Tag:  "button",
						Text: lark.CardText{Tag: "plain_text", Content: "View in SREAgent"},
						URL:  platformURL,
						Type: "primary",
					},
				},
			},
		)
	}

	return &lark.CardMessage{
		MsgType: "interactive",
		Card: lark.Card{
			Header: lark.CardHeader{
				Title:    lark.CardText{Tag: "plain_text", Content: headerContent},
				Template: template,
			},
			Elements: elements,
		},
	}
}

// larkSeverityTemplate returns the card header color template based on severity.
func larkSeverityTemplate(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "red"
	case "warning":
		return "orange"
	case "info":
		return "blue"
	default:
		return "blue"
	}
}

// larkSeverityEmoji returns an emoji indicator for the severity level.
func larkSeverityEmoji(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "🔴"
	case "warning":
		return "🟠"
	case "info":
		return "🔵"
	default:
		return "⚪"
	}
}

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours < 24 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	days := hours / 24
	hours = hours % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}
