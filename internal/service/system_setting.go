package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/pkg/crypto"
	"github.com/sreagent/sreagent/internal/repository"
)

// ---- typed config structs (replaces config.AIConfig / config.LarkConfig) ----

// AIConfig holds AI/LLM integration configuration stored in the DB.
// Deprecated: Use AIProviderConfig / AIProvidersConfig for multi-provider support.
// Retained for backward compatibility with existing callers.
type AIConfig struct {
	Provider        string  `json:"provider"` // openai, azure, ollama, custom, anthropic
	APIKey          string  `json:"api_key"`
	BaseURL         string  `json:"base_url"`
	Model           string  `json:"model"`
	Enabled         bool    `json:"enabled"`
	Temperature     float64 `json:"temperature"`       // 0.0-2.0, default 0.3
	MaxTokens       int     `json:"max_tokens"`        // default 1024
	TopP            float64 `json:"top_p"`             // 0.0-1.0, default 1.0
	SystemPrompt    string  `json:"system_prompt"`     // custom system prompt prefix
	RetryMax        int     `json:"retry_max"`         // LLM call retries, default 2
	ContextMaxChars int     `json:"context_max_chars"` // context text char limit, default 8000
}

// AIProviderConfig describes a single named AI provider configuration.
type AIProviderConfig struct {
	Key             string  `json:"key"`      // unique identifier, e.g. "openai-main"
	Provider        string  `json:"provider"` // openai, azure, ollama, custom, anthropic
	APIKey          string  `json:"api_key"`
	BaseURL         string  `json:"base_url"`
	Model           string  `json:"model"`
	Enabled         bool    `json:"enabled"`
	IsDefault       bool    `json:"is_default"`
	Temperature     float64 `json:"temperature,omitempty"`
	MaxTokens       int     `json:"max_tokens,omitempty"`
	TopP            float64 `json:"top_p,omitempty"`
	SystemPrompt    string  `json:"system_prompt,omitempty"`
	RetryMax        int     `json:"retry_max,omitempty"`
	ContextMaxChars int     `json:"context_max_chars,omitempty"`
}

// AIProvidersConfig holds the multi-provider AI configuration stored as a single JSON blob.
type AIProvidersConfig struct {
	DefaultProvider string             `json:"default_provider"`
	Providers       []AIProviderConfig `json:"providers"`
}

// AIGlobalConfig holds platform-wide AI settings (Tab 3 in unified AI settings page).
type AIGlobalConfig struct {
	RetryMax           int     `json:"retry_max"`            // default 2
	ContextMaxChars    int     `json:"context_max_chars"`    // default 8000
	DefaultTemperature float64 `json:"default_temperature"`  // default 0.3
	DefaultMaxTokens   int     `json:"default_max_tokens"`   // default 1024
	MonthlyTokenBudget int64   `json:"monthly_token_budget"` // 0 = unlimited
	DataMaskingEnabled bool    `json:"data_masking_enabled"` // default false
}

// LarkConfig holds Lark/Feishu bot configuration stored in the DB.
type LarkConfig struct {
	AppID             string `json:"app_id"`
	AppSecret         string `json:"app_secret"`
	DefaultWebhook    string `json:"default_webhook"`
	VerificationToken string `json:"verification_token"`
	EncryptKey        string `json:"encrypt_key"`
	BotEnabled        bool   `json:"bot_enabled"`

	// Section 2: region and connection
	Domain              string `json:"domain"`                // "feishu" | "larksuite", default "larksuite"
	ConnectionMode      string `json:"connection_mode"`       // "websocket" | "http_callback", default "websocket"
	CardInteractionMode string `json:"card_interaction_mode"` // "callback_ws" | "callback_http" | "open_url", default "open_url"
	CardSchemaVersion   string `json:"card_schema_version"`   // "v2" | "v1", default "v2"

	// Section 3: message behavior
	ResolveStrategy           string `json:"resolve_strategy"`              // "update" | "delete" | "none", default "update"
	UpdateOnStateChange       bool   `json:"update_on_state_change"`        // default true
	DeleteOnlyInBusinessHours bool   `json:"delete_only_in_business_hours"` // default false
	BusinessHoursStart        string `json:"business_hours_start"`          // "09:00"
	BusinessHoursEnd          string `json:"business_hours_end"`            // "18:00"

	// Section 4: interaction capabilities
	CommandsEnabled        bool `json:"commands_enabled"`         // default true
	NaturalLanguageEnabled bool `json:"natural_language_enabled"` // default false
	DebugMode              bool `json:"debug_mode"`               // default false
}

// SecurityConfig holds security-related settings stored in the DB.
type SecurityConfig struct {
	JWTExpireSeconds int `json:"jwt_expire_seconds"` // default from config file
}

const (
	groupAI       = "ai"
	groupLark     = "lark"
	groupOIDC     = "oidc"
	groupOAuth2   = "oauth2"
	groupLDAP     = "ldap"
	groupSMTP     = "smtp"
	groupSecurity = "security"
	groupSiteInfo = "site_info"

	// cacheTTL is how long a cached config entry is considered fresh.
	cacheTTL = 30 * time.Second
)

// sensitiveKeys lists the setting keys that must be encrypted at rest.
// Key format: "group.key_name".
var sensitiveKeys = map[string]bool{
	"ai.api_key":              true,
	"ai.providers":            true,
	"lark.app_secret":         true,
	"lark.verification_token": true,
	"lark.encrypt_key":        true,
	"oidc.client_secret":      true,
	"oauth2.client_secret":    true,
	"ldap.bind_password":      true,
	"smtp.password":           true,
}

