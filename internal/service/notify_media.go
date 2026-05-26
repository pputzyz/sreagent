package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/pkg/safehttp"
	"github.com/sreagent/sreagent/internal/repository"
)

// NotifyMediaService provides CRUD and notification dispatch for media backends.
type NotifyMediaService struct {
	repo   *repository.NotifyMediaRepository
	logger *zap.Logger
}

// NewNotifyMediaService creates a new NotifyMediaService.
func NewNotifyMediaService(
	repo *repository.NotifyMediaRepository,
	logger *zap.Logger,
) *NotifyMediaService {
	return &NotifyMediaService{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new notify media.
func (s *NotifyMediaService) Create(ctx context.Context, media *model.NotifyMedia) error {
	if err := s.repo.Create(ctx, media); err != nil {
		s.logger.Error("failed to create notify media", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns a notify media by its ID.
func (s *NotifyMediaService) GetByID(ctx context.Context, id uint) (*model.NotifyMedia, error) {
	media, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrNotifyMediaNotFound
	}
	return media, nil
}

// List returns a paginated list of notify medias.
func (s *NotifyMediaService) List(ctx context.Context, page, pageSize int) ([]model.NotifyMedia, int64, error) {
	list, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list notify medias", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// Update updates an existing notify media.
func (s *NotifyMediaService) Update(ctx context.Context, media *model.NotifyMedia) error {
	existing, err := s.repo.GetByID(ctx, media.ID)
	if err != nil {
		return apperr.ErrNotifyMediaNotFound
	}

	existing.Name = media.Name
	existing.Type = media.Type
	existing.Description = media.Description
	existing.IsEnabled = media.IsEnabled
	if media.Config != "" {
		existing.Config = media.Config
	}
	if media.Variables != "" {
		existing.Variables = media.Variables
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update notify media", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes a notify media by ID. Built-in media cannot be deleted.
func (s *NotifyMediaService) Delete(ctx context.Context, id uint) error {
	media, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperr.ErrNotifyMediaNotFound
	}

	if media.IsBuiltin {
		return apperr.ErrBuiltinDelete
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete notify media", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// ListEnabled returns all enabled notify medias.
func (s *NotifyMediaService) ListEnabled(ctx context.Context) ([]model.NotifyMedia, error) {
	return s.repo.ListEnabled(ctx)
}

// SendNotification dispatches a notification through the given media with rendered template content.
func (s *NotifyMediaService) SendNotification(ctx context.Context, media *model.NotifyMedia, renderedContent string, data *TemplateData) error {
	if !media.IsEnabled {
		s.logger.Warn("skipping disabled media", zap.Uint("media_id", media.ID), zap.String("media_name", media.Name))
		return nil
	}

	switch media.Type {
	case model.MediaTypeLarkWebhook:
		return s.sendLarkWebhook(ctx, media, renderedContent, data)
	case model.MediaTypeEmail:
		return s.sendEmail(ctx, media, renderedContent, data)
	case model.MediaTypeHTTP:
		return s.sendHTTP(ctx, media, renderedContent, data)
	case model.MediaTypeScript:
		return s.executeScript(ctx, media, renderedContent, data)
	case model.MediaTypeDingTalkWebhook:
		return s.sendDingTalkWebhook(ctx, media, renderedContent, data)
	case model.MediaTypeWeComWebhook:
		return s.sendWeComWebhook(ctx, media, renderedContent, data)
	case model.MediaTypeSlackWebhook:
		return s.sendSlackWebhook(ctx, media, renderedContent, data)
	case model.MediaTypeDiscordWebhook:
		return s.sendDiscordWebhook(ctx, media, renderedContent, data)
	case model.MediaTypeTelegramBot:
		return s.sendTelegramBot(ctx, media, renderedContent, data)
	case model.MediaTypeFeishuWebhook:
		return s.sendFeishuWebhook(ctx, media, renderedContent, data)
	case model.MediaTypeFeishuCard:
		return s.sendFeishuCard(ctx, media, renderedContent, data)
	case model.MediaTypeFeishuApp:
		return s.sendFeishuApp(ctx, media, renderedContent, data)
	case model.MediaTypeWeComApp:
		return s.sendWeComApp(ctx, media, renderedContent, data)
	case model.MediaTypeFlashDuty:
		return s.sendFlashDuty(ctx, media, renderedContent, data)
	case model.MediaTypePagerDuty:
		return s.sendPagerDuty(ctx, media, renderedContent, data)
	case model.MediaTypeTencentSMS:
		return s.sendTencentSMS(ctx, media, renderedContent, data)
	case model.MediaTypeAliyunSMS:
		return s.sendAliyunSMS(ctx, media, renderedContent, data)
	case model.MediaTypeCustomHTTP:
		return s.sendCustomHTTP(ctx, media, renderedContent, data)
	default:
		return fmt.Errorf("unsupported media type: %s", media.Type)
	}
}

// TestMedia sends a test notification through the given media.
func (s *NotifyMediaService) TestMedia(ctx context.Context, id uint) error {
	media, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperr.ErrNotifyMediaNotFound
	}

	testContent := fmt.Sprintf("[SREAgent Test] This is a test notification from media '%s' at %s",
		media.Name, time.Now().Format("2006-01-02 15:04:05"))

	testData := &TemplateData{
		AlertName: "TestAlert",
		Severity:  "info",
		Status:    "firing",
		Labels:    map[string]string{"test": "true"},
		FiredAt:   time.Now(),
		EventID:   0,
		Source:    "sreagent-test",
	}

	return s.SendNotification(ctx, media, testContent, testData)
}

// --- Private dispatch methods ---

// larkWebhookConfig represents the JSON config for Lark webhook media.
type larkWebhookConfig struct {
	WebhookURL string `json:"webhook_url"`
}

// sendLarkWebhook sends a notification via Lark webhook.
func (s *NotifyMediaService) sendLarkWebhook(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg larkWebhookConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid lark webhook config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("lark webhook_url is empty")
	}

	// Build a simple text card payload
	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": fmt.Sprintf("[%s] %s", strings.ToUpper(data.Severity), data.AlertName),
				},
				"template": severityToLarkColor(data.Severity),
			},
			"elements": []interface{}{
				map[string]interface{}{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": content,
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal lark payload: %w", err)
	}

	return s.doHTTPPostWithRetry(ctx, cfg.WebhookURL, "application/json", body, 3, 100)
}

// httpMediaConfig represents the JSON config for HTTP media.
type httpMediaConfig struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"` // template string for body
}

// sendHTTP sends a notification via a generic HTTP request.
func (s *NotifyMediaService) sendHTTP(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg httpMediaConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid http media config: %w", err)
	}
	if cfg.URL == "" {
		return fmt.Errorf("http media url is empty")
	}

	method := cfg.Method
	if method == "" {
		method = "POST"
	}

	// Use rendered content as the body
	reqBody := content

	req, err := http.NewRequestWithContext(ctx, method, cfg.URL, strings.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	// Set headers from config
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}
	// Default content type if not set
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	client := safehttp.NewSafeClient(30 * time.Second)
	retryTimes := 3
	retryIntervalMs := 100
	var lastErr error
	for i := 0; i < retryTimes; i++ {
		req, err := http.NewRequestWithContext(ctx, method, cfg.URL, strings.NewReader(reqBody))
		if err != nil {
			return fmt.Errorf("failed to create http request: %w", err)
		}
		for k, v := range cfg.Headers {
			req.Header.Set(k, v)
		}
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http request failed: %w", err)
			s.logger.Warn("http request transport error, retrying",
				zap.Int("attempt", i+1),
				zap.Int("max_attempts", retryTimes),
				zap.String("url", cfg.URL),
				zap.Error(err),
			)
			time.Sleep(time.Duration(retryIntervalMs) * time.Millisecond)
			continue
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("http request returned status %d: %s", resp.StatusCode, string(respBody))
		}

		s.logger.Info("http notification sent",
			zap.String("media", media.Name),
			zap.String("url", cfg.URL),
			zap.Int("status", resp.StatusCode),
		)
		return nil
	}

	return lastErr
}

// emailMediaConfig represents the JSON config for email media.
type emailMediaConfig struct {
	SMTPHost string   `json:"smtp_host"`
	SMTPPort int      `json:"smtp_port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to"`
	UseTLS   bool     `json:"use_tls"` // implicit TLS (port 465); false = STARTTLS (port 587) or plain
}

// sendEmail sends a notification via SMTP email.
func (s *NotifyMediaService) sendEmail(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg emailMediaConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid email media config: %w", err)
	}
	if cfg.SMTPHost == "" {
		return fmt.Errorf("email smtp_host is empty")
	}
	if len(cfg.To) == 0 {
		return fmt.Errorf("email recipients (to) list is empty")
	}

	port := cfg.SMTPPort
	if port == 0 {
		if cfg.UseTLS {
			port = 465
		} else {
			port = 587
		}
	}

	subject := fmt.Sprintf("[SREAgent] [%s] %s", strings.ToUpper(data.Severity), data.AlertName)
	from := cfg.From
	if from == "" {
		from = cfg.Username
	}

	// Build a minimal RFC-2822 message with UTF-8 content
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", from)
	fmt.Fprintf(&msg, "To: %s\r\n", strings.Join(cfg.To, ", "))
	fmt.Fprintf(&msg, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(content)

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, port)
	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	}

	var sendErr error
	if cfg.UseTLS {
		// Implicit TLS (SMTPS)
		tlsCfg := &tls.Config{ServerName: cfg.SMTPHost}
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr, tlsCfg)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server (TLS): %w", err)
		}
		client, err := smtp.NewClient(conn, cfg.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer func() { _ = client.Close() }()
		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("SMTP auth failed: %w", err)
			}
		}
		sendErr = sendSMTPMessage(client, from, cfg.To, msg.Bytes())
	} else {
		// STARTTLS or plain
		sendErr = smtp.SendMail(addr, auth, from, cfg.To, msg.Bytes())
	}

	if sendErr != nil {
		return fmt.Errorf("failed to send email: %w", sendErr)
	}

	s.logger.Info("email notification sent",
		zap.String("media", media.Name),
		zap.String("smtp_host", cfg.SMTPHost),
		zap.String("from", from),
		zap.Strings("to", cfg.To),
		zap.String("alert", data.AlertName),
	)
	return nil
}

