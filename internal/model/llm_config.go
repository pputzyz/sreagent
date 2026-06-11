package model

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// LLMConfig stores a named LLM provider configuration (API key, endpoint,
// model name, etc.). API keys are encrypted at rest with AES-256-GCM.
type LLMConfig struct {
	gorm.Model

	Name        string `json:"name" gorm:"size:128;not null;uniqueIndex"`
	Provider    string `json:"provider" gorm:"size:32;not null"` // openai, azure, ollama, anthropic, custom
	APIURL      string `json:"api_url" gorm:"size:512"`
	APIKey      string `json:"-" gorm:"size:512"` // AES-256-GCM encrypted
	ModelName   string `json:"model" gorm:"size:128;column:model"`
	ExtraConfig string `json:"extra_config" gorm:"type:text"` // JSON
	Enabled     bool   `json:"enabled" gorm:"default:true"`
	IsDefault   bool   `json:"is_default" gorm:"default:false"`
	Description string `json:"description" gorm:"size:512"`
	CreatedBy   uint   `json:"created_by" gorm:"not null;default:0"`
	UpdatedBy   uint   `json:"updated_by" gorm:"not null;default:0"`
}

func (LLMConfig) TableName() string { return "llm_configs" }

// MaskAPIKey returns a masked version of the API key showing only the first
// and last 4 characters. Returns empty string if the key is too short.
func (c *LLMConfig) MaskAPIKey() string {
	if c.APIKey == "" {
		return ""
	}
	// Don't mask an already-masked value
	if IsMaskedAPIKey(c.APIKey) {
		return c.APIKey
	}
	// Don't mask encrypted values — they should be decrypted first
	if strings.HasPrefix(c.APIKey, "enc:") {
		return "****"
	}
	key := c.APIKey
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

// IsMaskedAPIKey returns true if the key looks like a masked value
// (contains only asterisks in the middle, e.g. "sk-a****xyz").
func IsMaskedAPIKey(key string) bool {
	if key == "" {
		return false
	}
	// Simple heuristic: masked keys contain * but no enc: prefix
	if strings.HasPrefix(key, "enc:") {
		return false
	}
	if strings.Contains(key, "*") {
		return true
	}
	return false
}

// Verify validates required fields.
func (c *LLMConfig) Verify() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	switch c.Provider {
	case "openai", "azure", "ollama", "anthropic", "custom":
		// valid
	default:
		return fmt.Errorf("invalid provider: %s (must be openai, azure, ollama, anthropic, or custom)", c.Provider)
	}
	return nil
}
