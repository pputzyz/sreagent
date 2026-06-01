package lark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/pkg/safehttp"
)

// Client is a Lark/Feishu webhook client.
type Client struct {
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a new Lark webhook client with SSRF protection.
func NewClient(logger *zap.Logger) *Client {
	return &Client{
		httpClient: safehttp.NewSafeClient(10 * time.Second),
		logger:     logger,
	}
}

// WebhookResponse represents the response from the Lark webhook API.
type WebhookResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// SendWebhook sends a JSON message to a Lark webhook URL.
func (c *Client) SendWebhook(ctx context.Context, webhookURL string, message interface{}) (*WebhookResponse, error) {
	body, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var webhookResp WebhookResponse
	if err := json.Unmarshal(respBody, &webhookResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if webhookResp.Code != 0 {
		return &webhookResp, fmt.Errorf("webhook error: code=%d, msg=%s", webhookResp.Code, webhookResp.Msg)
	}

	c.logger.Debug("lark webhook sent successfully", zap.String("url", webhookURL))
	return &webhookResp, nil
}

// CardMessage represents a Lark interactive card message.
type CardMessage struct {
	MsgType string `json:"msg_type"`
	Card    Card   `json:"card"`
}

// Card represents the card body.
type Card struct {
	Header   CardHeader    `json:"header"`
	Elements []interface{} `json:"elements"`
}

// CardHeader represents the card header.
// Text is optional and enables i18n header rendering. When set, Lark uses
// Text instead of Title for locale-aware display. Currently Title.PlainText
// is used; upgrade path: set Text with a "zh_cn"/"en_us" map.
type CardHeader struct {
	Title    CardText   `json:"title"`
	Template string     `json:"template"`
	Text     *CardText  `json:"text,omitempty"`
}

// CardText represents a text element.
type CardText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// CardMarkdown represents a markdown element.
type CardMarkdown struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// CardDivider represents a divider element.
type CardDivider struct {
	Tag string `json:"tag"`
}

// CardAction represents an action element with buttons.
type CardAction struct {
	Tag     string        `json:"tag"`
	Actions []interface{} `json:"actions"`
}

// CardButton represents a button in an action element.
type CardButton struct {
	Tag       string      `json:"tag"`
	Text      CardText    `json:"text"`
	URL       string      `json:"url,omitempty"`
	Type      string      `json:"type"`
	Value     interface{} `json:"value,omitempty"`
	Behaviour string      `json:"behaviour,omitempty"`
}

// CardSelectMenu represents a select_static dropdown in a Lark card form.
type CardSelectMenu struct {
	Tag          string            `json:"tag"`
	Placeholder  *CardText         `json:"placeholder,omitempty"`
	Name         string            `json:"name"`
	Options      []CardSelectOption `json:"options"`
	DefaultValue string            `json:"value,omitempty"`
}

// CardSelectOption is a single option in a select_static menu.
type CardSelectOption struct {
	Text  CardText `json:"text"`
	Value string   `json:"value"`
}

// CardInput represents a text input field in a Lark card form.
type CardInput struct {
	Tag         string    `json:"tag"`
	Name        string    `json:"name"`
	Placeholder *CardText `json:"placeholder,omitempty"`
	MaxLength   int       `json:"max_length,omitempty"`
}

// CardForm represents a form container in a Lark card.
// Elements inside a form share a common submit action.
type CardForm struct {
	Tag      string        `json:"tag"`
	Elements []interface{} `json:"elements"`
}

// UserOption represents a selectable user for the assign form card.
type UserOption struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	OpenID string `json:"open_id"`
}

// severityTemplate returns the card header color template based on severity.
func severityTemplate(severity string) string {
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

// SeverityTemplate is the exported version of severityTemplate for use by other packages.
func SeverityTemplate(severity string) string {
	return severityTemplate(severity)
}

// SeverityRank returns a numeric rank for severity comparison (higher = more severe).
func SeverityRank(sev string) int {
	switch strings.ToLower(sev) {
	case "critical":
		return 4
	case "error":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

// severityEmoji returns an emoji indicator for the severity level.
func severityEmoji(severity string) string {
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

// BuildAlertCard builds an interactive card message for alert notifications.
func BuildAlertCard(
	alertName string,
	severity string,
	status string,
	labels map[string]string,
	annotations map[string]string,
	firedAt time.Time,
	platformURL string,
) *CardMessage {
	template := severityTemplate(severity)
	emoji := severityEmoji(severity)

	headerContent := fmt.Sprintf("%s [%s] %s", emoji, strings.ToUpper(severity), alertName)

	// Build labels text
	var labelsBuilder strings.Builder
	for k, v := range labels {
		if k == "alertname" || k == "severity" {
			continue
		}
		fmt.Fprintf(&labelsBuilder, "**%s:** %s\n", k, v)
	}
	labelsText := labelsBuilder.String()
	if labelsText == "" {
		labelsText = "_No additional labels_"
	}

	// Build annotations text
	var annotationsBuilder strings.Builder
	for k, v := range annotations {
		fmt.Fprintf(&annotationsBuilder, "**%s:** %s\n", k, v)
	}
	annotationsText := annotationsBuilder.String()
	if annotationsText == "" {
		annotationsText = "_No annotations_"
	}

	elements := []interface{}{
		// Status and basic info
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("**Status:** %s\n**Severity:** %s\n**Fired At:** %s", status, severity, firedAt.Format("2006-01-02 15:04:05 MST")),
		},
		// Divider
		CardDivider{Tag: "hr"},
		// Labels section
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("**📋 Labels**\n%s", labelsText),
		},
		// Divider
		CardDivider{Tag: "hr"},
		// Annotations section
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("**📝 Annotations**\n%s", annotationsText),
		},
	}

	// Add platform link button if URL is provided
	if platformURL != "" {
		elements = append(elements,
			CardDivider{Tag: "hr"},
			CardAction{
				Tag: "action",
				Actions: []interface{}{
					CardButton{
						Tag:  "button",
						Text: CardText{Tag: "plain_text", Content: "View in SREAgent"},
						URL:  platformURL,
						Type: "primary",
					},
				},
			},
		)
	}

	return &CardMessage{
		MsgType: "interactive",
		Card: Card{
			Header: CardHeader{
				Title:    CardText{Tag: "plain_text", Content: headerContent},
				Template: template,
			},
			Elements: elements,
		},
	}
}

// AIAnalysisResult holds the AI analysis to display in the card.
type AIAnalysisResult struct {
	Summary          string
	ProbableCauses   []string
	Impact           string
	RecommendedSteps []string
}

// BuildEnrichedAlertCard builds a card that combines alert info + AI analysis.
// If analysis is nil (AI disabled or failed), the card is equivalent to the basic alert card.
// actionBaseURL is the no-auth alert action page URL (e.g., http://host/alert-action/{token}).
// If empty, action buttons fall back to platformURL.
// eventID, when non-zero, enables callback buttons (acknowledge/silence) that POST back
// to the server via Lark's card.action.trigger mechanism instead of opening a URL.
func BuildEnrichedAlertCard(
	alertName string,
	severity string,
	status string,
	labels map[string]string,
	annotations map[string]string,
	firedAt time.Time,
	analysis *AIAnalysisResult,
	platformURL string,
	actionBaseURL string,
	eventID uint,
) *CardMessage {
	template := severityTemplate(severity)
	emoji := severityEmoji(severity)

	headerContent := fmt.Sprintf("%s [%s] %s", emoji, strings.ToUpper(severity), alertName)

	// Status section
	statusText := "告警中"
	if strings.ToLower(status) == "resolved" {
		statusText = "已恢复"
	} else if strings.ToLower(status) == "acknowledged" {
		statusText = "已确认"
	}

	elements := []interface{}{
		// Basic info
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("**状态:** %s\n**级别:** %s\n**触发时间:** %s", statusText, severity, firedAt.Format("2006-01-02 15:04:05")),
		},
		CardDivider{Tag: "hr"},
	}

	// Labels section
	var labelsBuilder strings.Builder
	for k, v := range labels {
		if k == "alertname" || k == "severity" {
			continue
		}
		fmt.Fprintf(&labelsBuilder, "%s: %s\n", k, v)
	}
	labelsText := labelsBuilder.String()
	if labelsText == "" {
		labelsText = "_无额外标签_"
	}
	elements = append(elements,
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("📋 **标签**\n%s", labelsText),
		},
		CardDivider{Tag: "hr"},
	)

	// Annotations/description section
	var annotationsBuilder strings.Builder
	for k, v := range annotations {
		fmt.Fprintf(&annotationsBuilder, "**%s:** %s\n", k, v)
	}
	annotationsText := annotationsBuilder.String()
	if annotationsText == "" {
		annotationsText = "_无描述_"
	}
	elements = append(elements,
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("📝 **描述**\n%s", annotationsText),
		},
	)

	// AI Analysis section (only if analysis is provided)
	if analysis != nil {
		var aiContent strings.Builder
		aiContent.WriteString("🤖 **AI 分析**\n\n")

		if analysis.Summary != "" {
			fmt.Fprintf(&aiContent, "**摘要:** %s\n\n", analysis.Summary)
		}

		if len(analysis.ProbableCauses) > 0 {
			aiContent.WriteString("**可能原因:**\n")
			for i, cause := range analysis.ProbableCauses {
				fmt.Fprintf(&aiContent, "%d. %s\n", i+1, cause)
			}
			aiContent.WriteString("\n")
		}

		if analysis.Impact != "" {
			fmt.Fprintf(&aiContent, "**影响范围:** %s\n\n", analysis.Impact)
		}

		if len(analysis.RecommendedSteps) > 0 {
			aiContent.WriteString("**建议操作:**\n")
			for i, step := range analysis.RecommendedSteps {
				fmt.Fprintf(&aiContent, "%d. %s\n", i+1, step)
			}
		}

		elements = append(elements,
			CardDivider{Tag: "hr"},
			CardMarkdown{
				Tag:     "markdown",
				Content: aiContent.String(),
			},
		)
	}

	// Action buttons
	if platformURL != "" || actionBaseURL != "" || eventID > 0 {
		var actions []interface{}

		// "View details" button links to the platform UI
		if platformURL != "" {
			actions = append(actions, CardButton{
				Tag:  "button",
				Text: CardText{Tag: "plain_text", Content: "📊 查看详情"},
				URL:  platformURL,
				Type: "primary",
			})
		}

		if eventID > 0 {
			// Callback buttons: POST to server via card.action.trigger
			actions = append(actions, CardButton{
				Tag:       "button",
				Text:      CardText{Tag: "plain_text", Content: "✅ 认领告警"},
				Type:      "default",
				Behaviour: "callback",
				Value:     map[string]interface{}{"action": "ack", "event_id": eventID},
			})
			actions = append(actions, CardButton{
				Tag:       "button",
				Text:      CardText{Tag: "plain_text", Content: "🔕 静默告警"},
				Type:      "default",
				Behaviour: "callback",
				Value:     map[string]interface{}{"action": "silence", "event_id": eventID},
			})
		} else if actionBaseURL != "" {
			// URL-based action buttons (fallback for webhook delivery)
			actions = append(actions, CardButton{
				Tag:  "button",
				Text: CardText{Tag: "plain_text", Content: "✅ 认领告警"},
				URL:  actionBaseURL + "?action=acknowledge",
				Type: "default",
			})
			actions = append(actions, CardButton{
				Tag:  "button",
				Text: CardText{Tag: "plain_text", Content: "🔕 静默告警"},
				URL:  actionBaseURL + "?action=silence",
				Type: "default",
			})
		} else if platformURL != "" {
			// Fallback to platform URL for acknowledge
			actions = append(actions, CardButton{
				Tag:  "button",
				Text: CardText{Tag: "plain_text", Content: "✅ 认领告警"},
				URL:  platformURL,
				Type: "default",
			})
		}

		if len(actions) > 0 {
			elements = append(elements,
				CardDivider{Tag: "hr"},
				CardAction{
					Tag:     "action",
					Actions: actions,
				},
			)
		}
	}

	return &CardMessage{
		MsgType: "interactive",
		Card: Card{
			Header: CardHeader{
				Title:    CardText{Tag: "plain_text", Content: headerContent},
				Template: template,
			},
			Elements: elements,
		},
	}
}