// SMTPConfig holds global SMTP configuration for system-wide email delivery.
// Used by the escalation executor to send personal email notifications.
type SMTPConfig struct {
	SMTPHost string `json:"smtp_host"`
	SMTPPort int    `json:"smtp_port"`
	SMTPTLS  bool   `json:"smtp_tls"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	Enabled  bool   `json:"enabled"`
}

// OIDCConfigDB holds OIDC/SSO integration configuration stored in the DB.
// This mirrors config.OIDCConfig but is persisted in the system_settings table,
// allowing admins to update it via the UI without redeploying.
type OIDCConfigDB struct {
	Enabled       bool   `json:"enabled"`
	IssuerURL     string `json:"issuer_url"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	RedirectURL   string `json:"redirect_url"`
	Scopes        string `json:"scopes"`         // comma-separated, e.g. "openid,profile,email"
	UsernameClaim string `json:"username_claim"` // default "preferred_username"
	EmailClaim    string `json:"email_claim"`    // default "email"
	RoleClaim     string `json:"role_claim"`     // default "realm_access.roles"
	RoleMapping   string `json:"role_mapping"`   // JSON object string, e.g. {"sre-admin":"admin"}
	DefaultRole   string `json:"default_role"`   // default "viewer"
	AutoProvision bool   `json:"auto_provision"` // default true
}
type cachedConfig[T any] struct {
	value     T
	expiresAt time.Time
}

func (c *cachedConfig[T]) valid() bool {
	return !c.expiresAt.IsZero() && time.Now().Before(c.expiresAt)
}

// SystemSettingService manages platform-level key-value settings stored in DB.
// AI and Lark configs are cached in memory for cacheTTL (30 s) to avoid a DB
// round-trip on every LLM/Lark call. Writes invalidate the cache immediately.
//
// Sensitive fields (api_key, app_secret, etc.) are encrypted with AES-256-GCM
// using the master key loaded from the SREAGENT_SECRET_KEY environment variable
// (32-byte hex string). If the env var is absent, values are stored plaintext
// and a warning is logged at startup.
type SystemSettingService struct {
	repo   *repository.SystemSettingRepository
	logger *zap.Logger

	aiMu    sync.RWMutex
	aiCache cachedConfig[AIConfig]

	providersMu    sync.RWMutex
	providersCache cachedConfig[AIProvidersConfig]

	larkMu    sync.RWMutex
	larkCache cachedConfig[LarkConfig]

	oidcMu    sync.RWMutex
	oidcCache cachedConfig[OIDCConfigDB]

	smtpMu    sync.RWMutex
	smtpCache cachedConfig[SMTPConfig]
}

// NewSystemSettingService creates a new SystemSettingService.
// Encryption is delegated to the shared crypto package (reads SREAGENT_SECRET_KEY).
func NewSystemSettingService(repo *repository.SystemSettingRepository, logger *zap.Logger) *SystemSettingService {
	return &SystemSettingService{repo: repo, logger: logger}
}

// encryptValue encrypts a plaintext string using AES-256-GCM (via shared crypto package).
func (s *SystemSettingService) encryptValue(plaintext string) (string, error) {
	return crypto.EncryptString(plaintext)
}

// decryptValue decrypts a value encrypted by encryptValue.
// Values not starting with "enc:" are returned as-is (backward compatible).
func (s *SystemSettingService) decryptValue(value string) (string, error) {
	return crypto.DecryptString(value)
}

// setEncrypted encrypts a value for a given group+key if it is sensitive.
func (s *SystemSettingService) setEncrypted(group, key, value string) (string, error) {
	if sensitiveKeys[group+"."+key] {
		return s.encryptValue(value)
	}
	return value, nil
}

// getDecrypted decrypts a value for a given group+key if it is sensitive.
func (s *SystemSettingService) getDecrypted(group, key, value string) string {
	if !sensitiveKeys[group+"."+key] {
		return value
	}
	plain, err := s.decryptValue(value)
	if err != nil {
		s.logger.Error("failed to decrypt sensitive setting",
			zap.String("group", group),
			zap.String("key", key),
			zap.Error(err),
		)
		return ""
	}
	return plain
}

// ---- AI config ---------------------------------------------------------------

