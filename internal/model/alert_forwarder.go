package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ForwarderDirection defines the direction of alert forwarding.
type ForwarderDirection string

const (
	ForwarderDirectionInbound      ForwarderDirection = "inbound"
	ForwarderDirectionOutbound     ForwarderDirection = "outbound"
	ForwarderDirectionBidirectional ForwarderDirection = "bidirectional"
)

func (d ForwarderDirection) IsValid() bool {
	switch d {
	case ForwarderDirectionInbound, ForwarderDirectionOutbound, ForwarderDirectionBidirectional:
		return true
	}
	return false
}

// ForwarderSourceFormat defines the format of incoming alert payloads.
type ForwarderSourceFormat string

const (
	SourceFormatAlertmanager ForwarderSourceFormat = "alertmanager"
	SourceFormatGrafana      ForwarderSourceFormat = "grafana"
	SourceFormatPrometheus   ForwarderSourceFormat = "prometheus"
	SourceFormatGeneric      ForwarderSourceFormat = "generic"
)

func (f ForwarderSourceFormat) IsValid() bool {
	switch f {
	case SourceFormatAlertmanager, SourceFormatGrafana, SourceFormatPrometheus, SourceFormatGeneric:
		return true
	}
	return false
}

// ForwarderAuthType defines the authentication type for inbound endpoints.
type ForwarderAuthType string

const (
	ForwarderAuthNone   ForwarderAuthType = "none"
	ForwarderAuthBearer ForwarderAuthType = "bearer"
	ForwarderAuthBasic  ForwarderAuthType = "basic"
	ForwarderAuthHMAC   ForwarderAuthType = "hmac"
)

func (a ForwarderAuthType) IsValid() bool {
	switch a {
	case ForwarderAuthNone, ForwarderAuthBearer, ForwarderAuthBasic, ForwarderAuthHMAC:
		return true
	}
	return false
}

// SeverityMappingDirection defines which direction the severity mapping applies to.
type SeverityMappingDirection string

const (
	SeverityMappingDirInbound  SeverityMappingDirection = "inbound"
	SeverityMappingDirOutbound SeverityMappingDirection = "outbound"
	SeverityMappingDirBoth     SeverityMappingDirection = "both"
)

func (d SeverityMappingDirection) IsValid() bool {
	switch d {
	case SeverityMappingDirInbound, SeverityMappingDirOutbound, SeverityMappingDirBoth:
		return true
	}
	return false
}

// AlertForwarder is the main model for alert forwarding configuration.
type AlertForwarder struct {
	ID          uint               `json:"id" gorm:"primaryKey"`
	Name        string             `json:"name" gorm:"size:128;not null;uniqueIndex"`
	Description string             `json:"description" gorm:"size:512"`
	Enabled     bool               `json:"enabled" gorm:"default:true;index"`
	Direction   ForwarderDirection `json:"direction" gorm:"size:32;not null"`
	Priority    int                `json:"priority" gorm:"default:0;index"`

	// Inbound configuration
	InboundConfig  *InboundConfig  `json:"inbound_config,omitempty" gorm:"type:json;column:inbound_config"`
	OutboundConfig *OutboundConfig `json:"outbound_config,omitempty" gorm:"type:json;column:outbound_config"`

	// Severity mapping
	SeverityMapping *SeverityMappingConfig `json:"severity_mapping,omitempty" gorm:"type:json;column:severity_mapping"`

	// Platform capabilities
	PlatformCapabilities *PlatformCapabilitiesConfig `json:"platform_capabilities,omitempty" gorm:"type:json;column:platform_capabilities"`

	// Match conditions
	MatchLabels JSONLabels `json:"match_labels,omitempty" gorm:"type:json;column:match_labels"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AlertForwarder) TableName() string {
	return "alert_forwarders"
}

// InboundConfig holds configuration for inbound alert receiving.
type InboundConfig struct {
	SourceFormat ForwarderSourceFormat `json:"source_format"`
	Path         string               `json:"path,omitempty"` // Auto-generated
	AuthType     ForwarderAuthType    `json:"auth_type"`
	AuthConfig   *AuthConfig          `json:"auth_config,omitempty"`
}

// AuthConfig holds authentication credentials.
type AuthConfig struct {
	// Bearer token
	Token string `json:"token,omitempty"`
	// Basic auth
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	// HMAC
	HMACSecret  string `json:"hmac_secret,omitempty"`
	HMACHeader  string `json:"hmac_header,omitempty"`
	HMACAlgorithm string `json:"hmac_algorithm,omitempty"` // sha256, sha1
}

// OutboundConfig holds configuration for outbound alert forwarding.
type OutboundConfig struct {
	TargetMediaID *uint             `json:"target_media_id,omitempty"` // Reference to NotifyMedia
	TargetURL     string            `json:"target_url,omitempty"`      // Direct URL
	Method        string            `json:"method,omitempty"`          // HTTP method, default POST
	Headers       map[string]string `json:"headers,omitempty"`
	BodyTemplate  string            `json:"body_template,omitempty"` // Go template
	Timeout       int               `json:"timeout,omitempty"`       // milliseconds
	RetryTimes    int               `json:"retry_times,omitempty"`
	RetryInterval int               `json:"retry_interval,omitempty"` // milliseconds
}

// SeverityMappingConfig holds severity mapping configuration.
type SeverityMappingConfig struct {
	Enabled         bool                      `json:"enabled"`
	Direction       SeverityMappingDirection  `json:"direction"`
	Mapping         map[string]string         `json:"mapping"`          // e.g., {"critical": "P0", "warning": "P2"}
	DefaultSeverity string                    `json:"default_severity"` // Fallback when no mapping found
}

// PlatformCapabilitiesConfig controls which platform features to engage during forwarding.
type PlatformCapabilitiesConfig struct {
	EnableEscalation   bool   `json:"enable_escalation"`
	EnableMute         bool   `json:"enable_mute"`
	EnableInhibition   bool   `json:"enable_inhibition"`
	EnableNotification bool   `json:"enable_notification"`
	EnableAIAnalysis   bool   `json:"enable_ai_analysis"`
	PipelineID         *uint  `json:"pipeline_id,omitempty"` // Optional: custom event pipeline
}

// Value/Scan implementations for GORM JSON columns.

func (c InboundConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *InboundConfig) Scan(src interface{}) error {
	return jsonScan(src, c)
}

func (c OutboundConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *OutboundConfig) Scan(src interface{}) error {
	return jsonScan(src, c)
}

func (c SeverityMappingConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *SeverityMappingConfig) Scan(src interface{}) error {
	return jsonScan(src, c)
}

func (c PlatformCapabilitiesConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *PlatformCapabilitiesConfig) Scan(src interface{}) error {
	return jsonScan(src, c)
}

func jsonScan(src interface{}, dst interface{}) error {
	if src == nil {
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported type for JSON scan: %T", src)
	}
	return json.Unmarshal(data, dst)
}
