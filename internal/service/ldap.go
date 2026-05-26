package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/middleware"
	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/ldapx"
	"github.com/sreagent/sreagent/internal/repository"
)

// LDAPConfig holds LDAP server configuration stored in the DB.
type LDAPConfig struct {
	Enabled           bool   `json:"enabled"`
	Host              string `json:"host"`
	Port              int    `json:"port"`               // default 389
	BaseDN            string `json:"base_dn"`
	BindDN            string `json:"bind_dn"`             // service account
	BindPassword      string `json:"bind_password"`
	UserFilter        string `json:"user_filter"`         // e.g. "(uid=%s)"
	UserAttr          string `json:"user_attr"`           // default "uid"
	EmailAttr         string `json:"email_attr"`          // default "mail"
	DisplayNameAttr   string `json:"display_name_attr"`   // default "displayName"
	StartTLS          bool   `json:"start_tls"`
	SkipVerify        bool   `json:"skip_verify"`
	DefaultRole       string `json:"default_role"`        // default "viewer"
	AutoProvision     bool   `json:"auto_provision"`      // default true
}

// LDAPUserInfo holds user information extracted from LDAP.
type LDAPUserInfo struct {
	Username    string
	Email       string
	DisplayName string
}

// LDAPService handles LDAP authentication, user provisioning, and config management.
type LDAPService struct {
	settingSvc *SystemSettingService
	userRepo   *repository.UserRepository
	logger     *zap.Logger
}

// NewLDAPService creates a new LDAPService.
func NewLDAPService(
	settingSvc *SystemSettingService,
	userRepo *repository.UserRepository,
	logger *zap.Logger,
) *LDAPService {
	return &LDAPService{
		settingSvc: settingSvc,
		userRepo:   userRepo,
		logger:     logger.Named("ldap"),
	}
}

// GetConfig loads the LDAP configuration from the DB.
func (s *LDAPService) GetConfig(ctx context.Context) (LDAPConfig, error) {
	kv, err := s.settingSvc.repo.ListByGroup(ctx, "ldap")
	if err != nil {
		return LDAPConfig{}, err
	}
	port := parseIntDef(kv["port"], 389)
	return LDAPConfig{
		Enabled:         parseBool(kv["enabled"]),
		Host:            kv["host"],
		Port:            port,
		BaseDN:          kv["base_dn"],
		BindDN:          kv["bind_dn"],
		BindPassword:    s.settingSvc.getDecrypted("ldap", "bind_password", kv["bind_password"]),
		UserFilter:      kv["user_filter"],
		UserAttr:        strDef(kv["user_attr"], "uid"),
		EmailAttr:       strDef(kv["email_attr"], "mail"),
		DisplayNameAttr: strDef(kv["display_name_attr"], "displayName"),
		StartTLS:        parseBool(kv["start_tls"]),
		SkipVerify:      parseBool(kv["skip_verify"]),
		DefaultRole:     strDef(kv["default_role"], "viewer"),
		AutoProvision:   parseBoolDef(kv["auto_provision"], true),
	}, nil
}

// SaveConfig persists the LDAP configuration to the DB.
func (s *LDAPService) SaveConfig(ctx context.Context, cfg LDAPConfig) error {
	kv := map[string]string{
		"enabled":            fmt.Sprintf("%t", cfg.Enabled),
		"host":               cfg.Host,
		"port":               fmt.Sprintf("%d", cfg.Port),
		"base_dn":            cfg.BaseDN,
		"bind_dn":            cfg.BindDN,
		"user_filter":        cfg.UserFilter,
		"user_attr":          cfg.UserAttr,
		"email_attr":         cfg.EmailAttr,
		"display_name_attr":  cfg.DisplayNameAttr,
		"start_tls":          fmt.Sprintf("%t", cfg.StartTLS),
		"skip_verify":        fmt.Sprintf("%t", cfg.SkipVerify),
		"default_role":       cfg.DefaultRole,
		"auto_provision":     fmt.Sprintf("%t", cfg.AutoProvision),
	}
	// Only save bind_password when caller provided a non-empty value.
	if cfg.BindPassword != "" {
		enc, err := s.settingSvc.setEncrypted("ldap", "bind_password", cfg.BindPassword)
		if err != nil {
			s.logger.Error("failed to encrypt ldap.bind_password", zap.Error(err))
			return err
		}
		kv["bind_password"] = enc
	}
	return s.settingSvc.repo.SetGroup(ctx, "ldap", kv)
}