// GetAIConfig loads the AI configuration from cache or DB.
// Cache TTL is cacheTTL (30 s); writes invalidate the cache immediately.
// For multi-provider setups, this returns the default provider's config.
func (s *SystemSettingService) GetAIConfig(ctx context.Context) (AIConfig, error) {
	// Fast path: read from cache.
	s.aiMu.RLock()
	if s.aiCache.valid() {
		cfg := s.aiCache.value
		s.aiMu.RUnlock()
		return cfg, nil
	}
	s.aiMu.RUnlock()

	// Try multi-provider config first.
	providersCfg, err := s.GetProvidersConfig(ctx)
	if err == nil && len(providersCfg.Providers) > 0 {
		provider := s.findProvider(providersCfg, providersCfg.DefaultProvider)
		if provider != nil {
			cfg := AIConfig{
				Provider: provider.Provider,
				APIKey:   provider.APIKey,
				BaseURL:  provider.BaseURL,
				Model:    provider.Model,
				Enabled:  provider.Enabled,
			}
			s.aiMu.Lock()
			s.aiCache = cachedConfig[AIConfig]{value: cfg, expiresAt: time.Now().Add(cacheTTL)}
			s.aiMu.Unlock()
			return cfg, nil
		}
	}

	// Fallback: legacy single-provider keys.
	kv, err := s.repo.ListByGroup(ctx, groupAI)
	if err != nil {
		return AIConfig{}, err
	}
	cfg := AIConfig{
		Provider:        strDef(kv["provider"], "openai"),
		APIKey:          s.getDecrypted(groupAI, "api_key", kv["api_key"]),
		BaseURL:         strDef(kv["base_url"], "https://api.openai.com/v1"),
		Model:           strDef(kv["model"], "gpt-4o"),
		Enabled:         parseBool(kv["enabled"]),
		Temperature:     parseFloatDef(kv["temperature"], 0.3),
		MaxTokens:       parseIntDef(kv["max_tokens"], 1024),
		SystemPrompt:    kv["system_prompt"],
		RetryMax:        parseIntDef(kv["retry_max"], 2),
		ContextMaxChars: parseIntDef(kv["context_max_chars"], 8000),
	}

	s.aiMu.Lock()
	s.aiCache = cachedConfig[AIConfig]{value: cfg, expiresAt: time.Now().Add(cacheTTL)}
	s.aiMu.Unlock()

	return cfg, nil
}

// SaveAIConfig persists all AI configuration keys to the DB and invalidates cache.
// Empty api_key means "do not overwrite the existing key".
func (s *SystemSettingService) SaveAIConfig(ctx context.Context, cfg AIConfig) error {
	kv := map[string]string{
		"provider":          cfg.Provider,
		"base_url":          cfg.BaseURL,
		"model":             cfg.Model,
		"enabled":           strconv.FormatBool(cfg.Enabled),
		"temperature":       strconv.FormatFloat(cfg.Temperature, 'f', -1, 64),
		"max_tokens":        strconv.Itoa(cfg.MaxTokens),
		"system_prompt":     cfg.SystemPrompt,
		"retry_max":         strconv.Itoa(cfg.RetryMax),
		"context_max_chars": strconv.Itoa(cfg.ContextMaxChars),
	}
	// Only save api_key when caller provided a non-empty value (avoids clearing
	// a stored key when the frontend sends back the masked placeholder).
	if cfg.APIKey != "" {
		enc, err := s.setEncrypted(groupAI, "api_key", cfg.APIKey)
		if err != nil {
			s.logger.Error("failed to encrypt ai.api_key", zap.Error(err))
			return err
		}
		kv["api_key"] = enc
	}
	if err := s.repo.SetGroup(ctx, groupAI, kv); err != nil {
		return err
	}

	// Also update the default provider in multi-provider config if it exists.
	providersCfg, err := s.GetProvidersConfig(ctx)
	if err == nil && len(providersCfg.Providers) > 0 {
		defaultKey := providersCfg.DefaultProvider
		if defaultKey == "" {
			defaultKey = providersCfg.Providers[0].Key
		}
		changed := false
		for i, p := range providersCfg.Providers {
			if p.Key == defaultKey {
				if cfg.Provider != "" {
					providersCfg.Providers[i].Provider = cfg.Provider
					changed = true
				}
				if cfg.BaseURL != "" {
					providersCfg.Providers[i].BaseURL = cfg.BaseURL
					changed = true
				}
				if cfg.Model != "" {
					providersCfg.Providers[i].Model = cfg.Model
					changed = true
				}
				if cfg.APIKey != "" {
					providersCfg.Providers[i].APIKey = cfg.APIKey
					changed = true
				}
				// Only update enabled/temperature/max_tokens if they were explicitly provided (non-zero)
				if cfg.Temperature > 0 {
					providersCfg.Providers[i].Temperature = cfg.Temperature
					changed = true
				}
				if cfg.MaxTokens > 0 {
					providersCfg.Providers[i].MaxTokens = cfg.MaxTokens
					changed = true
				}
				break
			}
		}
		if changed {
			if saveErr := s.SaveProvidersConfig(ctx, providersCfg); saveErr != nil {
				s.logger.Warn("failed to sync providers config on SaveAIConfig", zap.Error(saveErr))
			}
		}
	}

	// Invalidate cache so the next read fetches fresh data.
	s.aiMu.Lock()
	s.aiCache = cachedConfig[AIConfig]{}
	s.providersMu.Lock()
	s.providersCache = cachedConfig[AIProvidersConfig]{}
	s.providersMu.Unlock()
	s.aiMu.Unlock()
	return nil
}

// ---- AI Providers config (multi-provider) ------------------------------------

