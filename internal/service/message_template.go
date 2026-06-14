package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/pkg/tplx"
	"github.com/sreagent/sreagent/internal/repository"
)

const templateRenderTimeout = 5 * time.Second

// TemplateData holds the data available to message templates during rendering.
// Fields are aligned with Nightingale's AlertCurEvent for template parity.
type TemplateData struct {
	// Core fields (original)
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

	// Extended fields (Nightingale parity)
	RuleID           uint              `json:"rule_id"`            // alert rule ID
	RuleNote         string            `json:"rule_note"`          // alert rule note/description
	Cate             string            `json:"cate"`               // alert rule category (datasource type)
	GroupName        string            `json:"group_name"`         // business group name
	TargetIdent      string            `json:"target_ident"`       // target host/instance identifier (from labels)
	TargetNote       string            `json:"target_note"`        // target note/description
	TriggerValue     string            `json:"trigger_value"`      // raw trigger value
	TriggerValues    map[string]string `json:"trigger_values"`     // per-query trigger values (for multi-query rules)
	FirstTriggerTime time.Time         `json:"first_trigger_time"` // first time this alert fired
	LastEvalTime     time.Time         `json:"last_eval_time"`     // last evaluation time
	Callbacks        []string          `json:"callbacks"`          // callback URLs
	TagsJSON         []string          `json:"tags_json"`          // labels as "key=value" string array
	IsRecovered      bool              `json:"is_recovered"`       // true if resolved
	DatasourceID     uint              `json:"datasource_id"`      // datasource ID
	DatasourceName   string            `json:"datasource_name"`    // datasource name
	RunbookURL       string            `json:"runbook_url"`        // runbook URL
	GeneratorURL     string            `json:"generator_url"`      // source URL that generated the alert
	FireCount        int               `json:"fire_count"`         // number of times this alert has fired
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
	existing.ContentEN = tmpl.ContentEN
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

// RenderTemplate renders a message template with the given event data (default variant).
func (s *MessageTemplateService) RenderTemplate(ctx context.Context, templateID uint, data *TemplateData) (string, error) {
	return s.RenderTemplateLang(ctx, templateID, data, "")
}

// RenderTemplateLang renders a message template, selecting the body variant for the
// given language. When lang == "en" and the template has a non-empty English variant
// (ContentEN), that variant is rendered; otherwise it falls back to Content. This lets
// a group-broadcast channel's configured language drive the rendered notification body.
func (s *MessageTemplateService) RenderTemplateLang(ctx context.Context, templateID uint, data *TemplateData, lang string) (string, error) {
	tmpl, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return "", apperr.ErrTemplateNotFound
	}

	content := tmpl.Content
	if lang == "en" && strings.TrimSpace(tmpl.ContentEN) != "" {
		content = tmpl.ContentEN
	}
	return s.RenderContent(ctx, content, data)
}

// RenderContent renders a Go template string with the given data.
// A 5-second timeout prevents CPU exhaustion from malicious or buggy templates.
//
// SECURITY (B5-19): Uses html/template which auto-escapes HTML in template actions.
// This prevents XSS when the output is rendered as HTML (Lark rich cards, email).
// Template functions like `unescaped` and `safeHtml` return template.HTML which
// html/template treats as pre-escaped safe content — these still work correctly.
// For plain-text channels (SMS, webhook JSON) the escaping is harmless (no HTML entities
// appear in normal alert data).
func (s *MessageTemplateService) RenderContent(ctx context.Context, content string, data *TemplateData) (string, error) {
	funcMap := tplx.TemplateFuncMap

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
	now := time.Now()
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
		FiredAt:  now.Add(-5 * time.Minute),
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
		// Extended fields (Nightingale parity)
		RuleID:           101,
		RuleNote:         "Monitors CPU usage across production nodes",
		Cate:             "prometheus",
		GroupName:        "Infrastructure",
		TargetIdent:      "prod-server-01",
		TargetNote:       "Primary application server",
		TriggerValue:     "95.2%",
		TriggerValues:    map[string]string{"A": "95.2%"},
		FirstTriggerTime: now.Add(-10 * time.Minute),
		LastEvalTime:     now,
		Callbacks:        []string{"https://example.com/callback"},
		TagsJSON:         []string{"instance=prod-server-01", "job=node-exporter", "env=production"},
		IsRecovered:      false,
		DatasourceID:     1,
		DatasourceName:   "Prometheus",
		RunbookURL:       "https://wiki.example.com/runbooks/high-cpu",
		GeneratorURL:     "http://prometheus:9090/graph?g0.expr=cpu_usage",
		FireCount:        3,
	}

	return s.RenderContent(ctx, content, sampleData)
}