// sendSMTPMessage sends a message using an already-connected SMTP client.
func sendSMTPMessage(client *smtp.Client, from string, to []string, msg []byte) error {
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}

// scriptMediaConfig represents the JSON config for script media.
type scriptMediaConfig struct {
	Path string   `json:"path"`
	Args []string `json:"args"`
}

// executeScript runs an external script to send a notification.
func (s *NotifyMediaService) executeScript(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg scriptMediaConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid script media config: %w", err)
	}
	if cfg.Path == "" {
		return fmt.Errorf("script path is empty")
	}

	cmd := exec.CommandContext(ctx, cfg.Path, cfg.Args...)
	cmd.Stdin = strings.NewReader(content)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("script execution failed: %w, stderr: %s", err, stderr.String())
	}

	s.logger.Info("script notification executed",
		zap.String("media", media.Name),
		zap.String("path", cfg.Path),
		zap.String("output", string(output)),
	)
	return nil
}

// doHTTPPostWithRetry sends an HTTP POST with retry on transport errors.
// Only client.Do failures are retried; HTTP status errors (>=400) are returned immediately.
// retryTimes defaults to 3, retryIntervalMs defaults to 100 if non-positive.
func (s *NotifyMediaService) doHTTPPostWithRetry(ctx context.Context, url, contentType string, body []byte, retryTimes, retryIntervalMs int) error {
	if retryTimes <= 0 {
		retryTimes = 3
	}
	if retryIntervalMs <= 0 {
		retryIntervalMs = 100
	}

	var lastErr error
	for i := 0; i < retryTimes; i++ {
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", contentType)

		client := safehttp.NewSafeClient(30 * time.Second)
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http post failed: %w", err)
			s.logger.Warn("http post transport error, retrying",
				zap.Int("attempt", i+1),
				zap.Int("max_attempts", retryTimes),
				zap.String("url", url),
				zap.Error(err),
			)
			time.Sleep(time.Duration(retryIntervalMs) * time.Millisecond)
			continue
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("http post returned status %d: %s", resp.StatusCode, string(respBody))
		}

		return nil
	}

	return lastErr
}