// GetProvidersConfig loads the multi-provider AI configuration from cache or DB.
// API keys within providers are decrypted on load.
func (s *SystemSettingService) GetProvidersConfig(ctx context.Context) (AIProvidersConfig, error) {
	// Fast path: read from cache.
	s.providersMu.RLock()
	if s.providersCache.valid() {
		cfg := s.providersCache.value
		s.providersMu.RUnlock()
		return cfg, nil
	}
	s.providersMu.RUnlock()

	// Slow path: load from DB.
	kv, err := s.repo.ListByGroup(ctx, groupAI)
	if err != nil {
		return AIProvidersConfig{}, err
	}

	raw, ok := kv["providers"]
	if !ok || raw == "" {
		// No providers configured yet — return empty config.
		return AIProvidersConfig{DefaultProvider: "", Providers: nil}, nil
	}

	// Decrypt the stored JSON blob.
	decrypted := s.getDecrypted(groupAI, "providers", raw)

	var cfg AIProvidersConfig
	if err := json.Unmarshal([]byte(decrypted), &cfg); err != nil {
		s.logger.Error("failed to parse ai.providers JSON", zap.Error(err))
		return AIProvidersConfig{}, fmt.Errorf("invalid ai.providers config: %w", err)
	}

	// Cache the result.
	s.providersMu.Lock()
	s.providersCache = cachedConfig[AIProvidersConfig]{value: cfg, expiresAt: time.Now().Add(cacheTTL)}
	s.providersMu.Unlock()

	return cfg, nil
}

// isMaskedKey returns true if the key looks like a masked value (e.g., "abcd****efgh").
func isMaskedKey(key string) bool {
	if len(key) < 8 {
		return false
	}
	// Check for the mask pattern: at least 4 chars + "****" + at least 4 chars.
	idx := len(key) - 4
	if idx < 4 {
		return false
	}
	return key[4:idx] == "****"
}

// SaveProvidersConfig persists the multi-provider AI configuration to DB.
// API keys within providers are encrypted before storage.
// H3: Detects masked API keys and preserves the original values from DB.
func (s *SystemSettingService) SaveProvidersConfig(ctx context.Context, cfg AIProvidersConfig) error {
	// Validate at least one provider is enabled.
	hasEnabled := false
	for _, p := range cfg.Providers {
		if p.Enabled {
			hasEnabled = true
			break
		}
	}
	if !hasEnabled {
		return fmt.Errorf("at least one AI provider must be enabled")
	}

	// H3: Load existing config to preserve original keys when frontend sends masked values.
	existing, err := s.GetProvidersConfig(ctx)
	if err == nil && len(existing.Providers) > 0 {
		existingMap := make(map[string]string, len(existing.Providers))
		for _, p := range existing.Providers {
			existingMap[p.Key] = p.APIKey
		}
		for i := range cfg.Providers {
			if isMaskedKey(cfg.Providers[i].APIKey) {
				if orig, ok := existingMap[cfg.Providers[i].Key]; ok {
					cfg.Providers[i].APIKey = orig
				}
			}
		}
	}

	// Encrypt the entire JSON blob (which contains api_key fields).
	jsonBytes, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal providers config: %w", err)
	}

	enc, err := s.setEncrypted(groupAI, "providers", string(jsonBytes))
	if err != nil {
		s.logger.Error("failed to encrypt ai.providers", zap.Error(err))
		return err
	}

	kv := map[string]string{
		"providers": enc,
	}
	if err := s.repo.SetGroup(ctx, groupAI, kv); err != nil {
		return err
	}

	// Invalidate both caches.
	s.providersMu.Lock()
	s.providersCache = cachedConfig[AIProvidersConfig]{}
	s.providersMu.Unlock()

	s.aiMu.Lock()
	s.aiCache = cachedConfig[AIConfig]{}
	s.aiMu.Unlock()

	return nil
}

// GetProviderConfig loads a specific provider by key, or the default provider if key is empty.
func (s *SystemSettingService) GetProviderConfig(ctx context.Context, providerKey string) (AIProviderConfig, error) {
	cfg, err := s.GetProvidersConfig(ctx)
	if err != nil {
		return AIProviderConfig{}, err
	}

	key := providerKey
	if key == "" {
		key = cfg.DefaultProvider
	}

	p := s.findProvider(cfg, key)
	if p == nil {
		return AIProviderConfig{}, fmt.Errorf("AI provider %q not found", key)
	}
	return *p, nil
}

// findProvider returns the provider with the given key, or nil if not found.
func (s *SystemSettingService) findProvider(cfg AIProvidersConfig, key string) *AIProviderConfig {
	for i := range cfg.Providers {
		if cfg.Providers[i].Key == key {
			return &cfg.Providers[i]
		}
	}
	return nil
}

// ---- Lark config -------------------------------------------------------------