// BuildWebhookCard builds a rich interactive card for webhook delivery (Lark/Feishu).
// Unlike BuildEnrichedAlertCard, it omits action buttons (webhooks cannot receive callbacks)
// and includes the rendered content as a primary text block.
// If analysis is non-nil, the AI analysis section is appended.
// If platformURL is non-empty, a "View in SREAgent" link button is included.
func BuildWebhookCard(
	alertName string,
	severity string,
	status string,
	labels map[string]string,
	annotations map[string]string,
	firedAt time.Time,
	renderedContent string,
	analysis *AIAnalysisResult,
	platformURL string,
) *CardMessage {
	tmpl := severityTemplate(severity)
	emoji := severityEmoji(severity)

	headerContent := fmt.Sprintf("%s [%s] %s", emoji, strings.ToUpper(severity), alertName)

	// Status section
	statusText := "告警中"
	if strings.ToLower(status) == "resolved" {
		statusText = "已恢复"
	} else if strings.ToLower(status) == "acknowledged" {
		statusText = "已确认"
	}

	elements := []interface{}{
		// Basic info
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("**状态:** %s\n**级别:** %s\n**触发时间:** %s", statusText, severity, firedAt.Format("2006-01-02 15:04:05")),
		},
		CardDivider{Tag: "hr"},
	}

	// Rendered content (from the notification template)
	if renderedContent != "" {
		elements = append(elements,
			CardMarkdown{
				Tag:     "markdown",
				Content: renderedContent,
			},
			CardDivider{Tag: "hr"},
		)
	}

	// Labels section
	var labelsBuilder strings.Builder
	for k, v := range labels {
		if k == "alertname" || k == "severity" {
			continue
		}
		fmt.Fprintf(&labelsBuilder, "%s: %s\n", k, v)
	}
	labelsText := labelsBuilder.String()
	if labelsText == "" {
		labelsText = "_无额外标签_"
	}
	elements = append(elements,
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("📋 **标签**\n%s", labelsText),
		},
		CardDivider{Tag: "hr"},
	)

	// Annotations/description section
	var annotationsBuilder strings.Builder
	for k, v := range annotations {
		fmt.Fprintf(&annotationsBuilder, "**%s:** %s\n", k, v)
	}
	annotationsText := annotationsBuilder.String()
	if annotationsText == "" {
		annotationsText = "_无描述_"
	}
	elements = append(elements,
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("📝 **描述**\n%s", annotationsText),
		},
	)

	// AI Analysis section (only if analysis is provided)
	if analysis != nil {
		var aiContent strings.Builder
		aiContent.WriteString("🤖 **AI 分析**\n\n")

		if analysis.Summary != "" {
			fmt.Fprintf(&aiContent, "**摘要:** %s\n\n", analysis.Summary)
		}

		if len(analysis.ProbableCauses) > 0 {
			aiContent.WriteString("**可能原因:**\n")
			for i, cause := range analysis.ProbableCauses {
				fmt.Fprintf(&aiContent, "%d. %s\n", i+1, cause)
			}
			aiContent.WriteString("\n")
		}

		if analysis.Impact != "" {
			fmt.Fprintf(&aiContent, "**影响范围:** %s\n\n", analysis.Impact)
		}

		if len(analysis.RecommendedSteps) > 0 {
			aiContent.WriteString("**建议操作:**\n")
			for i, step := range analysis.RecommendedSteps {
				fmt.Fprintf(&aiContent, "%d. %s\n", i+1, step)
			}
		}

		elements = append(elements,
			CardDivider{Tag: "hr"},
			CardMarkdown{
				Tag:     "markdown",
				Content: aiContent.String(),
			},
		)
	}

	// Footer with source info
	elements = append(elements,
		CardDivider{Tag: "hr"},
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("_SREAgent · %s_", firedAt.Format("2006-01-02 15:04:05")),
		},
	)

	// Platform link button (read-only, no action buttons for webhooks)
	if platformURL != "" {
		elements = append(elements,
			CardAction{
				Tag: "action",
				Actions: []interface{}{
					CardButton{
						Tag:  "button",
						Text: CardText{Tag: "plain_text", Content: "📊 查看详情"},
						URL:  platformURL,
						Type: "primary",
					},
				},
			},
		)
	}

	return &CardMessage{
		MsgType: "interactive",
		Card: Card{
			Header: CardHeader{
				Title:    CardText{Tag: "plain_text", Content: headerContent},
				Template: tmpl,
			},
			Elements: elements,
		},
	}
}

