package model

// IntegrationType identifies the type of alert integration.
type IntegrationType string

const (
	IntegrationTypeAlertManager IntegrationType = "alertmanager" // Prometheus AlertManager webhook
	IntegrationTypeGrafana      IntegrationType = "grafana"      // Grafana webhook
	IntegrationTypeStandard     IntegrationType = "standard"     // Generic JSON format
)

// IntegrationMode: exclusive (belongs to a channel) or shared (uses routing rules).
type IntegrationMode string

const (
	IntegrationModeExclusive IntegrationMode = "exclusive" // 专属集成
	IntegrationModeShared    IntegrationMode = "shared"    // 共享集成
)

// Integration represents a webhook endpoint for receiving external alerts.
// Modeled after FlashCat's "告警集成".
type Integration struct {
	BaseModel
	Name        string          `json:"name" gorm:"size:128;not null"`
	Description string          `json:"description" gorm:"size:512"`
	Type        IntegrationType `json:"type" gorm:"size:32;not null;index"`
	Mode        IntegrationMode `json:"mode" gorm:"size:32;not null;default:exclusive"`

	// For exclusive mode: which channel this integration belongs to.
	ChannelID *uint    `json:"channel_id" gorm:"index"`
	Channel   *Channel `json:"channel,omitempty" gorm:"foreignKey:ChannelID"`

	// WebhookToken is the unique token in the webhook URL:
	// POST /api/v1/integrations/{webhook_token}/alerts
	WebhookToken string `json:"webhook_token" gorm:"size:64;uniqueIndex;not null"`

	// PipelineConfig stores the alert processing pipeline as JSON.
	// Array of pipeline steps: severity rewrite, title/desc rewrite, drop, suppress.
	PipelineConfig string `json:"pipeline_config" gorm:"type:json"`

	// LabelEnhancementConfig stores label enrichment rules as JSON.
	LabelEnhancementConfig string `json:"label_enhancement_config" gorm:"type:json"`

	IsEnabled bool `json:"is_enabled" gorm:"default:true"`

	// Counters (denormalized)
	TotalAlerts int `json:"total_alerts" gorm:"default:0"`
}

func (Integration) TableName() string {
	return "integrations"
}

// AlertPipelineStep defines a single step in the alert processing pipeline.
type AlertPipelineStep struct {
	// Action: "rewrite_severity" | "rewrite_title" | "rewrite_description" | "drop" | "suppress"
	Action     string            `json:"action"`
	Conditions []FilterCondition `json:"conditions,omitempty"` // when to apply this step
	// For rewrite actions:
	TargetValue string `json:"target_value,omitempty"` // new severity or template string
	// For suppress:
	SourceConditions []FilterCondition `json:"source_conditions,omitempty"`
	MatchLabels      []string          `json:"match_labels,omitempty"` // labels that must match between source and target
}

// LabelEnhancementRule defines a single label enrichment rule.
type LabelEnhancementRule struct {
	// Type: "extract" | "combine" | "map" | "delete"
	Type string `json:"type"`
	// Conditions: only apply to matching alerts
	Conditions []FilterCondition `json:"conditions,omitempty"`
	// For extract: source field + regex
	SourceField string `json:"source_field,omitempty"` // title, description, labels.xxx
	Regex       string `json:"regex,omitempty"`
	// For combine: template string
	Template string `json:"template,omitempty"`
	// For map: mapping table name or inline map
	MappingSourceLabel string            `json:"mapping_source_label,omitempty"`
	MappingTable       map[string]string `json:"mapping_table,omitempty"`
	// Target label key to create/update
	TargetLabel string `json:"target_label"`
	// Overwrite existing label
	Overwrite bool `json:"overwrite"`
}

// RoutingRule defines how alerts from a shared integration are routed to channels.
type RoutingRule struct {
	BaseModel
	IntegrationID uint `json:"integration_id" gorm:"index;not null"`
	// Conditions: match alert attributes/labels to route.
	Conditions string `json:"conditions" gorm:"type:json"` // []FilterCondition JSON
	// Target channel to route matched alerts to.
	TargetChannelID uint     `json:"target_channel_id" gorm:"index;not null"`
	TargetChannel   *Channel `json:"target_channel,omitempty" gorm:"foreignKey:TargetChannelID"`
	// Priority: lower number = higher priority, evaluated top-to-bottom.
	Priority  int  `json:"priority" gorm:"default:0;index"`
	IsEnabled bool `json:"is_enabled" gorm:"default:true"`
}

func (RoutingRule) TableName() string {
	return "routing_rules"
}
