package model

// InhibitionRule suppresses matching target alerts while a source alert is firing.
// Inspired by Alertmanager's inhibition rules.
//
// When an alert matching SourceMatch is currently firing, any alert matching
// TargetMatch will be suppressed — unless the two alerts differ on all labels
// listed in EqualLabels (if EqualLabels is empty, the target is always suppressed).
type InhibitionRule struct {
	BaseModel
	Name        string `json:"name" gorm:"size:128;not null"`
	Description string `json:"description" gorm:"size:512;not null;default:''"`
	// SourceMatch holds the label matchers for the inhibiting (source) alert.
	// Example: {"alertname":"DatabaseDown","severity":"critical"}
	SourceMatch JSONLabels `json:"source_match" gorm:"type:json;not null"`
	// TargetMatch holds the label matchers for alerts to be suppressed.
	// Example: {"team":"backend"}
	TargetMatch JSONLabels `json:"target_match" gorm:"type:json;not null"`
	// EqualLabels is a comma-separated list of label names whose values must be equal
	// in both the source and target alert for suppression to apply.
	// Empty string means the target is always suppressed when the source fires.
	EqualLabels string `json:"equal_labels" gorm:"size:512;not null;default:''"`
	IsEnabled   bool   `json:"is_enabled" gorm:"not null"` // create handler defaults to true; DB column keeps DEFAULT 1 for seeds
	CreatedBy   uint   `json:"created_by" gorm:"not null;default:0"`
}

func (InhibitionRule) TableName() string { return "inhibition_rules" }