// BuildTestCard builds a simple test card message for channel verification.
func BuildTestCard() *CardMessage {
	return &CardMessage{
		MsgType: "interactive",
		Card: Card{
			Header: CardHeader{
				Title:    CardText{Tag: "plain_text", Content: "🔔 SREAgent Test Notification"},
				Template: "blue",
			},
			Elements: []interface{}{
				CardMarkdown{
					Tag:     "markdown",
					Content: "This is a **test notification** from SREAgent.\n\nIf you see this message, your notification channel is configured correctly.",
				},
				CardMarkdown{
					Tag:     "markdown",
					Content: fmt.Sprintf("**Sent at:** %s", time.Now().Format("2006-01-02 15:04:05 MST")),
				},
			},
		},
	}
}

// BuildAckResponseCard builds a card shown after a successful acknowledge action.
func BuildAckResponseCard(alertName string) *CardMessage {
	return &CardMessage{
		MsgType: "interactive",
		Card: Card{
			Header: CardHeader{
				Title:    CardText{Tag: "plain_text", Content: "✅ 告警已认领"},
				Template: "green",
			},
			Elements: []interface{}{
				CardMarkdown{
					Tag:     "markdown",
					Content: fmt.Sprintf("告警 **%s** 已被认领。", alertName),
				},
				CardMarkdown{
					Tag:     "markdown",
					Content: fmt.Sprintf("_认领时间: %s_", time.Now().Format("2006-01-02 15:04:05")),
				},
			},
		},
	}
}

