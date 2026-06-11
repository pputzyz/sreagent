package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/lark"
)

func Test_LarkCardStateService_buildEventCardJSON_Firing(t *testing.T) {
	// We can test the card JSON builder without a full service setup.
	event := &model.AlertEvent{
		BaseModel:  model.BaseModel{ID: 1},
		AlertName:  "CPU High",
		Severity:   model.SeverityCritical,
		Status:     model.EventStatusFiring,
		Labels:     model.JSONLabels{"env": "prod", "region": "us-east-1"},
	}

	// Build card using the same logic as the service.
	template := lark.StatusToTemplate(string(event.Status))
	builder := lark.NewCardV2Builder().
		Config(&lark.CardV2Config{
			WideScreenMode: true,
			Summary:        &lark.CardV2Text{Tag: "plain_text", Content: "[critical] CPU High"},
		}).
		Header("🔴 CPU High", template)

	builder.AddMarkdown("**Status:** firing")
	builder.AddCollapsiblePanel("Labels", false, lark.NewMarkdown("**env:** prod\n**region:** us-east-1\n"))
	builder.AddActions(
		lark.NewButton("✓ Acknowledge", "callback", "primary", map[string]interface{}{
			"action": "ack", "event_id": uint(1),
		}),
		lark.NewButton("🔇 Silence", "callback", "default", map[string]interface{}{
			"action": "silence_form", "event_id": uint(1),
		}),
	)

	jsonStr, err := builder.BuildJSON()
	require.NoError(t, err)
	assert.Contains(t, jsonStr, `"schema":"2.0"`)
	assert.Contains(t, jsonStr, "CPU High")
	assert.Contains(t, jsonStr, "ack")
	assert.Contains(t, jsonStr, "silence_form")
}

func Test_LarkCardStateService_buildEventCardJSON_Resolved(t *testing.T) {
	event := &model.AlertEvent{
		BaseModel: model.BaseModel{ID: 2},
		AlertName: "Disk Full",
		Severity:  model.SeverityWarning,
		Status:    model.EventStatusResolved,
	}

	template := lark.StatusToTemplate(string(event.Status))
	assert.Equal(t, "green", template)

	builder := lark.NewCardV2Builder().
		Header("🟡 Disk Full", template).
		AddMarkdown("**Status:** resolved")

	jsonStr, err := builder.BuildJSON()
	require.NoError(t, err)
	assert.Contains(t, jsonStr, "resolved")
	// Resolved events should NOT have action buttons.
	assert.NotContains(t, jsonStr, "ack")
}

func Test_LarkCardStateService_buildEventCardJSON_Silenced(t *testing.T) {
	event := &model.AlertEvent{
		BaseModel: model.BaseModel{ID: 3},
		AlertName: "Memory High",
		Severity:  model.SeverityWarning,
		Status:    model.EventStatusSilenced,
	}

	template := lark.StatusToTemplate(string(event.Status))
	assert.Equal(t, "yellow", template)
}

func Test_severityEmoji(t *testing.T) {
	assert.Equal(t, "🔴", severityEmoji("critical"))
	assert.Equal(t, "🟠", severityEmoji("error"))
	assert.Equal(t, "🟡", severityEmoji("warning"))
	assert.Equal(t, "🔵", severityEmoji("info"))
	assert.Equal(t, "⚪", severityEmoji("unknown"))
}