// EventToTemplateData converts an AlertEvent into TemplateData for template rendering.
// rule and ds may be nil; when provided they populate extended fields (RuleNote, GroupName, etc.).
func EventToTemplateData(event *model.AlertEvent, analysis *AlertAnalysis, rule *model.AlertRule, ds *model.DataSource) *TemplateData {
	data := &TemplateData{
		AlertName:    event.AlertName,
		Severity:     string(event.Severity),
		Status:       string(event.Status),
		Labels:       event.Labels,
		Annotations:  event.Annotations,
		FiredAt:      event.FiredAt,
		EventID:      event.ID,
		Source:       event.Source,
		GeneratorURL: event.GeneratorURL,
		FireCount:    event.FireCount,
		AIAnalysis:   analysis,
		IsRecovered:  event.Status == model.EventStatusResolved,
	}

	// Timestamps
	data.FirstTriggerTime = event.FiredAt
	data.LastEvalTime = event.UpdatedAt

	// Extract value and duration from annotations if available
	if v, ok := event.Annotations["value"]; ok {
		data.Value = v
	}
	if d, ok := event.Annotations["duration"]; ok {
		data.Duration = d
	}

	// Build TagsJSON from labels (Nightingale-compatible "key=value" array)
	if event.Labels != nil {
		tags := make([]string, 0, len(event.Labels))
		for k, v := range event.Labels {
			tags = append(tags, k+"="+v)
		}
		data.TagsJSON = tags
	}

	// Populate from AlertRule if available
	if rule != nil {
		if event.RuleID != nil {
			data.RuleID = *event.RuleID
		}
		data.RuleName = rule.Name
		data.RuleNote = rule.Description
		data.Cate = string(rule.DatasourceType)
		data.GroupName = rule.GroupName
		data.RunbookURL = rule.Annotations["runbook_url"] // convention: runbook_url in rule annotations

		// Callbacks from rule annotations (Nightingale stores callback URLs in rule config)
		if cb, ok := rule.Annotations["callbacks"]; ok && cb != "" {
			data.Callbacks = strings.Split(cb, ",")
		}

		// TriggerValue: extract from event annotations (set by evaluator)
		if tv, ok := event.Annotations["trigger_value"]; ok {
			data.TriggerValue = tv
		}
		// TriggerValues: per-query values stored as JSON in annotations
		if tvs, ok := event.Annotations["trigger_values"]; ok && tvs != "" {
			var m map[string]string
			if jsonErr := json.Unmarshal([]byte(tvs), &m); jsonErr == nil {
				data.TriggerValues = m
			}
		}
	}

	// Populate from DataSource if available
	if ds != nil {
		data.DatasourceID = ds.ID
		data.DatasourceName = ds.Name
	} else if event.DataSourceID != nil {
		data.DatasourceID = *event.DataSourceID
	}

	// TargetIdent: prefer explicit label conventions
	if event.Labels != nil {
		if ident, ok := event.Labels["instance"]; ok {
			data.TargetIdent = ident
		} else if ident, ok := event.Labels["ident"]; ok {
			data.TargetIdent = ident
		}
		if note, ok := event.Labels["target_note"]; ok {
			data.TargetNote = note
		}
	}

	// Extract rule name from labels if not set from rule
	if data.RuleName == "" {
		if rn, ok := event.Labels["rule_name"]; ok {
			data.RuleName = rn
		}
	}

	return data
}