// severityToLarkColor maps alert severity to Lark card header template color.
func severityToLarkColor(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "red"
	case "warning":
		return "orange"
	case "info":
		return "blue"
	default:
		return "grey"
	}
}

// --- DingTalk Webhook ---

type dingTalkWebhookConfig struct {
	WebhookURL string `json:"webhook_url"`
}

func (s *NotifyMediaService) sendDingTalkWebhook(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg dingTalkWebhookConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid dingtalk config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("dingtalk webhook_url is empty")
	}
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": fmt.Sprintf("[%s] %s", strings.ToUpper(data.Severity), data.AlertName),
			"text":  content,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal dingtalk payload: %w", err)
	}
	return s.doHTTPPostWithRetry(ctx, cfg.WebhookURL, "application/json", body, 3, 100)
}

// --- WeCom Webhook ---

type weComWebhookConfig struct {
	WebhookURL string `json:"webhook_url"`
}

func (s *NotifyMediaService) sendWeComWebhook(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg weComWebhookConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid wecom webhook config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("wecom webhook_url is empty")
	}
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal wecom payload: %w", err)
	}
	return s.doHTTPPostWithRetry(ctx, cfg.WebhookURL, "application/json", body, 3, 100)
}

// --- Slack Webhook ---

type slackWebhookConfig struct {
	WebhookURL string `json:"webhook_url"`
}

