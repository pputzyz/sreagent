package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/config"
	"github.com/sreagent/sreagent/internal/middleware"
	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// OIDCService handles OIDC authorization code flow, user provisioning,
// and role mapping from an external identity provider (e.g. Keycloak).
type OIDCService struct {
	provider  *oidc.Provider
	verifier  *oidc.IDTokenVerifier
	oauth2Cfg *oauth2.Config
	oidcCfg   *config.OIDCConfig
	jwtCfg    *config.JWTConfig
	userRepo  *repository.UserRepository
	logger    *zap.Logger
}

// NewOIDCService initializes the OIDC provider via discovery and returns the service.
// This performs an HTTP call to the issuer's .well-known/openid-configuration endpoint.
func NewOIDCService(
	ctx context.Context,
	oidcCfg *config.OIDCConfig,
	jwtCfg *config.JWTConfig,
	userRepo *repository.UserRepository,
	logger *zap.Logger,
) (*OIDCService, error) {
	provider, err := oidc.NewProvider(ctx, oidcCfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidc provider discovery failed for %s: %w", oidcCfg.IssuerURL, err)
	}

	scopes := oidcCfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     oidcCfg.ClientID,
		ClientSecret: oidcCfg.ClientSecret,
		RedirectURL:  oidcCfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: oidcCfg.ClientID,
	})

	return &OIDCService{
		provider:  provider,
		verifier:  verifier,
		oauth2Cfg: oauth2Cfg,
		oidcCfg:   oidcCfg,
		jwtCfg:    jwtCfg,
		userRepo:  userRepo,
		logger:    logger.Named("oidc"),
	}, nil
}

// GenerateAuthURL returns the OIDC authorization URL and a random state value.
// The caller should store the state in a cookie/session to verify the callback.
func (s *OIDCService) GenerateAuthURL() (authURL string, state string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate oidc state: %w", err)
	}
	state = base64.RawURLEncoding.EncodeToString(b)
	authURL = s.oauth2Cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return authURL, state, nil
}

// ExchangeAndLogin exchanges the authorization code for tokens, extracts user info
// from the ID token, provisions or updates the user, and returns a platform JWT.
func (s *OIDCService) ExchangeAndLogin(ctx context.Context, code string) (token string, expiresIn int, err error) {
	// Exchange authorization code for OAuth2 token
	oauth2Token, err := s.oauth2Cfg.Exchange(ctx, code)
	if err != nil {
		return "", 0, fmt.Errorf("oidc code exchange failed: %w", err)
	}

	// Extract and verify the ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return "", 0, fmt.Errorf("oidc response missing id_token")
	}

	idToken, err := s.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return "", 0, fmt.Errorf("oidc id_token verification failed: %w", err)
	}

	// Extract claims
	var claims struct {
		Sub               string `json:"sub"`
		PreferredUsername string `json:"preferred_username"`
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		GivenName         string `json:"given_name"`
		FamilyName        string `json:"family_name"`
		Picture           string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return "", 0, fmt.Errorf("oidc parse claims: %w", err)
	}

	// Extract roles from the configurable claim path
	role := s.extractRole(rawIDToken)

	// Find or create user via shared SSO helpers
	ssoInfo := &SSOUserInfo{
		Subject:     claims.Sub,
		Username:    claims.PreferredUsername,
		DisplayName: claims.Name,
		Email:       claims.Email,
		Avatar:      claims.Picture,
		Role:        role,
		Source:      "oidc",
	}
	user, err := s.findOrCreateUser(ctx, ssoInfo)
	if err != nil {
		return "", 0, fmt.Errorf("oidc user provisioning: %w", err)
	}

	if !user.IsActive {
		return "", 0, fmt.Errorf("oidc login: account is disabled")
	}

	// Generate platform JWT
	token, err = middleware.GenerateToken(user.ID, user.Username, string(user.Role), s.jwtCfg.Secret, s.jwtCfg.Expire)
	if err != nil {
		return "", 0, fmt.Errorf("oidc generate platform token: %w", err)
	}

	s.logger.Info("OIDC login successful",
		zap.Uint("user_id", user.ID),
		zap.String("username", user.Username),
		zap.String("role", string(user.Role)),
		zap.String("oidc_sub", claims.Sub),
	)

	return token, s.jwtCfg.Expire, nil
}

