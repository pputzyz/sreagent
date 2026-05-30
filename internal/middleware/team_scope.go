package middleware

import (
	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyUserTeamIDs is the key for user's team IDs in gin context.
	ContextKeyUserTeamIDs = "user_team_ids"
)

// TeamIDQuerier abstracts the team repository for querying user team membership.
// Injected at startup to avoid circular imports between middleware and repository.
var TeamIDQuerier interface {
	// ListUserTeamIDs returns the team IDs the given user belongs to.
	ListUserTeamIDs(userID uint) ([]uint, error)
}

// GetUserTeamIDs extracts the user's team IDs from the gin context.
// Returns nil if no team IDs are set (e.g. admin bypass or middleware not loaded).
func GetUserTeamIDs(c *gin.Context) []uint {
	if raw, exists := c.Get(ContextKeyUserTeamIDs); exists {
		if ids, ok := raw.([]uint); ok {
			return ids
		}
	}
	return nil
}

// TeamScoped returns a middleware that loads the authenticated user's team IDs
// into the gin context on each request. Must run after JWTAuth.
//
// Admin users still get their team IDs loaded (for audit / display purposes),
// but service-layer ListScoped methods skip filtering for admins.
func TeamScoped() gin.HandlerFunc {
	return func(c *gin.Context) {
		if TeamIDQuerier == nil {
			c.Next()
			return
		}

		userIDRaw, exists := c.Get(ContextKeyUserID)
		if !exists {
			c.Next()
			return
		}
		userID, ok := userIDRaw.(uint)
		if !ok || userID == 0 {
			c.Next()
			return
		}

		teamIDs, err := TeamIDQuerier.ListUserTeamIDs(userID)
		if err != nil {
			// Log but don't block — fallback to no team filtering.
			// The logger may not be in context yet; use zap directly.
			c.Next()
			return
		}

		if len(teamIDs) > 0 {
			c.Set(ContextKeyUserTeamIDs, teamIDs)
		}

		c.Next()
	}
}