func (s *NotifyMediaService) sendSlackWebhook(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg slackWebhookConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid slack config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("slack webhook_url is empty")
	}
	payload := map[string]interface{}{
		"blocks": []interface{}{
			map[string]interface{}{
				"type": "header",
				"text": map[string]string{
					"type": "plain_text",
					"text": fmt.Sprintf("[%s] %s", strings.ToUpper(data.Severity), data.AlertName),
				},
			},
			map[string]interface{}{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": content,
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}
	return s.doHTTPPostWithRetry(ctx, cfg.WebhookURL, "application/json", body, 3, 100)
}

// --- Discord Webhook ---

type discordWebhookConfig struct {
	WebhookURL string `json:"webhook_url"`
}

func (s *NotifyMediaService) sendDiscordWebhook(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg discordWebhookConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid discord config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("discord webhook_url is empty")
	}
	payload := map[string]interface{}{
		"embeds": []interface{}{
			map[string]interface{}{
				"title":       fmt.Sprintf("[%s] %s", strings.ToUpper(data.Severity), data.AlertName),
				"description": content,
				"color":       severityToColor(data.Severity),
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal discord payload: %w", err)
	}
	return s.doHTTPPostWithRetry(ctx, cfg.WebhookURL, "application/json", body, 3, 100)
}

// --- Telegram Bot ---

type telegramBotConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

func (s *NotifyMediaService) sendTelegramBot(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg telegramBotConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid telegram config: %w", err)
	}
	if cfg.BotToken == "" || cfg.ChatID == "" {
		return fmt.Errorf("telegram bot_token or chat_id is empty")
	}
	title := fmt.Sprintf("*[%s] %s*", strings.ToUpper(data.Severity), data.AlertName)
	payload := map[string]interface{}{
		"chat_id":    cfg.ChatID,
		"text":       title + "\n\n" + content,
		"parse_mode": "Markdown",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %w", err)
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	return s.doHTTPPostWithRetry(ctx, url, "application/json", body, 3, 100)
}

// --- Feishu Webhook (CN region, same API as Lark) ---

type feishuWebhookConfig struct {
	WebhookURL string `json:"webhook_url"`
}

func (s *NotifyMediaService) sendFeishuWebhook(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg feishuWebhookConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid feishu webhook config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("feishu webhook_url is empty")
	}
	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": fmt.Sprintf("[%s] %s", strings.ToUpper(data.Severity), data.AlertName),
				},
				"template": severityToLarkColor(data.Severity),
			},
			"elements": []interface{}{
				map[string]interface{}{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": content,
					},
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal feishu payload: %w", err)
	}
	return s.doHTTPPostWithRetry(ctx, cfg.WebhookURL, "application/json", body, 3, 100)
}

