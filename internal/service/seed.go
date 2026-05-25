package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// SeedService handles seeding default/built-in data on first run.
type SeedService struct {
	mediaRepo    *repository.NotifyMediaRepository
	templateRepo *repository.MessageTemplateRepository
	logger       *zap.Logger
}

// NewSeedService creates a new SeedService.
func NewSeedService(
	mediaRepo *repository.NotifyMediaRepository,
	templateRepo *repository.MessageTemplateRepository,
	logger *zap.Logger,
) *SeedService {
	return &SeedService{
		mediaRepo:    mediaRepo,
		templateRepo: templateRepo,
		logger:       logger,
	}
}

// SeedDefaults seeds all default built-in data. It is idempotent - it skips
// items that already exist (by checking template name uniqueness).
func (s *SeedService) SeedDefaults(ctx context.Context) error {
	s.logger.Info("seeding default notification data...")

	if err := s.seedDefaultMedia(ctx); err != nil {
		s.logger.Error("failed to seed default media", zap.Error(err))
		return err
	}

	if err := s.seedDefaultTemplates(ctx); err != nil {
		s.logger.Error("failed to seed default templates", zap.Error(err))
		return err
	}

	s.logger.Info("default notification data seeded successfully")
	return nil
}

// seedDefaultMedia creates the built-in notification media backends.
func (s *SeedService) seedDefaultMedia(ctx context.Context) error {
	defaultMedias := []model.NotifyMedia{
		{
			Name:        "Default Lark Webhook",
			Type:        model.MediaTypeLarkWebhook,
			Description: "Built-in Lark/Feishu webhook notification media",
			IsEnabled:   true,
			Config:      `{"webhook_url":""}`,
			Variables:   `[{"name":"webhook_url","label":"Webhook URL","type":"string","required":true}]`,
			IsBuiltin:   true,
		},
		{
			Name:        "Default Email",
			Type:        model.MediaTypeEmail,
			Description: "Built-in email notification media via SMTP",
			IsEnabled:   false,
			Config:      `{"smtp_host":"","smtp_port":587,"username":"","password":"","from":""}`,
			Variables: `[{"name":"smtp_host","label":"SMTP Host","type":"string","required":true},` +
				`{"name":"smtp_port","label":"SMTP Port","type":"number","required":true},` +
				`{"name":"username","label":"Username","type":"string","required":true},` +
				`{"name":"password","label":"Password","type":"string","required":true},` +
				`{"name":"from","label":"From Address","type":"string","required":true}]`,
			IsBuiltin: true,
		},
		{
			Name:        "Default DingTalk Webhook",
			Type:        model.MediaTypeDingTalkWebhook,
			Description: "Built-in DingTalk robot webhook notification media",
			IsEnabled:   false,
			Config:      `{"webhook_url":""}`,
			Variables:   `[{"name":"webhook_url","label":"Webhook URL","type":"string","required":true}]`,
			IsBuiltin:   true,
		},
		{
			Name:        "Default WeCom Webhook",
			Type:        model.MediaTypeWeComWebhook,
			Description: "Built-in WeCom (Enterprise WeChat) robot webhook notification media",
			IsEnabled:   false,
			Config:      `{"webhook_url":""}`,
			Variables:   `[{"name":"webhook_url","label":"Webhook URL","type":"string","required":true}]`,
			IsBuiltin:   true,
		},
		{
			Name:        "Default Slack Webhook",
			Type:        model.MediaTypeSlackWebhook,
			Description: "Built-in Slack incoming webhook notification media",
			IsEnabled:   false,
			Config:      `{"webhook_url":""}`,
			Variables:   `[{"name":"webhook_url","label":"Webhook URL","type":"string","required":true}]`,
			IsBuiltin:   true,
		},
		{
			Name:        "Default Discord Webhook",
			Type:        model.MediaTypeDiscordWebhook,
			Description: "Built-in Discord webhook notification media",
			IsEnabled:   false,
			Config:      `{"webhook_url":""}`,
			Variables:   `[{"name":"webhook_url","label":"Webhook URL","type":"string","required":true}]`,
			IsBuiltin:   true,
		},
		{
			Name:        "Default Telegram Bot",
			Type:        model.MediaTypeTelegramBot,
			Description: "Built-in Telegram bot notification media",
			IsEnabled:   false,
			Config:      `{"bot_token":"","chat_id":""}`,
			Variables: `[{"name":"bot_token","label":"Bot Token","type":"string","required":true},` +
				`{"name":"chat_id","label":"Chat ID","type":"string","required":true}]`,
			IsBuiltin: true,
		},
		{
			Name:        "Default FlashDuty",
			Type:        model.MediaTypeFlashDuty,
			Description: "Built-in FlashDuty incident management notification media",
			IsEnabled:   false,
			Config:      `{"integration_url":""}`,
			Variables:   `[{"name":"integration_url","label":"Integration URL","type":"string","required":true}]`,
			IsBuiltin:   true,
		},
		{
			Name:        "Default PagerDuty",
			Type:        model.MediaTypePagerDuty,
			Description: "Built-in PagerDuty Events API v2 notification media",
			IsEnabled:   false,
			Config:      `{"routing_key":""}`,
			Variables:   `[{"name":"routing_key","label":"Routing Key","type":"string","required":true}]`,
			IsBuiltin:   true,
		},
	}

	for i := range defaultMedias {
		media := &defaultMedias[i]
		// Check if already exists by listing and matching name
		existing, _, err := s.mediaRepo.List(ctx, 1, 1000)
		if err != nil {
			return err
		}
		found := false
		for _, e := range existing {
			if e.Name == media.Name {
				found = true
				break
			}
		}
		if found {
			s.logger.Debug("default media already exists, skipping", zap.String("name", media.Name))
			continue
		}

		if err := s.mediaRepo.Create(ctx, media); err != nil {
			return err
		}
		s.logger.Info("seeded default media", zap.String("name", media.Name))
	}

	return nil
}