// GetLarkConfig loads the Lark bot configuration from cache or DB.
func (s *SystemSettingService) GetLarkConfig(ctx context.Context) (LarkConfig, error) {
	// Fast path: read from cache.
	s.larkMu.RLock()
	if s.larkCache.valid() {
		cfg := s.larkCache.value
		s.larkMu.RUnlock()
		return cfg, nil
	}
	s.larkMu.RUnlock()

	// Slow path: load from DB and repopulate cache.
	kv, err := s.repo.ListByGroup(ctx, groupLark)
	if err != nil {
		return LarkConfig{}, err
	}
	resolveStrategy := kv["resolve_strategy"]
	if resolveStrategy == "" {
		resolveStrategy = "update"
	}
	bhStart := kv["business_hours_start"]
	if bhStart == "" {
		bhStart = "09:00"
	}
	bhEnd := kv["business_hours_end"]
	if bhEnd == "" {
		bhEnd = "18:00"
	}

	domain := kv["domain"]
	if domain == "" {
		domain = "larksuite"
	}
	connMode := kv["connection_mode"]
	if connMode == "" {
		connMode = "websocket"
	}
	cardInterMode := kv["card_interaction_mode"]
	if cardInterMode == "" {
		cardInterMode = "open_url"
	}
	cardSchema := kv["card_schema_version"]
	if cardSchema == "" {
		cardSchema = "v2"
	}

	cfg := LarkConfig{
		AppID:             kv["app_id"],
		AppSecret:         s.getDecrypted(groupLark, "app_secret", kv["app_secret"]),
		DefaultWebhook:    kv["default_webhook"],
		VerificationToken: s.getDecrypted(groupLark, "verification_token", kv["verification_token"]),
		EncryptKey:        s.getDecrypted(groupLark, "encrypt_key", kv["encrypt_key"]),
		BotEnabled:        parseBool(kv["bot_enabled"]),

		Domain:              domain,
		ConnectionMode:      connMode,
		CardInteractionMode: cardInterMode,
		CardSchemaVersion:   cardSchema,

		ResolveStrategy:           resolveStrategy,
		UpdateOnStateChange:       parseBoolDef(kv["update_on_state_change"], true),
		DeleteOnlyInBusinessHours: parseBool(kv["delete_only_in_business_hours"]),
		BusinessHoursStart:        bhStart,
		BusinessHoursEnd:          bhEnd,
		CommandsEnabled:           parseBoolDef(kv["commands_enabled"], true),
		NaturalLanguageEnabled:    parseBool(kv["natural_language_enabled"]),
		DebugMode:                 parseBool(kv["debug_mode"]),
	}

	s.larkMu.Lock()
	s.larkCache = cachedConfig[LarkConfig]{value: cfg, expiresAt: time.Now().Add(cacheTTL)}
	s.larkMu.Unlock()

	return cfg, nil
}

// SaveLarkConfig persists all Lark bot configuration keys to the DB and invalidates cache.
// Empty secret fields are not overwritten (same pattern as AI).
func (s *SystemSettingService) SaveLarkConfig(ctx context.Context, cfg LarkConfig) error {
	kv := map[string]string{
		"app_id":                        cfg.AppID,
		"default_webhook":               cfg.DefaultWebhook,
		"bot_enabled":                   strconv.FormatBool(cfg.BotEnabled),
		"domain":                        cfg.Domain,
		"connection_mode":               cfg.ConnectionMode,
		"card_interaction_mode":         cfg.CardInteractionMode,
		"card_schema_version":           cfg.CardSchemaVersion,
		"resolve_strategy":              cfg.ResolveStrategy,
		"update_on_state_change":        strconv.FormatBool(cfg.UpdateOnStateChange),
		"delete_only_in_business_hours": strconv.FormatBool(cfg.DeleteOnlyInBusinessHours),
		"business_hours_start":          cfg.BusinessHoursStart,
		"business_hours_end":            cfg.BusinessHoursEnd,
		"commands_enabled":              strconv.FormatBool(cfg.CommandsEnabled),
		"natural_language_enabled":      strconv.FormatBool(cfg.NaturalLanguageEnabled),
		"debug_mode":                    strconv.FormatBool(cfg.DebugMode),
	}

	encryptField := func(group, key, value string) (string, error) {
		if value == "" {
			return "", nil
		}
		enc, err := s.setEncrypted(group, key, value)
		if err != nil {
			s.logger.Error("failed to encrypt lark field",
				zap.String("key", key),
				zap.Error(err),
			)
			return "", err
		}
		return enc, nil
	}

	if cfg.AppSecret != "" && cfg.AppSecret != "********" {
		enc, err := encryptField(groupLark, "app_secret", cfg.AppSecret)
		if err != nil {
			return err
		}
		kv["app_secret"] = enc
	}
	if cfg.EncryptKey != "" && cfg.EncryptKey != "********" {
		enc, err := encryptField(groupLark, "encrypt_key", cfg.EncryptKey)
		if err != nil {
			return err
		}
		kv["encrypt_key"] = enc
	}
	if cfg.VerificationToken != "" && cfg.VerificationToken != "********" {
		enc, err := encryptField(groupLark, "verification_token", cfg.VerificationToken)
		if err != nil {
			return err
		}
		kv["verification_token"] = enc
	}

	if err := s.repo.SetGroup(ctx, groupLark, kv); err != nil {
		return err
	}
	// Invalidate cache so the next read fetches fresh data.
	s.larkMu.Lock()
	s.larkCache = cachedConfig[LarkConfig]{}
	s.larkMu.Unlock()
	return nil
}

// ---- OIDC config -------------------------------------------------------------