// --- Feishu Interactive Card ---

type feishuCardConfig struct {
	WebhookURL string `json:"webhook_url"`
}

func (s *NotifyMediaService) sendFeishuCard(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg feishuCardConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid feishu card config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("feishu card webhook_url is empty")
	}
	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": fmt.Sprintf("[%s] %s", strings.ToUpper(data.Severity), data.AlertName),
				},
				"template": severityToLarkColor(data.Severity),
			},
			"elements": []interface{}{
				map[string]interface{}{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": content,
					},
				},
				map[string]interface{}{
					"tag": "hr",
				},
				map[string]interface{}{
					"tag": "note",
					"elements": []interface{}{
						map[string]interface{}{
							"tag":     "plain_text",
							"content": fmt.Sprintf("SREAgent · %s · %s", data.Source, data.FiredAt.Format("2006-01-02 15:04:05")),
						},
					},
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal feishu card payload: %w", err)
	}
	return s.doHTTPPostWithRetry(ctx, cfg.WebhookURL, "application/json", body, 3, 100)
}

// --- Feishu App (send via tenant_access_token) ---

type feishuAppConfig struct {
	AppID        string `json:"app_id"`
	AppSecret    string `json:"app_secret"`
	ReceiveID    string `json:"receive_id"`
	ReceiveIDType string `json:"receive_id_type"` // open_id, user_id, chat_id, email
}

func (s *NotifyMediaService) sendFeishuApp(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg feishuAppConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid feishu app config: %w", err)
	}
	if cfg.AppID == "" || cfg.AppSecret == "" || cfg.ReceiveID == "" {
		return fmt.Errorf("feishu app_id, app_secret, or receive_id is empty")
	}

	token, err := s.getFeishuTenantToken(ctx, cfg.AppID, cfg.AppSecret)
	if err != nil {
		return fmt.Errorf("failed to get feishu tenant token: %w", err)
	}

	ridType := cfg.ReceiveIDType
	if ridType == "" {
		ridType = "chat_id"
	}

	card := map[string]interface{}{
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": fmt.Sprintf("[%s] %s", strings.ToUpper(data.Severity), data.AlertName),
			},
			"template": severityToLarkColor(data.Severity),
		},
		"elements": []interface{}{
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":     "lark_md",
					"content": content,
				},
			},
		},
	}
	cardJSON, _ := json.Marshal(card)

	payload := map[string]interface{}{
		"receive_id": cfg.ReceiveID,
		"msg_type":   "interactive",
		"content":    string(cardJSON),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal feishu app payload: %w", err)
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=%s", ridType)
	retryTimes := 3
	retryIntervalMs := 100
	client := safehttp.NewSafeClient(30 * time.Second)
	var lastErr error
	for i := 0; i < retryTimes; i++ {
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create feishu app request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("feishu app request failed: %w", err)
			s.logger.Warn("feishu app transport error, retrying",
				zap.Int("attempt", i+1),
				zap.Int("max_attempts", retryTimes),
				zap.Error(err),
			)
			time.Sleep(time.Duration(retryIntervalMs) * time.Millisecond)
			continue
		}
		defer func() { _, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096)); _ = resp.Body.Close() }()

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("feishu app returned status %d: %s", resp.StatusCode, string(respBody))
		}
		return nil
	}

	return lastErr
}

