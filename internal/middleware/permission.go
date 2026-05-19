package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/pkg/rbac"
)

// RequirePerm returns a middleware that checks if the user's effective permissions
// (global role + team roles) include the required permission.
// For team-scoped endpoints, team roles can elevate permissions beyond the global role.
func RequirePerm(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextKeyRole)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    10200,
				"message": "forbidden",
			})
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok {
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

		c.JSON(http.StatusForbidden, gin.H{
			"code":    10200,
			"message": "insufficient permissions: " + perm,
		})
		c.Abort()
	}
}