// extractRole parses the raw ID token claims to find roles using the configured
// role_claim path (e.g. "realm_access.roles") and maps them to an SREAgent role.
func (s *OIDCService) extractRole(rawIDToken string) model.Role {
	roleClaim := s.oidcCfg.RoleClaim
	if roleClaim == "" {
		roleClaim = "realm_access.roles"
	}

	defaultRole := model.Role(s.oidcCfg.DefaultRole)
	if defaultRole == "" {
		defaultRole = model.RoleViewer
	}

	// Decode the JWT payload (second segment) without verification (already verified)
	parts := strings.Split(rawIDToken, ".")
	if len(parts) != 3 {
		return defaultRole
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		s.logger.Warn("failed to decode id_token payload for role extraction", zap.Error(err))
		return defaultRole
	}

	var rawClaims map[string]json.RawMessage
	if err := json.Unmarshal(payload, &rawClaims); err != nil {
		s.logger.Warn("failed to unmarshal id_token payload", zap.Error(err))
		return defaultRole
	}

	// Navigate the claim path (e.g. "realm_access.roles" -> rawClaims["realm_access"]["roles"])
	pathParts := strings.Split(roleClaim, ".")
	roles := s.navigateClaimPath(rawClaims, pathParts)
	if len(roles) == 0 {
		return defaultRole
	}

	// Map OIDC roles to SREAgent roles using the configured mapping
	return s.mapRole(roles, defaultRole)
}

// navigateClaimPath walks through nested JSON claims following the dot-separated path.
func (s *OIDCService) navigateClaimPath(claims map[string]json.RawMessage, path []string) []string {
	if len(path) == 0 {
		return nil
	}

	raw, ok := claims[path[0]]
	if !ok {
		return nil
	}

	if len(path) == 1 {
		// Terminal: expect a string array
		var roles []string
		if err := json.Unmarshal(raw, &roles); err != nil {
			// Try single string
			var single string
			if err2 := json.Unmarshal(raw, &single); err2 == nil {
				return []string{single}
			}
			return nil
		}
		return roles
	}

	// Intermediate: expect a nested object
	var nested map[string]json.RawMessage
	if err := json.Unmarshal(raw, &nested); err != nil {
		return nil
	}
	return s.navigateClaimPath(nested, path[1:])
}

// mapRole finds the highest-privilege SREAgent role from the OIDC roles.
func (s *OIDCService) mapRole(oidcRoles []string, defaultRole model.Role) model.Role {
	if len(s.oidcCfg.RoleMapping) == 0 {
		return defaultRole
	}

	// Priority order: admin > team_lead > member > global_viewer > viewer
	rolePriority := map[model.Role]int{
		model.RoleAdmin:        5,
		model.RoleTeamLead:     4,
		model.RoleMember:       3,
		model.RoleGlobalViewer: 2,
		model.RoleViewer:       1,
	}

	bestRole := defaultRole
	bestPriority := rolePriority[defaultRole]

	for _, oidcRole := range oidcRoles {
		if mapped, ok := s.oidcCfg.RoleMapping[oidcRole]; ok {
			sreRole := model.Role(mapped)
			if p, ok := rolePriority[sreRole]; ok && p > bestPriority {
				bestRole = sreRole
				bestPriority = p
			}
		}
	}

	return bestRole
}

// findOrCreateUser looks up the user by OIDC subject, then by email, then by username.
// Delegates to shared LookupSSOUser / AutoCreateSSOUser helpers to avoid duplication
// with LDAP and OAuth2 flows.
func (s *OIDCService) findOrCreateUser(ctx context.Context, info *SSOUserInfo) (*model.User, error) {
	user, err := LookupSSOUser(ctx, s.userRepo, info)
	if err == nil {
		// User found — update profile fields from OIDC claims
		if UpdateUserFromSSO(user, info) {
			if err := s.userRepo.Update(ctx, user); err != nil {
				s.logger.Warn("failed to update user from OIDC claims",
					zap.Uint("user_id", user.ID), zap.Error(err))
			}
		}
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Not found — auto-provision if allowed
	if !s.oidcCfg.AutoProvision {
		return nil, fmt.Errorf("user not found and auto_provision is disabled (sub=%s, email=%s)", info.Subject, info.Email)
	}

	defaultRole := model.Role(s.oidcCfg.DefaultRole)
	if defaultRole == "" {
		defaultRole = model.RoleViewer
	}

	return AutoCreateSSOUser(ctx, s.userRepo, info, defaultRole, s.logger)
}

// updateUserFromOIDC is no longer needed — profile updates are handled by
// UpdateUserFromSSO in sso_helper.go, called from findOrCreateUser.

// Enabled returns whether OIDC is configured and active.
func (s *OIDCService) Enabled() bool {
	return s != nil && s.provider != nil
}
