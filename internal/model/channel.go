package model

import "time"

// ChannelStatus defines the status of a collaboration channel.
type ChannelStatus string

const (
	ChannelStatusActive   ChannelStatus = "active"
	ChannelStatusDisabled ChannelStatus = "disabled"
)

// ChannelAccessLevel defines the visibility of a channel.
type ChannelAccessLevel string

const (
	ChannelAccessPublic  ChannelAccessLevel = "public"
	ChannelAccessPrivate ChannelAccessLevel = "private"
)

// Channel represents a collaboration space for grouping incidents by
// business or team. It is the core unit for noise reduction, dispatch
// and permission isolation — modeled after FlashCat's "协作空间".
type Channel struct {
	BaseModel
	Name        string             `json:"name" gorm:"size:128;not null;uniqueIndex"`
	Description string             `json:"description" gorm:"size:512"`
	TeamID      *uint              `json:"team_id" gorm:"index"`
	Team        *Team              `json:"team,omitempty" gorm:"foreignKey:TeamID"`
	Status      ChannelStatus      `json:"status" gorm:"size:32;not null;default:active;index"`
	AccessLevel ChannelAccessLevel `json:"access_level" gorm:"size:32;not null;default:public"`

	// --- Noise reduction config (stored as JSON) ---
	// AggregationConfig holds rule-based aggregation settings.
	AggregationConfig string `json:"aggregation_config" gorm:"type:json"`
	// FlappingConfig holds flapping detection parameters.
	FlappingConfig string `json:"flapping_config" gorm:"type:json"`

	// --- Auto-close config ---
	AutoCloseEnabled bool   `json:"auto_close_enabled" gorm:"default:false"`
	AutoCloseOrigin  string `json:"auto_close_origin" gorm:"size:32;default:triggered"` // triggered | last_alert
	AutoCloseMinutes int    `json:"auto_close_minutes" gorm:"default:0"`
	// FollowAlertClose: when all related alerts recover, auto-close the incident.
	FollowAlertClose bool `json:"follow_alert_close"`

	// --- Metrics (denormalized for list display) ---
	ActiveIncidentCount int `json:"active_incident_count" gorm:"default:0"`

	// --- Sorting ---
	SortOrder int `json:"sort_order" gorm:"default:0;index"`
}

func (Channel) TableName() string {
	return "channels"
}

// ChannelNoiseAggregation defines the aggregation rule config.
type ChannelNoiseAggregation struct {
	Enabled bool `json:"enabled"`
	// Mode: "unified" (all alerts same dimensions) or "fine_grained" (conditional branches)
	Mode string `json:"mode"` // unified | fine_grained
	// Dimensions for unified mode (max 5 label keys)
	Dimensions []string `json:"dimensions,omitempty"`
	// Branches for fine-grained mode (max 100)
	Branches []AggregationBranch `json:"branches,omitempty"`
	// DefaultDimensions: fallback when no branch matches
	DefaultDimensions []string `json:"default_dimensions,omitempty"`
	// Window config
	WindowEnabled bool   `json:"window_enabled"`
	WindowOrigin  string `json:"window_origin"` // triggered | alert_merged
	WindowMinutes int    `json:"window_minutes"`
	// Storm warning thresholds (max 5, range 2-10000)
	StormThresholds []int `json:"storm_thresholds,omitempty"`
	// StrictMode: empty label values treated as different (true) or same (false)
	StrictMode bool `json:"strict_mode"`
}

// AggregationBranch is a conditional aggregation rule.
type AggregationBranch struct {
	Conditions []FilterCondition `json:"conditions"`
	Dimensions []string          `json:"dimensions"`
}

// ChannelFlappingConfig defines flapping detection settings.
type ChannelFlappingConfig struct {
	// Mode: "off" | "notify_only" | "notify_then_silence"
	Mode string `json:"mode"`
	// MaxChanges: state change count to trigger flapping (2-100)
	MaxChanges int `json:"max_changes"`
	// WindowMinutes: observation window (1-1440)
	WindowMinutes int `json:"window_minutes"`
	// MuteMinutes: silence duration after flapping detected (30-1440, only for notify_then_silence)
	MuteMinutes int `json:"mute_minutes"`
}

// FilterCondition is a reusable condition matcher used across
// routing rules, mute rules, escalation policies, etc.
type FilterCondition struct {
	Field    string `json:"field"`    // severity, title, description, labels.xxx
	Operator string `json:"operator"` // eq, ne, contains, not_contains, regex, in, not_in
	Value    string `json:"value"`
}

// ChannelExclusionRule filters events before they become alerts.
type ChannelExclusionRule struct {
	BaseModel
	ChannelID   uint   `json:"channel_id" gorm:"index;not null"`
	Name        string `json:"name" gorm:"size:128;not null"`
	Description string `json:"description" gorm:"size:512"`
	Conditions  string `json:"conditions" gorm:"type:json"` // []FilterCondition JSON
	IsEnabled   bool   `json:"is_enabled"`
	Priority    int    `json:"priority" gorm:"default:0;index"`
}

func (ChannelExclusionRule) TableName() string {
	return "channel_exclusion_rules"
}

// ChannelStar represents a user's favorite/starred channel.
type ChannelStar struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint      `json:"user_id" gorm:"uniqueIndex:idx_user_channel;not null"`
	ChannelID uint      `json:"channel_id" gorm:"uniqueIndex:idx_user_channel;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (ChannelStar) TableName() string {
	return "channel_stars"
}
