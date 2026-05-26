package model

// SubscribeRule allows users or teams to subscribe to specific alert events
// based on label matchers, tag filters, and severity filters. Matched events
// are forwarded to the associated NotifyRule for processing.
type SubscribeRule struct {
	BaseModel
	Name        string `json:"name" gorm:"size:128;not null"`
	Description string `json:"description" gorm:"size:512"`
	IsEnabled   bool   `json:"is_enabled" gorm:"default:true"`
	// Match conditions - which events to subscribe to
	MatchLabels   JSONLabels  `json:"match_labels" gorm:"type:json"`
	Severities    string      `json:"severities" gorm:"size:128"`
	TagFilters    []TagFilter `json:"tag_filters" gorm:"serializer:json"`    // advanced tag filters with operators
	DatasourceIDs []uint      `json:"datasource_ids" gorm:"serializer:json"` // filter by datasource (empty = all)
	RuleIDs       []uint      `json:"rule_ids" gorm:"serializer:json"`       // subscribe to specific rules (0 or empty = all rules = global)
	ForDuration   int         `json:"for_duration"`                          // minimum firing duration (seconds) before subscribing
	// What to do with matched events
	NotifyRuleID uint `json:"notify_rule_id" gorm:"index;not null"`
	// Who subscribed
	UserID    *uint `json:"user_id" gorm:"index"`
	TeamID    *uint `json:"team_id" gorm:"index"`
	CreatedBy uint  `json:"created_by" gorm:"index"`
}

func (SubscribeRule) TableName() string {
	return "subscribe_rules"
}
