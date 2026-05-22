package handler

import (
	"fmt"
	"net/smtp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/service"
)

// containsNewline returns true if s contains CR or LF characters,
// which could be used for email header injection.
func containsNewline(s string) bool {
	return strings.ContainsAny(s, "\r\n")
}

// stripNewlines removes all CR and LF characters from s.
func stripNewlines(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, s)
}

// sanitizeStringFields removes non-printable control characters (except common whitespace)
// and strips \r\n to prevent header injection when the value is used in email headers.
func sanitizeEmailField(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\r' || r == '\n' {
			continue // strip CR/LF
		}
		if unicode.IsControl(r) && r != '\t' {
			continue // strip other control chars except tab
		}
		b.WriteRune(r)
	}
	return b.String()
}

// SMTPSettingsHandler manages global SMTP configuration.
type SMTPSettingsHandler struct {
	svc *service.SystemSettingService
}

// NewSMTPSettingsHandler creates a new SMTPSettingsHandler.
func NewSMTPSettingsHandler(svc *service.SystemSettingService) *SMTPSettingsHandler {
	return &SMTPSettingsHandler{svc: svc}
}

// GetConfig returns the current global SMTP configuration with password masked.
func (h *SMTPSettingsHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetSMTPConfig(c.Request.Context())
	if err != nil {
		Error(c, err)
		return
	}
	if cfg.Password != "" {
		cfg.Password = "********"
	}
	Success(c, cfg)
}

// UpdateConfig saves global SMTP configuration.
// Sending password = "********" preserves the existing password.
func (h *SMTPSettingsHandler) UpdateConfig(c *gin.Context) {
	var req service.SMTPConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	// Prevent email header injection: reject fields containing \r\n
	if containsNewline(req.From) || containsNewline(req.SMTPHost) || containsNewline(req.Username) {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "SMTP fields must not contain newline characters"))
		return
	}
	// Sanitize fields to strip any remaining control characters
	req.From = sanitizeEmailField(req.From)
	req.SMTPHost = sanitizeEmailField(req.SMTPHost)
	req.Username = sanitizeEmailField(req.Username)

	if req.Password == "********" {
		req.Password = ""
	}
	if err := h.svc.SaveSMTPConfig(c.Request.Context(), req); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// TestConnection sends a test email using the stored SMTP config.
// POST /settings/smtp/test   body: {"to": "user@example.com"}
func (h *SMTPSettingsHandler) TestConnection(c *gin.Context) {
	var req struct {
		To string `json:"to" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	// Prevent email header injection via the "To" field
	if containsNewline(req.To) {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "recipient address must not contain newline characters"))
		return
	}
	req.To = sanitizeEmailField(req.To)

	cfg, err := h.svc.GetSMTPConfig(c.Request.Context())
	if err != nil {
		Error(c, err)
		return
	}
	if !cfg.Enabled || cfg.SMTPHost == "" {
		Error(c, apperr.WithMessage(apperr.ErrMissingParam, "SMTP is not configured or disabled"))
		return
	}
	if cfg.SMTPPort == 0 {
		cfg.SMTPPort = 587
	}
	from := cfg.From
	if from == "" {
		from = cfg.Username
	}

	msg := strings.Join([]string{
		"From: " + from,
		"To: " + req.To,
		"Subject: SREAgent SMTP Test",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"This is a test email from SREAgent. Your SMTP configuration is working correctly.",
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)
	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, from, []string{req.To}, []byte(msg)); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrMissingParam, "SMTP test failed: "+err.Error()))
		return
	}

	Success(c, gin.H{"message": "Test email sent successfully to " + req.To})
}
