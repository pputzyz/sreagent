package service

import (
	"testing"
	"time"
)

func Test_sanitizeLarkMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"escapes brackets", "see [here](url)", "see \\[here\\]\\(url\\)"},
		{"escapes backticks", "use `code`", "use \\`code\\`"},
		{"all special chars", "[link](http://x.com)`code`", "\\[link\\]\\(http://x.com\\)\\`code\\`"},
		{"empty string", "", ""},
		{"no special chars", "alert-name: high-cpu", "alert-name: high-cpu"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeLarkMarkdown(tt.input); got != tt.want {
				t.Errorf("sanitizeLarkMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func Test_formatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"30 seconds", 30 * time.Second, "30s"},
		{"59 seconds", 59 * time.Second, "59s"},
		{"1 minute", 1 * time.Minute, "1m 0s"},
		{"5 minutes 30 seconds", 5*time.Minute + 30*time.Second, "5m 30s"},
		{"59 minutes", 59 * time.Minute, "59m 0s"},
		{"1 hour", 1 * time.Hour, "1h 0m"},
		{"2 hours 15 minutes", 2*time.Hour + 15*time.Minute, "2h 15m"},
		{"23 hours", 23 * time.Hour, "23h 0m"},
		{"1 day", 24 * time.Hour, "1d 0h"},
		{"3 days 5 hours", 3*24*time.Hour + 5*time.Hour, "3d 5h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.d); got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func Test_larkSeverityTemplate(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"critical", "red"},
		{"CRITICAL", "red"},
		{"warning", "orange"},
		{"WARNING", "orange"},
		{"info", "blue"},
		{"INFO", "blue"},
		{"unknown", "blue"},
		{"", "blue"},
	}
	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			if got := larkSeverityTemplate(tt.severity); got != tt.want {
				t.Errorf("larkSeverityTemplate(%q) = %q, want %q", tt.severity, got, tt.want)
			}
		})
	}
}

func Test_larkSeverityEmoji(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"critical", "🔴"},
		{"CRITICAL", "🔴"},
		{"warning", "🟠"},
		{"info", "🔵"},
		{"unknown", "⚪"},
		{"", "⚪"},
	}
	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			if got := larkSeverityEmoji(tt.severity); got != tt.want {
				t.Errorf("larkSeverityEmoji(%q) = %q, want %q", tt.severity, got, tt.want)
			}
		})
	}
}

func Test_BuildResolvedCard(t *testing.T) {
	firedAt := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	resolvedAt := firedAt.Add(2*time.Hour + 30*time.Minute)

	labels := map[string]string{
		"alertname": "HighCPU",
		"severity":  "critical",
		"instance":  "web-01",
		"job":       "node-exporter",
	}

	card := BuildResolvedCard("HighCPU", "critical", "resolved", labels, firedAt, resolvedAt, "https://sre.example.com/alert/1")

	if card.MsgType != "interactive" {
		t.Errorf("MsgType = %q, want interactive", card.MsgType)
	}
	if card.Card.Header.Template != "red" {
		t.Errorf("Header.Template = %q, want red", card.Card.Header.Template)
	}
	if len(card.Card.Elements) < 3 {
		t.Errorf("expected at least 3 elements, got %d", len(card.Card.Elements))
	}
}
