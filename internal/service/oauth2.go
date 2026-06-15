package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/middleware"
	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/safehttp"
	"github.com/sreagent/sreagent/internal/repository"
)

// OAuth2Config holds generic OAuth2 SSO configuration stored in the DB.
type OAuth2Config struct {
	Enabled       bool   `json:"enabled"`
	Name          string `json:"name"` // display name, e.g. "GitHub"
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"` // encrypted
	AuthURL       string `json:"auth_url"`
	TokenURL      string `json:"token_url"`
	UserInfoURL   string `json:"user_info_url"`
	RedirectURL   string `json:"redirect_url"`
	Scopes        string `json:"scopes"`        // comma-separated
	UserIDField   string `json:"user_id_field"` // field in userinfo JSON
	EmailField    string `json:"email_field"`
	UsernameField string `json:"username_field"`
	DefaultRole   string `json:"default_role"`   // default "viewer"
	AutoProvision bool   `json:"auto_provision"` // default true
}

// OAuth2UserInfo holds user information from the OAuth2 provider.
type OAuth2UserInfo struct {
	UserID      string
	Username    string
	Email       string
	DisplayName string
	// EmailVerified mirrors the provider's email_verified claim when present
	// (nil = not provided). Used to guard email-based account linking.
	EmailVerified *bool
}

// OAuth2Service handles generic OAuth2 authorization code flow.
type OAuth2Service struct {
	settingSvc *SystemSettingService
	userRepo   *repository.UserRepository
	logger     *zap.Logger
}

// NewOAuth2Service creates a new OAuth2Service.
func NewOAuth2Service(
	settingSvc *SystemSettingService,
	userRepo *repository.UserRepository,
	logger *zap.Logger,
) *OAuth2Service {
	return &OAuth2Service{
		settingSvc: settingSvc,
		userRepo:   userRepo,
		logger:     logger.Named("oauth2"),
	}
}

// GetConfig loads the OAuth2 configuration from the DB.
func (s *OAuth2Service) GetConfig(ctx context.Context) (OAuth2Config, error) {
	kv, err := s.settingSvc.repo.ListByGroup(ctx, "oauth2")
	if err != nil {
		return OAuth2Config{}, err
	}
	return OAuth2Config{
		Enabled:       parseBool(kv["enabled"]),
		Name:          strDef(kv["name"], "OAuth2"),
		ClientID:      kv["client_id"],
		ClientSecret:  s.settingSvc.getDecrypted("oauth2", "client_secret", kv["client_secret"]),
		AuthURL:       kv["auth_url"],
		TokenURL:      kv["token_url"],
		UserInfoURL:   kv["user_info_url"],
		RedirectURL:   kv["redirect_url"],
		Scopes:        strDef(kv["scopes"], "openid,profile,email"),
		UserIDField:   strDef(kv["user_id_field"], "sub"),
		EmailField:    strDef(kv["email_field"], "email"),
		UsernameField: strDef(kv["username_field"], "preferred_username"),
		DefaultRole:   strDef(kv["default_role"], "viewer"),
		AutoProvision: parseBoolDef(kv["auto_provision"], true),
	}, nil
}

// SaveConfig persists the OAuth2 configuration to the DB.
func (s *OAuth2Service) SaveConfig(ctx context.Context, cfg OAuth2Config) error {
	kv := map[string]string{
		"enabled":        fmt.Sprintf("%t", cfg.Enabled),
		"name":           cfg.Name,
		"client_id":      cfg.ClientID,
		"auth_url":       cfg.AuthURL,
		"token_url":      cfg.TokenURL,
		"user_info_url":  cfg.UserInfoURL,
		"redirect_url":   cfg.RedirectURL,
		"scopes":         cfg.Scopes,
		"user_id_field":  cfg.UserIDField,
		"email_field":    cfg.EmailField,
		"username_field": cfg.UsernameField,
		"default_role":   cfg.DefaultRole,
		"auto_provision": fmt.Sprintf("%t", cfg.AutoProvision),
	}
	// Only save client_secret when caller provided a non-empty value.
	if cfg.ClientSecret != "" {
		enc, err := s.settingSvc.setEncrypted("oauth2", "client_secret", cfg.ClientSecret)
		if err != nil {
			s.logger.Error("failed to encrypt oauth2.client_secret", zap.Error(err))
			return err
		}
		kv["client_secret"] = enc
	}
	return s.settingSvc.repo.SetGroup(ctx, "oauth2", kv)
}

