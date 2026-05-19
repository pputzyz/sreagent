package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

const templateRenderTimeout = 5 * time.Second

// TemplateData holds the data available to message templates during rendering.
type TemplateData struct {
	AlertName   string            `json:"alert_name"`
	Severity    string            `json:"severity"`
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	FiredAt     time.Time         `json:"fired_at"`
	Value       string            `json:"value"`
	Duration    string            `json:"duration"`
	RuleName    string            `json:"rule_name"`
	EventID     uint              `json:"event_id"`
	Source      string            `json:"source"`
	// AI analysis fields (may be empty if AI is disabled)
	AIAnalysis *AlertAnalysis `json:"ai_analysis,omitempty"`
}

// MessageTemplateService provides CRUD and rendering for message templates.
type MessageTemplateService struct {
	repo   *repository.MessageTemplateRepository
	logger *zap.Logger
}

// NewMessageTemplateService creates a new MessageTemplateService.
func NewMessageTemplateService(
	repo *repository.MessageTemplateRepository,
	logger *zap.Logger,
) *MessageTemplateService {
	return &MessageTemplateService{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new message template.
func (s *MessageTemplateService) Create(ctx context.Context, tmpl *model.MessageTemplate) error {
	// Check for duplicate name
	existing, err := s.repo.GetByName(ctx, tmpl.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("failed to check template name", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if existing != nil {
		return apperr.WithMessage(apperr.ErrDuplicateName, fmt.Sprintf("template '%s' already exists", tmpl.Name))
	}

	if err := s.repo.Create(ctx, tmpl); err != nil {
		s.logger.Error("failed to create message template", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns a message template by its ID.
func (s *MessageTemplateService) GetByID(ctx context.Context, id uint) (*model.MessageTemplate, error) {
	tmpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrTemplateNotFound
	}
	return tmpl, nil
}

// GetByName returns a message template by its unique name.
func (s *MessageTemplateService) GetByName(ctx context.Context, name string) (*model.MessageTemplate, error) {
	tmpl, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, apperr.ErrTemplateNotFound
	}
	return tmpl, nil
}

// List returns a paginated list of message templates.
func (s *MessageTemplateService) List(ctx context.Context, page, pageSize int) ([]model.MessageTemplate, int64, error) {
	list, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list message templates", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// Update updates an existing message template.
func (s *MessageTemplateService) Update(ctx context.Context, tmpl *model.MessageTemplate) error {
	existing, err := s.repo.GetByID(ctx, tmpl.ID)
	if err != nil {
		return apperr.ErrTemplateNotFound
	}

	existing.Name = tmpl.Name
	existing.Description = tmpl.Description
	existing.Content = tmpl.Content
	existing.Type = tmpl.Type

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update message template", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes a message template by ID. Built-in templates cannot be deleted.
func (s *MessageTemplateService) Delete(ctx context.Context, id uint) error {
	tmpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperr.ErrTemplateNotFound
	}

	if tmpl.IsBuiltin {
		return apperr.ErrBuiltinDelete
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete message template", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// RenderTemplate renders a message template with the given event data.
func (s *MessageTemplateService) RenderTemplate(ctx context.Context, templateID uint, data *TemplateData) (string, error) {
	tmpl, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return "", apperr.ErrTemplateNotFound
	}

	return s.RenderContent(ctx, tmpl.Content, data)
}

// RenderContent renders a Go template string with the given data.
// A 5-second timeout prevents CPU exhaustion from malicious or buggy templates.
func (s *MessageTemplateService) RenderContent(ctx context.Context, content string, data *TemplateData) (string, error) {
	funcMap := template.FuncMap{
		"join": func(items []string, sep string) string {
			result := ""
			for i, item := range items {
				if i > 0 {
					result += sep
				}
				result += item
			}
			return result
		},
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
	}

	t, err := template.New("message").Funcs(funcMap).Parse(content)
	if err != nil {
		s.logger.Error("failed to parse template", zap.Error(err))
		return "", apperr.WithMessage(apperr.ErrTemplateRender, fmt.Sprintf("parse error: %v", err))
	}

	type renderResult struct {
		output string
		err    error
	}

	// Use the caller's context deadline if it is tighter, otherwise default timeout.
	renderCtx, cancel := context.WithTimeout(ctx, templateRenderTimeout)
	defer cancel()

	ch := make(chan renderResult, 1)
	go func() {
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			ch <- renderResult{err: err}
			return
		}
		ch <- renderResult{output: buf.String()}
	}()

	select {
	case <-renderCtx.Done():
		s.logger.Error("template render timed out",
			zap.Duration("timeout", templateRenderTimeout),
			zap.Error(renderCtx.Err()),
		)
		return "", apperr.WithMessage(apperr.ErrTemplateRender, fmt.Sprintf("render timed out after %s", templateRenderTimeout))
	case result := <-ch:
		if result.err != nil {
			s.logger.Error("failed to execute template", zap.Error(result.err))
			return "", apperr.WithMessage(apperr.ErrTemplateRender, fmt.Sprintf("render error: %v", result.err))
		}
		return result.output, nil
	}
}

// RenderPreview renders a template with sample data for preview purposes.
func (s *MessageTemplateService) RenderPreview(ctx context.Context, content string) (string, error) {
	sampleData := &TemplateData{
		AlertName: "HighCPUUsage",
		Severity:  "critical",
		Status:    "firing",
		Labels: map[string]string{
			"instance": "prod-server-01",
			"job":      "node-exporter",
			"env":      "production",
		},
		Annotations: map[string]string{
			"summary":     "CPU usage is above 90% for 5 minutes",
			"description": "The CPU usage on prod-server-01 has been above 90% for the last 5 minutes.",
		},
		FiredAt:  time.Now().Add(-5 * time.Minute),
		Value:    "95.2%",
		Duration: "5m",
		RuleName: "cpu_high_usage",
		EventID:  42,
		Source:   "prometheus",
		AIAnalysis: &AlertAnalysis{
			Summary:        "High CPU usage on production server, likely caused by increased traffic.",
			Severity:       "critical",
			ProbableCauses: []string{"Traffic spike", "Runaway process", "Resource leak"},
			Impact:         "Service degradation for end users",
			RecommendedSteps: []string{
				"Check top processes on the server",
				"Review recent deployments",
				"Scale horizontally if traffic-related",
			},
		},
	}

	return s.RenderContent(ctx, content, sampleData)
}

// EventToTemplateData converts an AlertEvent into TemplateData for template rendering.
func EventToTemplateData(event *model.AlertEvent, analysis *AlertAnalysis) *TemplateData {
	data := &TemplateData{
		AlertName:   event.AlertName,
		Severity:    string(event.Severity),
		Status:      string(event.Status),
		Labels:      event.Labels,
		Annotations: event.Annotations,
		FiredAt:     event.FiredAt,
		EventID:     event.ID,
		Source:      event.Source,
		AIAnalysis:  analysis,
	}

	// Extract value and duration from annotations if available
	if v, ok := event.Annotations["value"]; ok {
		data.Value = v
	}
	if d, ok := event.Annotations["duration"]; ok {
		data.Duration = d
	}
	// Extract rule name from labels if available
	if rn, ok := event.Labels["rule_name"]; ok {
		data.RuleName = rn
	}

	return data
}