func (s *NotifyMediaService) getFeishuTenantToken(ctx context.Context, appID, appSecret string) (string, error) {
	payload := map[string]string{
		"app_id":     appID,
		"app_secret": appSecret,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := safehttp.NewSafeClient(15 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Code != 0 {
		return "", fmt.Errorf("feishu token error %d: %s", result.Code, result.Msg)
	}
	return result.TenantAccessToken, nil
}

// --- WeCom App (send via access_token) ---

type weComAppConfig struct {
	CorpID     string `json:"corp_id"`
	CorpSecret string `json:"corp_secret"`
	AgentID    int    `json:"agent_id"`
	ToUser     string `json:"to_user"` // "@all" or user ids separated by "|"
}

func (s *NotifyMediaService) sendWeComApp(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg weComAppConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid wecom app config: %w", err)
	}
	if cfg.CorpID == "" || cfg.CorpSecret == "" {
		return fmt.Errorf("wecom corp_id or corp_secret is empty")
	}
	if cfg.AgentID == 0 {
		return fmt.Errorf("wecom agent_id is empty")
	}

	token, err := s.getWeComAccessToken(ctx, cfg.CorpID, cfg.CorpSecret)
	if err != nil {
		return fmt.Errorf("failed to get wecom access token: %w", err)
	}

	toUser := cfg.ToUser
	if toUser == "" {
		toUser = "@all"
	}

	payload := map[string]interface{}{
		"touser":  toUser,
		"msgtype": "markdown",
		"agentid": cfg.AgentID,
		"markdown": map[string]string{
			"content": content,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal wecom app payload: %w", err)
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)
	retryTimes := 3
	retryIntervalMs := 100
	client := safehttp.NewSafeClient(30 * time.Second)
	var lastErr error
	for i := 0; i < retryTimes; i++ {
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create wecom app request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("wecom app request failed: %w", err)
			s.logger.Warn("wecom app transport error, retrying",
				zap.Int("attempt", i+1),
				zap.Int("max_attempts", retryTimes),
				zap.Error(err),
			)
			time.Sleep(time.Duration(retryIntervalMs) * time.Millisecond)
			continue
		}
		defer func() { _, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096)); _ = resp.Body.Close() }()

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("wecom app returned status %d: %s", resp.StatusCode, string(respBody))
		}
		return nil
	}

	return lastErr
}

func (s *NotifyMediaService) getWeComAccessToken(ctx context.Context, corpID, corpSecret string) (string, error) {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", corpID, corpSecret)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	client := safehttp.NewSafeClient(15 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("wecom token error %d: %s", result.ErrCode, result.ErrMsg)
	}
	return result.AccessToken, nil
}

// --- FlashDuty ---

type flashDutyConfig struct {
	IntegrationURL string `json:"integration_url"`
}

func (s *NotifyMediaService) sendFlashDuty(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg flashDutyConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid flashduty config: %w", err)
	}
	if cfg.IntegrationURL == "" {
		return fmt.Errorf("flashduty integration_url is empty")
	}
	payload := map[string]interface{}{
		"event_id":   fmt.Sprintf("%d", data.EventID),
		"alert_name": data.AlertName,
		"severity":   strings.ToUpper(data.Severity),
		"status":     strings.ToUpper(data.Status),
		"description": content,
		"labels":     data.Labels,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal flashduty payload: %w", err)
	}
	return s.doHTTPPostWithRetry(ctx, cfg.IntegrationURL, "application/json", body, 3, 100)
}

// --- PagerDuty ---

type pagerDutyConfig struct {
	RoutingKey string `json:"routing_key"`
}

