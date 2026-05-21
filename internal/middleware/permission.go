package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/pkg/metrics"
	"github.com/sreagent/sreagent/internal/pkg/rbac"
)

// EnforceMode controls the behaviour of RequirePerm when a permission check fails.
//   - "deny" (default): return 403 and abort the request.
//   - "warn": log the denial but allow the request through (for gradual rollout).
var EnforceMode = "deny"

// SetEnforceMode sets the RBAC enforce mode. Valid values: "warn", "deny".
func SetEnforceMode(mode string) {
	if mode == "warn" || mode == "deny" {
		EnforceMode = mode
	}
}

// OnPermissionDenied is an optional callback invoked when RequirePerm denies access.
// Wired at startup to enable audit logging of permission denials without circular imports.
var OnPermissionDenied func(userID uint, perm string, path string)

// permLogger is set once at startup for warn-mode logging.
var permLogger *zap.Logger

// SetPermLogger sets the logger used by RequirePerm for warn-mode logging.
func SetPermLogger(l *zap.Logger) {
	permLogger = l
}

// RequirePerm returns a middleware that checks if the user's effective permissions
// (global role + team roles) include the required permission.
// For team-scoped endpoints, team roles can elevate permissions beyond the global role.
func RequirePerm(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextKeyRole)
		if !exists {
			if EnforceMode == "warn" {
				if permLogger != nil {
					permLogger.Warn("RBAC warn: no role in context",
						zap.String("path", c.Request.URL.Path),
						zap.String("required_perm", perm),
					)
				}
				c.Next()
				return
			}
			c.JSON(http.StatusForbidden, gin.H{
				"code":    10200,
				"message": "forbidden",
			})
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			if EnforceMode == "warn" {
				if permLogger != nil {
					permLogger.Warn("RBAC warn: invalid role type in context",
						zap.String("path", c.Request.URL.Path),
						zap.String("required_perm", perm),
					)
				}
				c.Next()
				return
			}
			c.JSON(http.StatusForbidden, gin.H{
				"code":    10200,
				"message": "forbidden: invalid role type in context",
			})
			c.Abort()
			return
		}

		// Check global role first (fast path — no DB query)
		if rbac.HasPerm(roleStr, perm) {
			c.Next()
			return
		}

		// Check team-level roles from context (set by auth middleware if available)
		if teamRolesRaw, ok := c.Get("user_team_roles"); ok {
			if teamRoles, ok := teamRolesRaw.([]string); ok {
				if perms := rbac.EffectivePerms(roleStr, teamRoles); perms[perm] {
					c.Next()
					return
				}
			}
		}

		// Permission denied — collect user info for audit callback.
		var uid uint
		if id, exists := c.Get(ContextKeyUserID); exists {
			if idUint, ok := id.(uint); ok {
				uid = idUint
			}
		}

		// Fire audit callback if wired (non-blocking, best-effort).
		if OnPermissionDenied != nil {
			OnPermissionDenied(uid, perm, c.Request.URL.Path)
		}

		if EnforceMode == "warn" {
			metrics.IncRBACWarn(perm, c.Request.URL.Path)
			if permLogger != nil {
				permLogger.Warn("RBAC warn: permission denied (request allowed)",
					zap.Uint("user_id", uid),
					zap.String("path", c.Request.URL.Path),
					zap.String("required_perm", perm),
					zap.String("role", roleStr),
				)
			}
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{
			"code":    10200,
			"message": "insufficient permissions: " + perm,
		})
		c.Abort()
	}
}
