package model

// NotifyMediaType defines the type of notification media backend.
type NotifyMediaType string

const (
	MediaTypeLarkWebhook NotifyMediaType = "lark_webhook"
	MediaTypeEmail       NotifyMediaType = "email"
	MediaTypeHTTP        NotifyMediaType = "http"
	MediaTypeScript      NotifyMediaType = "script"

	// --- IM Webhook ---
	MediaTypeDingTalkWebhook NotifyMediaType = "dingtalk_webhook"
	MediaTypeWeComWebhook    NotifyMediaType = "wecom_webhook"
	MediaTypeSlackWebhook    NotifyMediaType = "slack_webhook"
	MediaTypeDiscordWebhook  NotifyMediaType = "discord_webhook"
	MediaTypeTelegramBot     NotifyMediaType = "telegram_bot"

	// --- Feishu variants ---
	MediaTypeFeishuWebhook NotifyMediaType = "feishu_webhook"
	MediaTypeFeishuCard    NotifyMediaType = "feishu_card"

	// --- Enterprise App ---
	MediaTypeFeishuApp NotifyMediaType = "feishu_app"
	MediaTypeWeComApp  NotifyMediaType = "wecom_app"

	// --- Incident Management ---
	MediaTypeFlashDuty NotifyMediaType = "flashduty"
	MediaTypePagerDuty NotifyMediaType = "pagerduty"

	// --- SMS ---
	MediaTypeTencentSMS NotifyMediaType = "tencent_sms"
	MediaTypeAliyunSMS  NotifyMediaType = "aliyun_sms"

	// --- Custom HTTP ---
	MediaTypeCustomHTTP NotifyMediaType = "custom_http"
)

// NotifyMedia represents a configurable notification backend (e.g., Lark webhook,
// Email SMTP, HTTP endpoint, or script executor). Each media type has its own
// config schema and optional variable definitions.
type NotifyMedia struct {
	BaseModel
	Name        string          `json:"name" gorm:"size:128;not null"`
	Type        NotifyMediaType `json:"type" gorm:"size:32;not null"`
	Description string          `json:"description" gorm:"size:512"`
	IsEnabled   bool            `json:"is_enabled" gorm:"default:true"`
	// Type-specific configuration (stored as JSON):
	// For HTTP type: {"method":"POST","url":"https://...","headers":{"Content-Type":"application/json"},"body":"{{template}}"}
	// For Email type: {"smtp_host":"...","smtp_port":587,"username":"...","password":"...","from":"..."}
	// For Lark type: {"webhook_url":"https://open.feishu.cn/..."}
	// For Script type: {"path":"/usr/local/bin/notify.sh","args":["{{.AlertName}}","{{.Severity}}"]}
	Config string `json:"config" gorm:"type:text;not null"`
	// Variable definitions - parameters this media accepts
	// JSON: [{"name":"key","label":"Webhook Key","type":"string","required":true}]
	Variables string `json:"variables" gorm:"type:text"`
	// Built-in flag (built-in media cannot be deleted)
	IsBuiltin bool `json:"is_builtin" gorm:"default:false"`
	// TeamID: if set, this media belongs to a specific team (nil = global/shared)
	TeamID *uint `json:"team_id" gorm:"index"`
}

func (NotifyMedia) TableName() string {
	return "notify_medias"
}

// MediaVariable represents a variable definition for a NotifyMedia.
// This is the deserialized form of one element in NotifyMedia.Variables.
type MediaVariable struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Type     string `json:"type"` // string, number, boolean
	Required bool   `json:"required"`
}

// CustomHTTPConfig is the JSON config schema for "custom_http" media type.
// Body supports Go template syntax with the following fields:
//
//	{{.Content}}      - rendered notification content
//	{{.AlertName}}    - alert name
//	{{.Severity}}     - severity level
//	{{.Status}}       - firing / resolved
//	{{.Source}}       - event source
//	{{.EventID}}      - event ID
//	{{.FiredAt}}      - fire time (RFC3339)
//	{{.RuleName}}     - rule name
//	{{.Labels}}       - label map
//	{{.Annotations}}  - annotation map
type CustomHTTPConfig struct {
	URL           string            `json:"url"`
	Method        string            `json:"method"` // GET, POST, PUT
	Headers       map[string]string `json:"headers"`
	Body          string            `json:"body"`           // Go template
	Timeout       int               `json:"timeout"`        // milliseconds, default 30000
	RetryTimes    int               `json:"retry_times"`    // default 3
	RetryInterval int               `json:"retry_interval"` // milliseconds, default 100
}