func (s *NotifyMediaService) sendPagerDuty(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg pagerDutyConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid pagerduty config: %w", err)
	}
	if cfg.RoutingKey == "" {
		return fmt.Errorf("pagerduty routing_key is empty")
	}
	action := "trigger"
	if strings.ToLower(data.Status) == "resolved" {
		action = "resolve"
	}
	payload := map[string]interface{}{
		"routing_key":  cfg.RoutingKey,
		"event_action": action,
		"dedup_key":    fmt.Sprintf("%d", data.EventID),
		"payload": map[string]interface{}{
			"summary":   fmt.Sprintf("[%s] %s", strings.ToUpper(data.Severity), data.AlertName),
			"severity":  strings.ToLower(data.Severity),
			"source":    data.Source,
			"timestamp": data.FiredAt.Format(time.RFC3339),
			"custom_details": map[string]string{
				"labels": fmt.Sprintf("%v", data.Labels),
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal pagerduty payload: %w", err)
	}
	return s.doHTTPPostWithRetry(ctx, "https://events.pagerduty.com/v2/enqueue", "application/json", body, 3, 100)
}

// --- Tencent SMS ---

type tencentSMSConfig struct {
	SecretID   string   `json:"secret_id"`
	SecretKey  string   `json:"secret_key"`
	SdkAppID   string   `json:"sdk_app_id"`
	TemplateID string   `json:"template_id"`
	SignName   string   `json:"sign_name"`
	PhoneNumbers []string `json:"phone_numbers"`
}

func (s *NotifyMediaService) sendTencentSMS(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg tencentSMSConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid tencent sms config: %w", err)
	}
	if cfg.SecretID == "" || cfg.SecretKey == "" || len(cfg.PhoneNumbers) == 0 {
		return fmt.Errorf("tencent sms secret_id, secret_key, or phone_numbers is empty")
	}

	// Tencent Cloud SMS API v3 - simplified implementation using their REST API
	// In production, use the official Tencent Cloud SDK for proper signing
	payload := map[string]interface{}{
		"SmsSdkAppId": cfg.SdkAppID,
		"SignName":    cfg.SignName,
		"TemplateId":  cfg.TemplateID,
		"PhoneNumberSet": cfg.PhoneNumbers,
		"TemplateParamSet": []string{
			data.AlertName,
			strings.ToUpper(data.Severity),
			data.Status,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal tencent sms payload: %w", err)
	}

	retryTimes := 3
	retryIntervalMs := 100
	client := safehttp.NewSafeClient(30 * time.Second)
	var lastErr error
	for i := 0; i < retryTimes; i++ {
		req, err := http.NewRequestWithContext(ctx, "POST", "https://sms.tencentcloudapi.com", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create tencent sms request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-TC-Action", "SendSms")
		req.Header.Set("X-TC-Version", "2021-01-11")
		req.Header.Set("X-TC-Region", "ap-guangzhou")
		// Note: In production, implement TC3-HMAC-SHA256 signing
		req.Header.Set("Authorization", fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s", cfg.SecretID))

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("tencent sms request failed: %w", err)
			s.logger.Warn("tencent sms transport error, retrying",
				zap.Int("attempt", i+1),
				zap.Int("max_attempts", retryTimes),
				zap.Error(err),
			)
			time.Sleep(time.Duration(retryIntervalMs) * time.Millisecond)
			continue
		}
		defer func() { _, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096)); _ = resp.Body.Close() }()

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("tencent sms returned status %d: %s", resp.StatusCode, string(respBody))
		}
		return nil
	}

	return lastErr
}

// --- Aliyun SMS ---

type aliyunSMSConfig struct {
	AccessKeyID     string   `json:"access_key_id"`
	AccessKeySecret string   `json:"access_key_secret"`
	SignName        string   `json:"sign_name"`
	TemplateCode    string   `json:"template_code"`
	PhoneNumbers    []string `json:"phone_numbers"`
}

func (s *NotifyMediaService) sendAliyunSMS(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg aliyunSMSConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid aliyun sms config: %w", err)
	}
	if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" || len(cfg.PhoneNumbers) == 0 {
		return fmt.Errorf("aliyun sms access_key_id, access_key_secret, or phone_numbers is empty")
	}

	// Aliyun SMS API - simplified implementation using their REST API
	// In production, use the official Aliyun SDK for proper POP signing
	templateParam, _ := json.Marshal(map[string]string{
		"alert":  data.AlertName,
		"level":  strings.ToUpper(data.Severity),
		"status": data.Status,
	})

	payload := map[string]string{
		"PhoneNumbers":  strings.Join(cfg.PhoneNumbers, ","),
		"SignName":      cfg.SignName,
		"TemplateCode":  cfg.TemplateCode,
		"TemplateParam": string(templateParam),
	}
	body, _ := json.Marshal(payload)

	retryTimes := 3
	retryIntervalMs := 100
	client := safehttp.NewSafeClient(30 * time.Second)
	var lastErr error
	for i := 0; i < retryTimes; i++ {
		req, err := http.NewRequestWithContext(ctx, "POST", "https://dysmsapi.aliyuncs.com", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create aliyun sms request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		// Note: In production, implement Alibaba Cloud POP signing
		req.Header.Set("Authorization", fmt.Sprintf("acs %s", cfg.AccessKeyID))

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("aliyun sms request failed: %w", err)
			s.logger.Warn("aliyun sms transport error, retrying",
				zap.Int("attempt", i+1),
				zap.Int("max_attempts", retryTimes),
				zap.Error(err),
			)
			time.Sleep(time.Duration(retryIntervalMs) * time.Millisecond)
			continue
		}
		defer func() { _, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096)); _ = resp.Body.Close() }()

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("aliyun sms returned status %d: %s", resp.StatusCode, string(respBody))
		}
		return nil
	}

	return lastErr
}

