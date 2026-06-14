package model

// MessageTemplate defines a reusable Go template for rendering notification messages.
// Available template variables include: {{.AlertName}} {{.Severity}} {{.Status}}
// {{.Labels}} {{.Annotations}} {{.FiredAt}} {{.Value}} {{.Duration}} {{.RuleName}}
// {{.EventID}} and {{.AIAnalysis}} (if AI pipeline is enabled).
type MessageTemplate struct {
	BaseModel
	Name        string `json:"name" gorm:"size:128;not null;uniqueIndex"`
	Description string `json:"description" gorm:"size:512"`
	// Content is a Go template string — the default/Chinese variant.
	Content string `json:"content" gorm:"type:text;not null"`
	// ContentEN is the optional English variant. When a notification's channel
	// (or recipient) language resolves to "en" and this is non-empty, it is rendered
	// instead of Content; otherwise Content is the fallback.
	ContentEN string `json:"content_en" gorm:"type:text"`
	// Template type determines the output format
	Type      string `json:"type" gorm:"size:32;default:text"` // text, html, markdown, lark_card
	IsBuiltin bool   `json:"is_builtin" gorm:"default:false"`
}

func (MessageTemplate) TableName() string {
	return "message_templates"
}