// GetAuthURL returns the OAuth2 authorization URL with a random state parameter.
func (s *OAuth2Service) GetAuthURL(ctx context.Context) (string, string, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return "", "", fmt.Errorf("oauth2: failed to load config: %w", err)
	}
	if !cfg.Enabled {
		return "", "", fmt.Errorf("oauth2: authentication is disabled")
	}

	// Generate random state
	state, err := generateRandomState()
	if err != nil {
		return "", "", fmt.Errorf("oauth2: failed to generate state: %w", err)
	}

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", cfg.ClientID)
	params.Set("redirect_uri", cfg.RedirectURL)
	params.Set("scope", cfg.Scopes)
	params.Set("state", state)

	authURL := cfg.AuthURL + "?" + params.Encode()
	return authURL, state, nil
}

// ExchangeAndLogin exchanges the authorization code for tokens, fetches user info,
// provisions or finds the user, and returns a platform JWT.
func (s *OAuth2Service) ExchangeAndLogin(ctx context.Context, code string, jwtSecret string, jwtExpire int) (string, int, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("oauth2: failed to load config: %w", err)
	}
	if !cfg.Enabled {
		return "", 0, fmt.Errorf("oauth2: authentication is disabled")
	}

	// Exchange code for access token
	accessToken, err := s.exchangeToken(ctx, cfg, code)
	if err != nil {
		return "", 0, err
	}

	// Fetch user info from the provider
	info, err := s.fetchUserInfo(ctx, cfg, accessToken)
	if err != nil {
		return "", 0, err
	}

	// Find or create user
	user, err := s.findOrCreateUser(ctx, cfg, info)
	if err != nil {
		return "", 0, fmt.Errorf("oauth2: user provisioning failed: %w", err)
	}

	if !user.IsActive {
		return "", 0, fmt.Errorf("oauth2: account is disabled")
	}

	// Generate JWT
	token, err := middleware.GenerateToken(user.ID, user.Username, string(user.Role), jwtSecret, jwtExpire)
	if err != nil {
		return "", 0, fmt.Errorf("oauth2: failed to generate token: %w", err)
	}

	s.logger.Info("OAuth2 login successful",
		zap.Uint("user_id", user.ID),
		zap.String("username", user.Username),
		zap.String("role", string(user.Role)),
	)

	return token, jwtExpire, nil
}

