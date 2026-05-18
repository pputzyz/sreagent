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
	defer resp.Body.Close()

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
type CardHeader struct {
	Title    CardText `json:"title"`
	Template string   `json:"template"`
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
	Tag  string   `json:"tag"`
	Text CardText `json:"text"`
	URL  string   `json:"url"`
	Type string   `json:"type"`
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
		labelsBuilder.WriteString(fmt.Sprintf("**%s:** %s\n", k, v))
	}
	labelsText := labelsBuilder.String()
	if labelsText == "" {
		labelsText = "_No additional labels_"
	}

	// Build annotations text
	var annotationsBuilder strings.Builder
	for k, v := range annotations {
		annotationsBuilder.WriteString(fmt.Sprintf("**%s:** %s\n", k, v))
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
		labelsBuilder.WriteString(fmt.Sprintf("%s: %s\n", k, v))
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
		annotationsBuilder.WriteString(fmt.Sprintf("**%s:** %s\n", k, v))
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
			aiContent.WriteString(fmt.Sprintf("**摘要:** %s\n\n", analysis.Summary))
		}

		if len(analysis.ProbableCauses) > 0 {
			aiContent.WriteString("**可能原因:**\n")
			for i, cause := range analysis.ProbableCauses {
				aiContent.WriteString(fmt.Sprintf("%d. %s\n", i+1, cause))
			}
			aiContent.WriteString("\n")
		}

		if analysis.Impact != "" {
			aiContent.WriteString(fmt.Sprintf("**影响范围:** %s\n\n", analysis.Impact))
		}

		if len(analysis.RecommendedSteps) > 0 {
			aiContent.WriteString("**建议操作:**\n")
			for i, step := range analysis.RecommendedSteps {
				aiContent.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
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
	if platformURL != "" || actionBaseURL != "" {
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

		if actionBaseURL != "" {
			// Acknowledge button
			actions = append(actions, CardButton{
				Tag:  "button",
				Text: CardText{Tag: "plain_text", Content: "✅ 认领告警"},
				URL:  actionBaseURL + "?action=acknowledge",
				Type: "default",
			})
			// Silence button — opens the action page with the silence
			// dropdown pre-selected. The user picks the duration (preset
			// chips 30m/2h/8h/1d/3d/7d/30d or custom) on the page, so we
			// no longer hardcode 1 hour here.
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
