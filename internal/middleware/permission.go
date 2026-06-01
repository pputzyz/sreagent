package middleware

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/pkg/metrics"
	"github.com/sreagent/sreagent/internal/pkg/rbac"
)

// EnforceMode controls the behaviour of RequirePerm when a permission check fails.
//   - "deny" (default): return 403 and abort the request.
//   - "warn": log the denial but allow the request through (for gradual rollout).
var enforceMode atomic.Value

func init() {
	enforceMode.Store("deny")
}

// warnModeActivatedAt tracks when warn mode was last activated.
var (
	warnModeMu          sync.RWMutex
	warnModeActivatedAt time.Time
	lastWarnLongLog     atomic.Int64
)

// SetEnforceMode sets the RBAC enforce mode. Valid values: "warn", "deny".
func SetEnforceMode(mode string) {
	if mode == "warn" || mode == "deny" {
		enforceMode.Store(mode)
		warnModeMu.Lock()
		if mode == "warn" {
			warnModeActivatedAt = time.Now()
		} else {
			warnModeActivatedAt = time.Time{}
		}
		warnModeMu.Unlock()
	}
}

// getEnforceMode returns the current enforce mode.
func getEnforceMode() string {
	return enforceMode.Load().(string)
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

// CheckRBACWarnModeInRelease should be called at startup. It logs a loud warning
// if the RBAC enforce mode is "warn" while running in release mode.
func CheckRBACWarnModeInRelease() {
	if gin.Mode() == gin.ReleaseMode && getEnforceMode() == "warn" {
		if permLogger != nil {
			permLogger.Error("SECURITY WARNING: RBAC enforce mode is 'warn' in release mode — " +
				"permission checks are logged but NOT enforced. " +
				"Set enforce mode to 'deny' before deploying to production.")
		}
	}
}

// logWarnModeLongActive logs a warning if warn mode has been active for over 1 hour.
// Rate-limited to at most once per minute to avoid log spam.
func logWarnModeLongActive(path string) {
	warnModeMu.RLock()
	activatedAt := warnModeActivatedAt
	warnModeMu.RUnlock()
	if activatedAt.IsZero() || time.Since(activatedAt) <= time.Hour {
		return
	}
	now := time.Now().Unix()
	last := lastWarnLongLog.Load()
	if now-last < 60 {
		return
	}
	if lastWarnLongLog.CompareAndSwap(last, now) {
		if permLogger != nil {
			permLogger.Warn("RBAC warn mode has been active for over 1 hour — requests are NOT being denied",
				zap.Time("activated_at", activatedAt),
				zap.Duration("active_for", time.Since(activatedAt)),
				zap.String("path", path),
			)
		}
	}
}

// RequirePerm returns a middleware that checks if the user's effective permissions
// (global role + team roles) include the required permission.
// For team-scoped endpoints, team roles can elevate permissions beyond the global role.
func RequirePerm(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextKeyRole)
		if !exists {
			if getEnforceMode() == "warn" {
				logWarnModeLongActive(c.Request.URL.Path)
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
			if getEnforceMode() == "warn" {
				logWarnModeLongActive(c.Request.URL.Path)
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
				"message": "forbidden",
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

		if getEnforceMode() == "warn" {
			logWarnModeLongActive(c.Request.URL.Path)
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

		// #16: Log permission name server-side; do not expose to client.
		if permLogger != nil {
			permLogger.Warn("permission denied",
				zap.Uint("user_id", uid),
				zap.String("path", c.Request.URL.Path),
				zap.String("required_perm", perm),
				zap.String("role", roleStr),
			)
		}
		c.JSON(http.StatusForbidden, gin.H{
			"code":    10200,
			"message": "insufficient permissions",
		})
		c.Abort()
	}
}