// GetOIDCConfig loads the OIDC configuration from cache or DB.
// Cache TTL is cacheTTL (30 s); writes invalidate the cache immediately.
// Returns empty struct (Enabled=false) if no settings have been saved yet.
func (s *SystemSettingService) GetOIDCConfig(ctx context.Context) (OIDCConfigDB, error) {
	// Fast path: read from cache.
	s.oidcMu.RLock()
	if s.oidcCache.valid() {
		cfg := s.oidcCache.value
		s.oidcMu.RUnlock()
		return cfg, nil
	}
	s.oidcMu.RUnlock()

	// Slow path: load from DB and repopulate cache.
	kv, err := s.repo.ListByGroup(ctx, groupOIDC)
	if err != nil {
		return OIDCConfigDB{}, err
	}
	cfg := OIDCConfigDB{
		Enabled:       parseBool(kv["enabled"]),
		IssuerURL:     kv["issuer_url"],
		ClientID:      kv["client_id"],
		ClientSecret:  s.getDecrypted(groupOIDC, "client_secret", kv["client_secret"]),
		RedirectURL:   kv["redirect_url"],
		Scopes:        strDef(kv["scopes"], "openid,profile,email"),
		UsernameClaim: strDef(kv["username_claim"], "preferred_username"),
		EmailClaim:    strDef(kv["email_claim"], "email"),
		RoleClaim:     strDef(kv["role_claim"], "realm_access.roles"),
		RoleMapping:   kv["role_mapping"],
		DefaultRole:   strDef(kv["default_role"], "viewer"),
		AutoProvision: parseBoolDef(kv["auto_provision"], true),
	}

	s.oidcMu.Lock()
	s.oidcCache = cachedConfig[OIDCConfigDB]{value: cfg, expiresAt: time.Now().Add(cacheTTL)}
	s.oidcMu.Unlock()

	return cfg, nil
}

// SaveOIDCConfig persists all OIDC configuration keys to the DB and invalidates cache.
// Empty client_secret means "do not overwrite the existing secret".
func (s *SystemSettingService) SaveOIDCConfig(ctx context.Context, cfg OIDCConfigDB) error {
	kv := map[string]string{
		"enabled":        strconv.FormatBool(cfg.Enabled),
		"issuer_url":     cfg.IssuerURL,
		"client_id":      cfg.ClientID,
		"redirect_url":   cfg.RedirectURL,
		"scopes":         cfg.Scopes,
		"username_claim": cfg.UsernameClaim,
		"email_claim":    cfg.EmailClaim,
		"role_claim":     cfg.RoleClaim,
		"role_mapping":   cfg.RoleMapping,
		"default_role":   cfg.DefaultRole,
		"auto_provision": strconv.FormatBool(cfg.AutoProvision),
	}
	// Only save client_secret when caller provided a non-empty value.
	if cfg.ClientSecret != "" {
		enc, err := s.setEncrypted(groupOIDC, "client_secret", cfg.ClientSecret)
		if err != nil {
			s.logger.Error("failed to encrypt oidc.client_secret", zap.Error(err))
			return err
		}
		kv["client_secret"] = enc
	}
	if err := s.repo.SetGroup(ctx, groupOIDC, kv); err != nil {
		return err
	}
	// Invalidate cache so the next read fetches fresh data.
	s.oidcMu.Lock()
	s.oidcCache = cachedConfig[OIDCConfigDB]{}
	s.oidcMu.Unlock()
	return nil
}

// ---- SMTP config -------------------------------------------------------------

// GetSMTPConfig loads global SMTP configuration from cache or DB.
func (s *SystemSettingService) GetSMTPConfig(ctx context.Context) (SMTPConfig, error) {
	s.smtpMu.RLock()
	if s.smtpCache.valid() {
		cfg := s.smtpCache.value
		s.smtpMu.RUnlock()
		return cfg, nil
	}
	s.smtpMu.RUnlock()

	kv, err := s.repo.ListByGroup(ctx, groupSMTP)
	if err != nil {
		return SMTPConfig{}, err
	}
	port := 587
	if v, ok := kv["smtp_port"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			port = n
		}
	}
	cfg := SMTPConfig{
		SMTPHost: kv["smtp_host"],
		SMTPPort: port,
		SMTPTLS:  parseBool(kv["smtp_tls"]),
		Username: kv["username"],
		Password: s.getDecrypted(groupSMTP, "password", kv["password"]),
		From:     kv["from"],
		Enabled:  parseBool(kv["enabled"]),
	}

	s.smtpMu.Lock()
	s.smtpCache = cachedConfig[SMTPConfig]{value: cfg, expiresAt: time.Now().Add(cacheTTL)}
	s.smtpMu.Unlock()
	return cfg, nil
}

// SaveSMTPConfig persists global SMTP configuration to DB and invalidates cache.
// Empty password means "do not overwrite existing password".
func (s *SystemSettingService) SaveSMTPConfig(ctx context.Context, cfg SMTPConfig) error {
	kv := map[string]string{
		"smtp_host": cfg.SMTPHost,
		"smtp_port": strconv.Itoa(cfg.SMTPPort),
		"smtp_tls":  strconv.FormatBool(cfg.SMTPTLS),
		"username":  cfg.Username,
		"from":      cfg.From,
		"enabled":   strconv.FormatBool(cfg.Enabled),
	}
	if cfg.Password != "" {
		enc, err := s.setEncrypted(groupSMTP, "password", cfg.Password)
		if err != nil {
			s.logger.Error("failed to encrypt smtp.password", zap.Error(err))
			return err
		}
		kv["password"] = enc
	}
	if err := s.repo.SetGroup(ctx, groupSMTP, kv); err != nil {
		return err
	}
	s.smtpMu.Lock()
	s.smtpCache = cachedConfig[SMTPConfig]{}
	s.smtpMu.Unlock()
	return nil
}

// ---- AI Module config --------------------------------------------------------

// AIModule describes a single AI capability module.
type AIModule struct {
	Enabled     bool   `json:"enabled"`
	Desc        string `json:"description"`
	ProviderKey string `json:"provider_key,omitempty"` // which provider this module uses (empty = default)
}

