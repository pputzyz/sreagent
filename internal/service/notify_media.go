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

	return s.doHTTPPost(ctx, cfg.WebhookURL, "application/json", body)
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
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
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

// doHTTPPost is a helper to send an HTTP POST request with SSRF protection.
func (s *NotifyMediaService) doHTTPPost(ctx context.Context, url, contentType string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	client := safehttp.NewSafeClient(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http post failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http post returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
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