// Authenticate performs LDAP bind + search and returns user info.
// This is the main authentication entry point.
func (s *LDAPService) Authenticate(ctx context.Context, username, password string) (*LDAPUserInfo, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("ldap: failed to load config: %w", err)
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("ldap: authentication is disabled")
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	// Connect to LDAP server
	conn, err := ldapx.Connect(addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("ldap: connection failed: %w", err)
	}
	defer conn.Close()

	conn.SetTimeout(10 * time.Second)

	// StartTLS if configured (on a plaintext connection)
	if cfg.StartTLS {
		if err := conn.StartTLS(cfg.SkipVerify); err != nil {
			return nil, fmt.Errorf("ldap: starttls failed: %w", err)
		}
	}

	// Bind with service account to search for the user
	if cfg.BindDN != "" {
		if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
			return nil, fmt.Errorf("ldap: service account bind failed: %w", err)
		}
	}

	// Search for the user
	userFilter := cfg.UserFilter
	if userFilter == "" {
		userFilter = fmt.Sprintf("(%s=%%s)", cfg.UserAttr)
	}
	filter := fmt.Sprintf(userFilter, escapeLDAPFilter(username))

	searchResult, err := conn.Search(ldapx.SearchRequest{
		BaseDN:       cfg.BaseDN,
		Scope:        ldapx.ScopeWholeSubtree,
		DerefAliases: 2, // derefAlways
		SizeLimit:    1,
		TimeLimit:    10,
		Filter:       filter,
		Attributes:   []string{"dn", cfg.UserAttr, cfg.EmailAttr, cfg.DisplayNameAttr},
	})
	if err != nil {
		return nil, fmt.Errorf("ldap: search failed: %w", err)
	}
	if len(searchResult.Entries) == 0 {
		return nil, fmt.Errorf("ldap: user not found: %s", username)
	}

	entry := searchResult.Entries[0]
	userDN := entry.DN

	// Bind as the user to verify password
	if err := conn.Bind(userDN, password); err != nil {
		return nil, fmt.Errorf("ldap: authentication failed for %s: %w", username, err)
	}

	// Extract user info
	info := &LDAPUserInfo{
		Username:    firstAttr(entry.Attributes, cfg.UserAttr, username),
		Email:       firstAttr(entry.Attributes, cfg.EmailAttr, ""),
		DisplayName: firstAttr(entry.Attributes, cfg.DisplayNameAttr, username),
	}

	return info, nil
}

// AuthenticateAndLogin performs LDAP authentication and returns a JWT token.
// Auto-creates the user on first login if auto_provision is enabled.
func (s *LDAPService) AuthenticateAndLogin(ctx context.Context, username, password string, jwtSecret string, jwtExpire int) (string, int, error) {
	info, err := s.Authenticate(ctx, username, password)
	if err != nil {
		return "", 0, err
	}

	// Find or create user
	user, err := s.findOrCreateUser(ctx, info)
	if err != nil {
		return "", 0, fmt.Errorf("ldap: user provisioning failed: %w", err)
	}

	if !user.IsActive {
		return "", 0, fmt.Errorf("ldap: account is disabled")
	}

	// Generate JWT
	token, err := middleware.GenerateToken(user.ID, user.Username, string(user.Role), jwtSecret, jwtExpire)
	if err != nil {
		return "", 0, fmt.Errorf("ldap: failed to generate token: %w", err)
	}

	s.logger.Info("LDAP login successful",
		zap.Uint("user_id", user.ID),
		zap.String("username", user.Username),
		zap.String("role", string(user.Role)),
	)

	return token, jwtExpire, nil
}

// findOrCreateUser looks up the user by LDAP username, then by email.
// Auto-creates if not found and auto_provision is enabled.
func (s *LDAPService) findOrCreateUser(ctx context.Context, info *LDAPUserInfo) (*model.User, error) {
	ssoInfo := &SSOUserInfo{
		Username:    info.Username,
		DisplayName: info.DisplayName,
		Email:       info.Email,
		Source:      "ldap",
	}

	user, err := LookupSSOUser(ctx, s.userRepo, ssoInfo)
	if err == nil {
		if UpdateUserFromSSO(user, ssoInfo) {
			if err := s.userRepo.Update(ctx, user); err != nil {
				s.logger.Warn("failed to update user from LDAP", zap.Uint("user_id", user.ID), zap.Error(err))
			}
		}
		return user, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Load config to check auto_provision
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.AutoProvision {
		return nil, fmt.Errorf("ldap: user not found and auto_provision is disabled")
	}

	defaultRole := model.Role(cfg.DefaultRole)
	return AutoCreateSSOUser(ctx, s.userRepo, ssoInfo, defaultRole, s.logger)
}

// Enabled returns whether LDAP is configured and active.
func (s *LDAPService) Enabled() bool {
	if s == nil {
		return false
	}
	cfg, err := s.GetConfig(context.Background())
	return err == nil && cfg.Enabled
}

// TestConnection tests the LDAP connection by performing a bind with the service account.
func (s *LDAPService) TestConnection(ctx context.Context) (string, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load LDAP config: %w", err)
	}
	if cfg.Host == "" {
		return "", fmt.Errorf("LDAP host is not configured")
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	conn, err := ldapx.Connect(addr, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	if cfg.StartTLS {
		if err := conn.StartTLS(cfg.SkipVerify); err != nil {
			return "", fmt.Errorf("StartTLS failed: %w", err)
		}
	}

	if cfg.BindDN != "" {
		if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
			return "", fmt.Errorf("bind failed: %w", err)
		}
		return fmt.Sprintf("Successfully connected and authenticated as %s", cfg.BindDN), nil
	}

	return fmt.Sprintf("Successfully connected to %s (anonymous bind)", addr), nil
}

// escapeLDAPFilter escapes special characters in LDAP filter values per RFC 4515.
func escapeLDAPFilter(s string) string {
	r := strings.NewReplacer(
		"\\", "\\5c",
		"*", "\\2a",
		"(", "\\28",
		")", "\\29",
		"\x00", "\\00",
	)
	return r.Replace(s)
}

// firstAttr returns the first value of the named attribute, or def if not found.
func firstAttr(attrs map[string][]string, name, def string) string {
	if vals, ok := attrs[name]; ok && len(vals) > 0 {
		return vals[0]
	}
	return def
}