// AIModuleConfig holds the on/off state for each AI capability.
type AIModuleConfig struct {
	Platform AIModule `json:"platform"`
	Chat     AIModule `json:"chat"`
	RuleGen  AIModule `json:"rule_gen"`
	Analysis AIModule `json:"analysis"`
	Agent    AIModule `json:"agent"`
}

// defaultAIModuleConfig returns a config with all modules disabled.
func defaultAIModuleConfig() AIModuleConfig {
	return AIModuleConfig{
		Platform: AIModule{Desc: "AI platform foundation"},
		Chat:     AIModule{Desc: "AI chat assistant"},
		RuleGen:  AIModule{Desc: "AI-powered rule generation"},
		Analysis: AIModule{Desc: "AI alert analysis & root cause"},
		Agent:    AIModule{Desc: "Autonomous AI agent"},
	}
}

// GetAIModules loads the AI module configuration from DB.
// Returns defaults (all disabled) if no settings have been saved yet.
func (s *SystemSettingService) GetAIModules(ctx context.Context) (*AIModuleConfig, error) {
	kv, err := s.repo.ListByGroup(ctx, "ai_modules")
	if err != nil {
		return nil, err
	}
	if len(kv) == 0 {
		cfg := defaultAIModuleConfig()
		return &cfg, nil
	}
	cfg := defaultAIModuleConfig()
	cfg.Platform.Enabled = parseBool(kv["platform_enabled"])
	cfg.Chat.Enabled = parseBool(kv["chat_enabled"])
	cfg.RuleGen.Enabled = parseBool(kv["rule_gen_enabled"])
	cfg.Analysis.Enabled = parseBool(kv["analysis_enabled"])
	cfg.Agent.Enabled = parseBool(kv["agent_enabled"])
	if v, ok := kv["platform_desc"]; ok && v != "" {
		cfg.Platform.Desc = v
	}
	if v, ok := kv["chat_desc"]; ok && v != "" {
		cfg.Chat.Desc = v
	}
	if v, ok := kv["rule_gen_desc"]; ok && v != "" {
		cfg.RuleGen.Desc = v
	}
	if v, ok := kv["analysis_desc"]; ok && v != "" {
		cfg.Analysis.Desc = v
	}
	if v, ok := kv["agent_desc"]; ok && v != "" {
		cfg.Agent.Desc = v
	}
	cfg.Platform.ProviderKey = kv["platform_provider_key"]
	cfg.Chat.ProviderKey = kv["chat_provider_key"]
	cfg.RuleGen.ProviderKey = kv["rule_gen_provider_key"]
	cfg.Analysis.ProviderKey = kv["analysis_provider_key"]
	cfg.Agent.ProviderKey = kv["agent_provider_key"]
	return &cfg, nil
}

// UpdateAIModules persists the AI module configuration to DB.
func (s *SystemSettingService) UpdateAIModules(ctx context.Context, cfg *AIModuleConfig) error {
	kv := map[string]string{
		"platform_enabled":      strconv.FormatBool(cfg.Platform.Enabled),
		"platform_desc":         cfg.Platform.Desc,
		"platform_provider_key": cfg.Platform.ProviderKey,
		"chat_enabled":          strconv.FormatBool(cfg.Chat.Enabled),
		"chat_desc":             cfg.Chat.Desc,
		"chat_provider_key":     cfg.Chat.ProviderKey,
		"rule_gen_enabled":      strconv.FormatBool(cfg.RuleGen.Enabled),
		"rule_gen_desc":         cfg.RuleGen.Desc,
		"rule_gen_provider_key": cfg.RuleGen.ProviderKey,
		"analysis_enabled":      strconv.FormatBool(cfg.Analysis.Enabled),
		"analysis_desc":         cfg.Analysis.Desc,
		"analysis_provider_key": cfg.Analysis.ProviderKey,
		"agent_enabled":         strconv.FormatBool(cfg.Agent.Enabled),
		"agent_desc":            cfg.Agent.Desc,
		"agent_provider_key":    cfg.Agent.ProviderKey,
	}
	if err := s.repo.SetGroup(ctx, "ai_modules", kv); err != nil {
		return err
	}
	// Invalidate AI caches since module config affects provider resolution.
	s.aiMu.Lock()
	s.aiCache = cachedConfig[AIConfig]{}
	s.aiMu.Unlock()
	s.providersMu.Lock()
	s.providersCache = cachedConfig[AIProvidersConfig]{}
	s.providersMu.Unlock()
	return nil
}

// ---- AI Global config (Tab 3 in unified settings) -----------------------------

// GetAIGlobalConfig loads platform-wide AI settings from DB.
func (s *SystemSettingService) GetAIGlobalConfig(ctx context.Context) (AIGlobalConfig, error) {
	kv, err := s.repo.ListByGroup(ctx, "ai_global")
	if err != nil {
		return AIGlobalConfig{}, err
	}
	if len(kv) == 0 {
		return AIGlobalConfig{
			RetryMax:           2,
			ContextMaxChars:    8000,
			DefaultTemperature: 0.3,
			DefaultMaxTokens:   1024,
		}, nil
	}
	cfg := AIGlobalConfig{
		RetryMax:           parseIntDef(kv["retry_max"], 2),
		ContextMaxChars:    parseIntDef(kv["context_max_chars"], 8000),
		DefaultTemperature: parseFloatDef(kv["default_temperature"], 0.3),
		DefaultMaxTokens:   parseIntDef(kv["default_max_tokens"], 1024),
		MonthlyTokenBudget: parseInt64Def(kv["monthly_token_budget"], 0),
		DataMaskingEnabled: parseBool(kv["data_masking_enabled"]),
	}
	return cfg, nil
}

