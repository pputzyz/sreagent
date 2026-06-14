package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ForwarderDirection defines the direction of alert forwarding.
type ForwarderDirection string

const (
	ForwarderDirectionInbound       ForwarderDirection = "inbound"
	ForwarderDirectionOutbound      ForwarderDirection = "outbound"
	ForwarderDirectionBidirectional ForwarderDirection = "bidirectional"
)

func (d ForwarderDirection) IsValid() bool {
	switch d {
	case ForwarderDirectionInbound, ForwarderDirectionOutbound, ForwarderDirectionBidirectional:
		return true
	}
	return false
}

// InboundMode defines how inbound alerts are processed.
type InboundMode string

const (
	// InboundModeIntegrate: alert enters platform core lifecycle (like engine-generated alerts).
	// Goes through: create AlertEvent → inhibition → mute → noise reduction → notification → escalation.
	InboundModeIntegrate InboundMode = "integrate"
	// InboundModeProxy: alert is forwarded to an external target without entering platform lifecycle.
	// Only applies severity mapping, then forwards to configured outbound target.
	InboundModeProxy InboundMode = "proxy"
)

func (m InboundMode) IsValid() bool {
	switch m {
	case InboundModeIntegrate, InboundModeProxy:
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

// AlertForwarder is the main model for alert forwarding configuration.
type AlertForwarder struct {
	ID          uint               `json:"id" gorm:"primaryKey"`
	Name        string             `json:"name" gorm:"size:128;not null;uniqueIndex"`
	Description string             `json:"description" gorm:"size:512"`
	Enabled     bool               `json:"enabled" gorm:"index"` // create handler defaults to true; DB column keeps DEFAULT 1 for seeds
	Direction   ForwarderDirection `json:"direction" gorm:"size:32;not null"`
	Priority    int                `json:"priority" gorm:"default:0;index"`

	// Inbound configuration (for inbound/bidirectional)
	InboundConfig  *InboundConfig  `json:"inbound_config,omitempty" gorm:"type:json;column:inbound_config"`
	OutboundConfig *OutboundConfig `json:"outbound_config,omitempty" gorm:"type:json;column:outbound_config"`

	// Severity mapping (independent for inbound and outbound)
	InboundSeverityMapping  *SeverityMappingConfig `json:"inbound_severity_mapping,omitempty" gorm:"type:json;column:inbound_severity_mapping"`
	OutboundSeverityMapping *SeverityMappingConfig `json:"outbound_severity_mapping,omitempty" gorm:"type:json;column:outbound_severity_mapping"`

	// Platform capabilities (only for integrate mode)
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
	Mode         InboundMode           `json:"mode"` // "integrate" or "proxy"
	AuthType     ForwarderAuthType     `json:"auth_type"`
	AuthConfig   *AuthConfig           `json:"auth_config,omitempty"`
	// ProxyTarget is used when mode=proxy: forward to this target after inbound processing.
	ProxyTarget *OutboundConfig `json:"proxy_target,omitempty"`
}

// AuthConfig holds authentication credentials.
type AuthConfig struct {
	// Bearer token
	Token string `json:"token,omitempty"`
	// Basic auth
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	// HMAC
	HMACSecret    string `json:"hmac_secret,omitempty"`
	HMACHeader    string `json:"hmac_header,omitempty"`
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
	Enabled         bool              `json:"enabled"`
	Mapping         map[string]string `json:"mapping"`          // e.g., {"critical": "P0", "warning": "P2"}
	DefaultSeverity string            `json:"default_severity"` // Fallback when no mapping found
}

// PlatformCapabilitiesConfig controls which platform features to engage during forwarding.
// Only applies to integrate mode.
type PlatformCapabilitiesConfig struct {
	EnableEscalation   bool  `json:"enable_escalation"`
	EnableMute         bool  `json:"enable_mute"`
	EnableInhibition   bool  `json:"enable_inhibition"`
	EnableNotification bool  `json:"enable_notification"`
	EnableAIAnalysis   bool  `json:"enable_ai_analysis"`
	PipelineID         *uint `json:"pipeline_id,omitempty"` // Optional: custom event pipeline
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

// ApplySeverityMapping applies severity mapping to a given severity string.
// Returns the mapped severity and whether mapping was applied.
func (c *SeverityMappingConfig) ApplySeverityMapping(severity string) (string, bool) {
	if c == nil || !c.Enabled {
		return severity, false
	}
	if mapped, ok := c.Mapping[severity]; ok {
		return mapped, true
	}
	if c.DefaultSeverity != "" {
		return c.DefaultSeverity, true
	}
	return severity, false
}

// sensitiveHeaderKeys are HTTP header keys that typically carry secrets.
var sensitiveHeaderKeys = map[string]bool{
	"authorization": true,
	"x-api-key":     true,
	"x-auth-token":  true,
	"cookie":        true,
	"set-cookie":    true,
}

// SanitizeForResponse returns a copy of the forwarder with sensitive fields masked.
func (f *AlertForwarder) SanitizeForResponse() *AlertForwarder {
	out := *f // shallow copy
	if out.InboundConfig != nil {
		ic := *out.InboundConfig // deep copy InboundConfig
		if ic.AuthConfig != nil {
			ac := *ic.AuthConfig
			if ac.Token != "" {
				ac.Token = "***"
			}
			if ac.Password != "" {
				ac.Password = "***"
			}
			if ac.HMACSecret != "" {
				ac.HMACSecret = "***"
			}
			ic.AuthConfig = &ac
		}
		// Mask ProxyTarget headers
		if ic.ProxyTarget != nil {
			pt := *ic.ProxyTarget
			pt.Headers = maskSensitiveHeaders(pt.Headers)
			ic.ProxyTarget = &pt
		}
		out.InboundConfig = &ic
	}
	// Mask OutboundConfig headers
	if out.OutboundConfig != nil {
		oc := *out.OutboundConfig
		oc.Headers = maskSensitiveHeaders(oc.Headers)
		out.OutboundConfig = &oc
	}
	return &out
}

// maskSensitiveHeaders returns a copy of headers with sensitive values masked.
func maskSensitiveHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return headers
	}
	masked := make(map[string]string, len(headers))
	for k, v := range headers {
		if sensitiveHeaderKeys[strings.ToLower(k)] && v != "" {
			masked[k] = "***"
		} else {
			masked[k] = v
		}
	}
	return masked
}
