package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// SSOUserInfo holds user information from an SSO provider (LDAP, OAuth2, OIDC).
type SSOUserInfo struct {
	Subject     string // OIDC subject or "oauth2:<id>"; empty for pure LDAP
	Username    string
	DisplayName string
	Email       string
	Avatar      string     // picture URL (OIDC only)
	Role        model.Role // override role from OIDC claims; empty to use default
	Source      string     // "ldap", "oauth2", "oidc" (for logging)

	// EmailVerified tri-states the IdP's email_verified assertion:
	//   nil   = unknown / claim not provided (e.g. LDAP, or an OAuth2 provider that
	//           omits it) — preserve legacy behavior and allow email-based linking.
	//   false = IdP explicitly asserted the email is NOT verified — refuse to link by
	//           email (account-takeover guard).
	//   true  = verified.
	EmailVerified *bool
}

// LookupSSOUser finds an existing user by subject, email, then username.
// When a user is found by email or username and subject is non-empty,
// the subject is linked to the existing account.
func LookupSSOUser(ctx context.Context, userRepo *repository.UserRepository, info *SSOUserInfo) (*model.User, error) {
	// 1. Try by OIDC subject (most reliable for OIDC/OAuth2)
	if info.Subject != "" {
		user, err := userRepo.GetByOIDCSubject(ctx, info.Subject)
		if err == nil {
			return user, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	// 2. Try by email (link existing account).
	// Account-takeover guard: only refuse email linking when the IdP *explicitly*
	// asserts the email is unverified. A nil (unknown) value preserves legacy
	// behavior so providers that omit email_verified keep working.
	if info.Email != "" && info.EmailVerified != nil && !*info.EmailVerified {
		// Skip email-based linking; fall through to username / auto-provision.
	} else if info.Email != "" {
		user, err := userRepo.GetByEmail(ctx, info.Email)
		if err == nil {
			if info.Subject != "" {
				user.OIDCSubject = info.Subject
			}
			return user, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	// 3. Try by username (link existing account)
	if info.Username != "" {
		user, err := userRepo.GetByUsername(ctx, info.Username)
		if err == nil {
			if info.Subject != "" {
				user.OIDCSubject = info.Subject
			}
			return user, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	return nil, gorm.ErrRecordNotFound
}

// AutoCreateSSOUser creates a new user from SSO info when auto-provisioning is allowed.
func AutoCreateSSOUser(ctx context.Context, userRepo *repository.UserRepository, info *SSOUserInfo, defaultRole model.Role, logger *zap.Logger) (*model.User, error) {
	role := info.Role
	if role == "" || !role.IsValid() {
		role = defaultRole
		if !role.IsValid() {
			role = model.RoleViewer
		}
	}

	username := info.Username
	if username == "" {
		username = info.Email
	}
	if username == "" {
		username = info.Subject
	}

	displayName := info.DisplayName
	if displayName == "" {
		displayName = username
	}

	newUser := &model.User{
		Username:    username,
		Password:    "",
		DisplayName: displayName,
		Email:       info.Email,
		Avatar:      info.Avatar,
		Role:        role,
		IsActive:    true,
		UserType:    model.UserTypeHuman,
		OIDCSubject: info.Subject,
	}

	if err := userRepo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("create %s user: %w", info.Source, err)
	}

	logger.Info("auto-provisioned SSO user",
		zap.String("source", info.Source),
		zap.Uint("user_id", newUser.ID),
		zap.String("username", newUser.Username),
		zap.String("email", newUser.Email),
		zap.String("role", string(newUser.Role)),
	)

	return newUser, nil
}

// UpdateUserFromSSO updates user profile fields from SSO info if they have changed.
// Returns true if any field was modified.
// When a logger is provided, role changes are logged for audit purposes.
func UpdateUserFromSSO(user *model.User, info *SSOUserInfo, logger *zap.Logger) bool {
	changed := false
	if info.DisplayName != "" && user.DisplayName != info.DisplayName {
		user.DisplayName = info.DisplayName
		changed = true
	}
	if info.Email != "" && user.Email != info.Email {
		user.Email = info.Email
		changed = true
	}
	if info.Avatar != "" && user.Avatar != info.Avatar {
		user.Avatar = info.Avatar
		changed = true
	}
	if info.Role != "" && info.Role.IsValid() && user.Role != info.Role {
		oldRole := user.Role
		user.Role = info.Role
		changed = true
		if logger != nil {
			logger.Warn("SSO role change audit",
				zap.String("source", info.Source),
				zap.Uint("user_id", user.ID),
				zap.String("username", user.Username),
				zap.String("old_role", string(oldRole)),
				zap.String("new_role", string(info.Role)),
				zap.String("subject", info.Subject),
			)
		}
	}
	return changed
}