// SaveAIGlobalConfig persists platform-wide AI settings to DB.
func (s *SystemSettingService) SaveAIGlobalConfig(ctx context.Context, cfg AIGlobalConfig) error {
	kv := map[string]string{
		"retry_max":            strconv.Itoa(cfg.RetryMax),
		"context_max_chars":    strconv.Itoa(cfg.ContextMaxChars),
		"default_temperature":  fmt.Sprintf("%.2f", cfg.DefaultTemperature),
		"default_max_tokens":   strconv.Itoa(cfg.DefaultMaxTokens),
		"monthly_token_budget": strconv.FormatInt(cfg.MonthlyTokenBudget, 10),
		"data_masking_enabled": strconv.FormatBool(cfg.DataMaskingEnabled),
	}
	return s.repo.SetGroup(ctx, "ai_global", kv)
}

// ---- Security config ----------------------------------------------------------

// GetSecurityConfig loads security settings from cache or DB.
func (s *SystemSettingService) GetSecurityConfig(ctx context.Context, defaultExpire int) (SecurityConfig, error) {
	kv, err := s.repo.ListByGroup(ctx, groupSecurity)
	if err != nil {
		return SecurityConfig{JWTExpireSeconds: defaultExpire}, err
	}
	expire := defaultExpire
	if v, ok := kv["jwt_expire_seconds"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			expire = n
		}
	}
	return SecurityConfig{JWTExpireSeconds: expire}, nil
}

// SaveSecurityConfig persists security settings to DB.
func (s *SystemSettingService) SaveSecurityConfig(ctx context.Context, cfg SecurityConfig) error {
	kv := map[string]string{
		"jwt_expire_seconds": strconv.Itoa(cfg.JWTExpireSeconds),
	}
	return s.repo.SetGroup(ctx, groupSecurity, kv)
}

// ---- Label Validation config -------------------------------------------------

// LabelValidationConfig holds the label validation settings stored in the DB.
type LabelValidationConfig struct {
	Enabled bool `json:"enabled"`
}

// GetLabelValidationConfig loads the label validation configuration from DB.
// Returns default (disabled) if no settings have been saved yet.
func (s *SystemSettingService) GetLabelValidationConfig(ctx context.Context) (LabelValidationConfig, error) {
	kv, err := s.repo.ListByGroup(ctx, "alert_rule")
	if err != nil {
		return LabelValidationConfig{}, err
	}
	return LabelValidationConfig{
		Enabled: parseBool(kv["label_validation_enabled"]),
	}, nil
}

// SaveLabelValidationConfig persists the label validation configuration to DB.
func (s *SystemSettingService) SaveLabelValidationConfig(ctx context.Context, cfg LabelValidationConfig) error {
	kv := map[string]string{
		"label_validation_enabled": strconv.FormatBool(cfg.Enabled),
	}
	return s.repo.SetGroup(ctx, "alert_rule", kv)
}

// ---- Site Info config --------------------------------------------------------

// SiteInfo holds site-wide branding and customization settings.
type SiteInfo struct {
	SiteName      string `json:"site_name"`
	LogoURL       string `json:"logo_url"`
	FaviconURL    string `json:"favicon_url"`
	LoginTitle    string `json:"login_title"`
	LoginSubTitle string `json:"login_subtitle"`
	FooterText    string `json:"footer_text"`
	CustomCSS     string `json:"custom_css"`
}

// GetSiteInfo loads site branding configuration from DB.
func (s *SystemSettingService) GetSiteInfo(ctx context.Context) (SiteInfo, error) {
	kv, err := s.repo.ListByGroup(ctx, groupSiteInfo)
	if err != nil {
		return SiteInfo{}, err
	}
	return SiteInfo{
		SiteName:      kv["site_name"],
		LogoURL:       kv["logo_url"],
		FaviconURL:    kv["favicon_url"],
		LoginTitle:    kv["login_title"],
		LoginSubTitle: kv["login_subtitle"],
		FooterText:    kv["footer_text"],
		CustomCSS:     kv["custom_css"],
	}, nil
}

// SaveSiteInfo persists site branding configuration to DB.
func (s *SystemSettingService) SaveSiteInfo(ctx context.Context, cfg SiteInfo) error {
	kv := map[string]string{
		"site_name":      cfg.SiteName,
		"logo_url":       cfg.LogoURL,
		"favicon_url":    cfg.FaviconURL,
		"login_title":    cfg.LoginTitle,
		"login_subtitle": cfg.LoginSubTitle,
		"footer_text":    cfg.FooterText,
		"custom_css":     cfg.CustomCSS,
	}
	return s.repo.SetGroup(ctx, groupSiteInfo, kv)
}

// ---- helpers -----------------------------------------------------------------

func strDef(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func parseBool(v string) bool {
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}

// parseBoolDef parses a bool string with a default value when the string is empty.
func parseBoolDef(v string, def bool) bool {
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

// parseIntDef parses an int string with a default value when the string is empty.
func parseIntDef(v string, def int) int {
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// parseFloatDef parses a float64 string with a default value when the string is empty.
func parseFloatDef(v string, def float64) float64 {
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func parseInt64Def(v string, def int64) int64 {
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}