// exchangeToken exchanges the authorization code for an access token.
func (s *OAuth2Service) exchangeToken(ctx context.Context, cfg OAuth2Config, code string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", cfg.RedirectURL)
	data.Set("client_id", cfg.ClientID)
	data.Set("client_secret", cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", cfg.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("oauth2: create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// SSRF-safe client: TokenURL is admin-configurable, so block requests
	// to loopback/link-local/metadata addresses (same as the OIDC/Lark paths).
	client := safehttp.NewSafeClient(15 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth2: token exchange request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return "", fmt.Errorf("oauth2: read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oauth2: token exchange failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse the token response — support both JSON and form-encoded
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("oauth2: parse token response: %w (body: %s)", err, string(body))
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("oauth2: token error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("oauth2: no access_token in response: %s", string(body))
	}

	return tokenResp.AccessToken, nil
}

// fetchUserInfo fetches user information from the OAuth2 provider's userinfo endpoint.
func (s *OAuth2Service) fetchUserInfo(ctx context.Context, cfg OAuth2Config, accessToken string) (*OAuth2UserInfo, error) {
	if cfg.UserInfoURL == "" {
		return nil, fmt.Errorf("oauth2: user_info_url is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", cfg.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("oauth2: create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	// SSRF-safe client: UserInfoURL is admin-configurable (see exchangeToken).
	client := safehttp.NewSafeClient(15 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth2: userinfo request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return nil, fmt.Errorf("oauth2: read userinfo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oauth2: userinfo failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse the userinfo response as a flat map for flexible field mapping
	var rawInfo map[string]interface{}
	if err := json.Unmarshal(body, &rawInfo); err != nil {
		return nil, fmt.Errorf("oauth2: parse userinfo response: %w", err)
	}

	info := &OAuth2UserInfo{
		UserID:        getStringField(rawInfo, cfg.UserIDField),
		Username:      getStringField(rawInfo, cfg.UsernameField),
		Email:         getStringField(rawInfo, cfg.EmailField),
		DisplayName:   getStringField(rawInfo, "name"),
		EmailVerified: getBoolField(rawInfo, "email_verified"),
	}

	// Fallback: if no display name, try nickname or username
	if info.DisplayName == "" {
		info.DisplayName = getStringField(rawInfo, "nickname")
	}
	if info.DisplayName == "" {
		info.DisplayName = info.Username
	}
	// Fallback: if no username, use user_id
	if info.Username == "" {
		info.Username = info.UserID
	}

	return info, nil
}

// findOrCreateUser looks up the user by OAuth2 user ID (stored in OIDCSubject),
// then by email, then by username. Auto-creates if not found and auto_provision is enabled.
func (s *OAuth2Service) findOrCreateUser(ctx context.Context, cfg OAuth2Config, info *OAuth2UserInfo) (*model.User, error) {
	ssoInfo := &SSOUserInfo{
		Subject:       "oauth2:" + info.UserID,
		Username:      info.Username,
		DisplayName:   info.DisplayName,
		Email:         info.Email,
		Source:        "oauth2",
		EmailVerified: info.EmailVerified,
	}

	user, err := LookupSSOUser(ctx, s.userRepo, ssoInfo)
	if err == nil {
		if UpdateUserFromSSO(user, ssoInfo, s.logger) {
			if err := s.userRepo.Update(ctx, user); err != nil {
				s.logger.Warn("failed to update user from OAuth2", zap.Uint("user_id", user.ID), zap.Error(err))
			}
		}
		return user, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if !cfg.AutoProvision {
		return nil, fmt.Errorf("oauth2: user not found and auto_provision is disabled")
	}

	defaultRole := model.Role(cfg.DefaultRole)
	return AutoCreateSSOUser(ctx, s.userRepo, ssoInfo, defaultRole, s.logger)
}

// Enabled returns whether OAuth2 is configured and active.
func (s *OAuth2Service) Enabled() bool {
	if s == nil {
		return false
	}
	cfg, err := s.GetConfig(context.Background())
	return err == nil && cfg.Enabled
}

// GetName returns the display name of the OAuth2 provider.
func (s *OAuth2Service) GetName() string {
	if s == nil {
		return ""
	}
	cfg, err := s.GetConfig(context.Background())
	if err != nil {
		return ""
	}
	return cfg.Name
}

// generateRandomState generates a cryptographically random state string.
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// getStringField extracts a string value from a map, supporting nested keys with dot notation.
// getBoolField extracts a boolean claim, tolerating bool and "true"/"false"
// string encodings. Returns nil when the key is absent so callers can distinguish
// "not provided" from an explicit false.
func getBoolField(m map[string]interface{}, key string) *bool {
	if key == "" {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case bool:
		return &t
	case string:
		if t == "true" {
			b := true
			return &b
		}
		if t == "false" {
			b := false
			return &b
		}
	}
	return nil
}

func getStringField(m map[string]interface{}, key string) string {
	if key == "" {
		return ""
	}
	// Support dot-notation for nested fields (e.g. "realm_access.roles")
	parts := strings.Split(key, ".")
	var current interface{} = m
	for _, part := range parts {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current = obj[part]
	}
	if s, ok := current.(string); ok {
		return s
	}
	return ""
}