// BuildSilenceResponseCard builds a card shown after a successful silence action.
func BuildSilenceResponseCard(alertName string) *CardMessage {
	return &CardMessage{
		MsgType: "interactive",
		Card: Card{
			Header: CardHeader{
				Title:    CardText{Tag: "plain_text", Content: "🔕 告警已静默"},
				Template: "yellow",
			},
			Elements: []interface{}{
				CardMarkdown{
					Tag:     "markdown",
					Content: fmt.Sprintf("告警 **%s** 已被静默。", alertName),
				},
				CardMarkdown{
					Tag:     "markdown",
					Content: fmt.Sprintf("_静默时间: %s_", time.Now().Format("2006-01-02 15:04:05")),
				},
			},
		},
	}
}

// BuildSilenceFormCard builds a card with a form for choosing silence duration and reason.
// The form submits back via Lark's card.action.trigger mechanism.
func BuildSilenceFormCard(eventID uint, alertName string) *CardMessage {
	elements := []interface{}{
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("为告警 **%s** 设置静默:", alertName),
		},
		CardDivider{Tag: "hr"},
		CardForm{
			Tag: "form",
			Elements: []interface{}{
				CardSelectMenu{
					Tag: "select_static",
					Name: "duration",
					Placeholder: &CardText{Tag: "plain_text", Content: "选择静默时长"},
					Options: []CardSelectOption{
						{Text: CardText{Tag: "plain_text", Content: "15 分钟"}, Value: "15"},
						{Text: CardText{Tag: "plain_text", Content: "30 分钟"}, Value: "30"},
						{Text: CardText{Tag: "plain_text", Content: "1 小时"}, Value: "60"},
						{Text: CardText{Tag: "plain_text", Content: "2 小时"}, Value: "120"},
						{Text: CardText{Tag: "plain_text", Content: "4 小时"}, Value: "240"},
						{Text: CardText{Tag: "plain_text", Content: "8 小时"}, Value: "480"},
						{Text: CardText{Tag: "plain_text", Content: "24 小时"}, Value: "1440"},
					},
					DefaultValue: "60",
				},
				CardInput{
					Tag:         "input",
					Name:        "reason",
					Placeholder: &CardText{Tag: "plain_text", Content: "静默原因（可选）"},
					MaxLength:   200,
				},
				CardButton{
					Tag:       "button",
					Text:      CardText{Tag: "plain_text", Content: "确认静默"},
					Type:      "primary",
					Behaviour: "callback",
					Value:     map[string]interface{}{"action": "silence_form", "event_id": eventID},
				},
			},
		},
	}

	return &CardMessage{
		MsgType: "interactive",
		Card: Card{
			Header: CardHeader{
				Title:    CardText{Tag: "plain_text", Content: "🔕 静默告警"},
				Template: "yellow",
			},
			Elements: elements,
		},
	}
}