// --- Custom HTTP ---

// customHTTPRenderData is the data context passed to Go templates in CustomHTTPConfig.Body.
type customHTTPRenderData struct {
	Content     string            `json:"content"`
	AlertName   string            `json:"alert_name"`
	Severity    string            `json:"severity"`
	Status      string            `json:"status"`
	Source      string            `json:"source"`
	EventID     uint              `json:"event_id"`
	FiredAt     string            `json:"fired_at"`
	RuleName    string            `json:"rule_name"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// sendCustomHTTP sends a notification via a DB-configured custom HTTP request.
func (s *NotifyMediaService) sendCustomHTTP(ctx context.Context, media *model.NotifyMedia, content string, data *TemplateData) error {
	var cfg model.CustomHTTPConfig
	if err := json.Unmarshal([]byte(media.Config), &cfg); err != nil {
		return fmt.Errorf("invalid custom_http config: %w", err)
	}
	if cfg.URL == "" {
		return fmt.Errorf("custom_http url is empty")
	}

	method := cfg.Method
	if method == "" {
		method = "POST"
	}

	// Render body template
	renderData := customHTTPRenderData{
		Content:     content,
		AlertName:   data.AlertName,
		Severity:    data.Severity,
		Status:      data.Status,
		Source:      data.Source,
		EventID:     data.EventID,
		FiredAt:     data.FiredAt.Format(time.RFC3339),
		RuleName:    data.RuleName,
		Labels:      data.Labels,
		Annotations: data.Annotations,
	}

	var bodyBuf bytes.Buffer
	if cfg.Body != "" {
		tmpl, err := template.New("custom_http_body").Parse(cfg.Body)
		if err != nil {
			return fmt.Errorf("custom_http body template parse error: %w", err)
		}
		if err := tmpl.Execute(&bodyBuf, renderData); err != nil {
			return fmt.Errorf("custom_http body template execute error: %w", err)
		}
	} else {
		// Default: use rendered content as-is
		bodyBuf.WriteString(content)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30000
	}
	retryTimes := cfg.RetryTimes
	if retryTimes <= 0 {
		retryTimes = 3
	}
	retryInterval := cfg.RetryInterval
	if retryInterval <= 0 {
		retryInterval = 100
	}

	client := safehttp.NewSafeClient(time.Duration(timeout) * time.Millisecond)
	var lastErr error
	for i := 0; i < retryTimes; i++ {
		req, err := http.NewRequestWithContext(ctx, method, cfg.URL, bytes.NewReader(bodyBuf.Bytes()))
		if err != nil {
			return fmt.Errorf("failed to create custom_http request: %w", err)
		}
		for k, v := range cfg.Headers {
			req.Header.Set(k, v)
		}
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("custom_http request failed: %w", err)
			s.logger.Warn("custom_http transport error, retrying",
				zap.Int("attempt", i+1),
				zap.Int("max_attempts", retryTimes),
				zap.String("url", cfg.URL),
				zap.Error(err),
			)
			time.Sleep(time.Duration(retryInterval) * time.Millisecond)
			continue
		}
		defer func() { _, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096)); _ = resp.Body.Close() }()

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("custom_http returned status %d: %s", resp.StatusCode, string(respBody))
		}

		s.logger.Info("custom_http notification sent",
			zap.String("media", media.Name),
			zap.String("url", cfg.URL),
			zap.String("method", method),
			zap.Int("status", resp.StatusCode),
		)
		return nil
	}

	return lastErr
}

// --- Helpers ---

// severityToColor maps severity to a decimal color integer (for Discord embeds).
func severityToColor(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 16711680 // red #FF0000
	case "warning":
		return 16744448 // orange #FF8000
	case "info":
		return 3447003  // blue #3498DB
	default:
		return 9807270  // grey #95A5A6
	}
}