// seedDefaultTemplates creates the built-in message templates.
func (s *SeedService) seedDefaultTemplates(ctx context.Context) error {
	defaultTemplates := []model.MessageTemplate{
		{
			Name:        "default-text",
			Description: "Default plain text alert notification template",
			Type:        "text",
			IsBuiltin:   true,
			Content:     defaultTextTemplate,
		},
		{
			Name:        "default-markdown",
			Description: "Default Markdown alert notification template",
			Type:        "markdown",
			IsBuiltin:   true,
			Content:     defaultMarkdownTemplate,
		},
		{
			Name:        "default-lark-card",
			Description: "Default Lark interactive card template with AI analysis support",
			Type:        "lark_card",
			IsBuiltin:   true,
			Content:     defaultLarkCardTemplate,
		},
	}

	for i := range defaultTemplates {
		tmpl := &defaultTemplates[i]
		// Check if already exists by name
		existing, err := s.templateRepo.GetByName(ctx, tmpl.Name)
		if err != nil {
			return err
		}
		if existing != nil {
			s.logger.Debug("default template already exists, skipping", zap.String("name", tmpl.Name))
			continue
		}

		if err := s.templateRepo.Create(ctx, tmpl); err != nil {
			return err
		}
		s.logger.Info("seeded default template", zap.String("name", tmpl.Name))
	}

	return nil
}

// --- Default template content ---

const defaultTextTemplate = `[{{.Severity | upper}}] {{.AlertName}}
Status: {{.Status}}
Fired At: {{formatTime .FiredAt}}
Source: {{.Source}}
{{- if .Value}}
Value: {{.Value}}
{{- end}}
{{- if .Duration}}
Duration: {{.Duration}}
{{- end}}

Labels:
{{- range $key, $value := .Labels}}
  {{$key}}: {{$value}}
{{- end}}

{{- if .Annotations}}
Annotations:
{{- range $key, $value := .Annotations}}
  {{$key}}: {{$value}}
{{- end}}
{{- end}}

{{- if .AIAnalysis}}

AI Analysis:
Summary: {{.AIAnalysis.Summary}}
Impact: {{.AIAnalysis.Impact}}
Probable Causes:
{{- range .AIAnalysis.ProbableCauses}}
  - {{.}}
{{- end}}
Recommended Steps:
{{- range .AIAnalysis.RecommendedSteps}}
  - {{.}}
{{- end}}
{{- end}}`

const defaultMarkdownTemplate = `## {{.Severity}} Alert: {{.AlertName}}

**Status:** {{.Status}}
**Fired At:** {{formatTime .FiredAt}}
**Source:** {{.Source}}
{{- if .Value}}
**Value:** {{.Value}}
{{- end}}
{{- if .Duration}}
**Duration:** {{.Duration}}
{{- end}}

### Labels
{{- range $key, $value := .Labels}}
- **{{$key}}:** {{$value}}
{{- end}}

{{- if .Annotations}}
### Annotations
{{- range $key, $value := .Annotations}}
- **{{$key}}:** {{$value}}
{{- end}}
{{- end}}

{{- if .AIAnalysis}}

---

### AI Analysis
**Summary:** {{.AIAnalysis.Summary}}

**Impact:** {{.AIAnalysis.Impact}}

**Probable Causes:**
{{- range .AIAnalysis.ProbableCauses}}
- {{.}}
{{- end}}

**Recommended Steps:**
{{- range .AIAnalysis.RecommendedSteps}}
1. {{.}}
{{- end}}
{{- end}}`

const defaultLarkCardTemplate = `{
  "header": {
    "title": {
      "tag": "plain_text",
      "content": "[{{.Severity}}] {{.AlertName}}"
    },
    "template": "{{if eq .Severity "critical"}}red{{else if eq .Severity "warning"}}orange{{else}}blue{{end}}"
  },
  "elements": [
    {
      "tag": "div",
      "fields": [
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "**Status:** {{.Status}}"
          }
        },
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "**Severity:** {{.Severity}}"
          }
        },
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "**Fired At:** {{formatTime .FiredAt}}"
          }
        },
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "**Source:** {{.Source}}"
          }
        }
      ]
    },
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**Labels:**\n{{range $key, $value := .Labels}}` + "`" + `{{$key}}` + "`" + `: {{$value}}\n{{end}}"
      }
    }{{if .Annotations}},
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**Details:**\n{{range $key, $value := .Annotations}}{{$key}}: {{$value}}\n{{end}}"
      }
    }{{end}}{{if .AIAnalysis}},
    {
      "tag": "hr"
    },
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**AI Analysis**\n{{.AIAnalysis.Summary}}"
      }
    },
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**Impact:** {{.AIAnalysis.Impact}}\n\n**Probable Causes:**\n{{range .AIAnalysis.ProbableCauses}}- {{.}}\n{{end}}\n**Recommended Steps:**\n{{range .AIAnalysis.RecommendedSteps}}- {{.}}\n{{end}}"
      }
    }{{end}}
  ]
}`