// BuildAssignFormCard builds a card with a form for assigning an alert to a user.
// The form submits back via Lark's card.action.trigger mechanism.
func BuildAssignFormCard(eventID uint, alertName string, users []UserOption) *CardMessage {
	options := make([]CardSelectOption, 0, len(users))
	for _, u := range users {
		options = append(options, CardSelectOption{
			Text:  CardText{Tag: "plain_text", Content: u.Name},
			Value: fmt.Sprintf("%d", u.ID),
		})
	}

	elements := []interface{}{
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("将告警 **%s** 指派给:", alertName),
		},
		CardDivider{Tag: "hr"},
		CardForm{
			Tag: "form",
			Elements: []interface{}{
				CardSelectMenu{
					Tag: "select_static",
					Name: "assignee",
					Placeholder: &CardText{Tag: "plain_text", Content: "选择值班人员"},
					Options: options,
				},
				CardInput{
					Tag:         "input",
					Name:        "note",
					Placeholder: &CardText{Tag: "plain_text", Content: "备注（可选）"},
					MaxLength:   200,
				},
				CardButton{
					Tag:       "button",
					Text:      CardText{Tag: "plain_text", Content: "确认指派"},
					Type:      "primary",
					Behaviour: "callback",
					Value:     map[string]interface{}{"action": "assign_form", "event_id": eventID},
				},
			},
		},
	}

	return &CardMessage{
		MsgType: "interactive",
		Card: Card{
			Header: CardHeader{
				Title:    CardText{Tag: "plain_text", Content: "👤 指派告警"},
				Template: "blue",
			},
			Elements: elements,
		},
	}
}

// BuildAIResponseCard builds a card displaying an AI conversation response.
// viewURL links to the full AI chat page in SREAgent.
func BuildAIResponseCard(question string, answer string, viewURL string) *CardMessage {
	// Truncate answer for card display (Lark has a ~30KB card limit)
	displayAnswer := answer
	if len(displayAnswer) > 2000 {
		displayAnswer = displayAnswer[:2000] + "\n\n..._(内容过长，请在 SREAgent 中查看完整回复)_"
	}

	elements := []interface{}{
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("**问题:** %s", question),
		},
		CardDivider{Tag: "hr"},
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("🤖 **AI 回复:**\n%s", displayAnswer),
		},
	}

	if viewURL != "" {
		elements = append(elements,
			CardDivider{Tag: "hr"},
			CardAction{
				Tag: "action",
				Actions: []interface{}{
					CardButton{
						Tag:  "button",
						Text: CardText{Tag: "plain_text", Content: "📊 在 SREAgent 中查看"},
						URL:  viewURL,
						Type: "primary",
					},
				},
			},
		)
	}

	return &CardMessage{
		MsgType: "interactive",
		Card: Card{
			Header: CardHeader{
				Title:    CardText{Tag: "plain_text", Content: "🤖 AI 助手"},
				Template: "purple",
			},
			Elements: elements,
		},
	}
}

// BuildErrorResponseCard builds a card shown when a card action fails.
// Includes an optional retry button when eventID and originalAction are provided.
func BuildErrorResponseCard(errMsg string) *CardMessage {
	return BuildErrorResponseCardWithRetry(errMsg, 0, "")
}

// BuildErrorResponseCardWithRetry builds an error card with a retry callback button.
// When eventID > 0 and originalAction is non-empty, a "Retry" button is included.
func BuildErrorResponseCardWithRetry(errMsg string, eventID uint, originalAction string) *CardMessage {
	elements := []interface{}{
		CardMarkdown{
			Tag:     "markdown",
			Content: fmt.Sprintf("操作失败: %s", errMsg),
		},
	}

	if eventID > 0 && originalAction != "" {
		elements = append(elements,
			CardDivider{Tag: "hr"},
			CardAction{
				Tag: "action",
				Actions: []interface{}{
					CardButton{
						Tag:       "button",
						Text:      CardText{Tag: "plain_text", Content: "🔄 重试"},
						Type:      "danger",
						Behaviour: "callback",
						Value: map[string]interface{}{
							"action":          "retry",
							"event_id":        eventID,
							"original_action": originalAction,
						},
					},
				},
			},
		)
	}

	return &CardMessage{
		MsgType: "interactive",
		Card: Card{
			Header: CardHeader{
				Title:    CardText{Tag: "plain_text", Content: "❌ 操作失败"},
				Template: "red",
			},
			Elements: elements,
		},
	}
}
