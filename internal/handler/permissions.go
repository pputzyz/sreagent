package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
	"github.com/sreagent/sreagent/internal/pkg/rbac"
	"github.com/sreagent/sreagent/internal/service"
)

// PermissionsHandler handles GET /me/permissions.
type PermissionsHandler struct {
	teamSvc *service.TeamService
}

func NewPermissionsHandler(teamSvc *service.TeamService) *PermissionsHandler {
	return &PermissionsHandler{teamSvc: teamSvc}
}

// GetMyPermissions handles GET /me/permissions — returns the current user's full permission set.
func (h *PermissionsHandler) GetMyPermissions(c *gin.Context) {
	uid, ok := GetCurrentUserIDOK(c)
	if !ok {
		Success(c, gin.H{"role": "", "teams": []gin.H{}})
		return
	}

	role, _ := c.Get(middleware.ContextKeyRole)
	roleStr, _ := role.(string)

	// Get team-level roles and merge with global role
	teams := h.getTeamRoles(c, uid)
	teamRoles := make([]string, 0, len(teams))
	for _, t := range teams {
		if r, ok := t["role"].(string); ok {
			teamRoles = append(teamRoles, r)
		}
	}

	// Build merged permission list (global role + team roles)
	effectivePerms := rbac.EffectivePerms(roleStr, teamRoles)
	perms := make([]string, 0, len(effectivePerms))
	for p := range effectivePerms {
		perms = append(perms, p)
	}

	Success(c, gin.H{
		"role":  roleStr,
		"perms": perms,
		"teams": teams,
	})
}

func (h *PermissionsHandler) getTeamRoles(c *gin.Context, userID uint) []gin.H {
	teams, err := h.teamSvc.ListByUser(c.Request.Context(), userID)
	if err != nil {
		return []gin.H{}
	}
	result := make([]gin.H, 0, len(teams))
	for _, t := range teams {
		result = append(result, gin.H{
			"team_id": t.TeamID,
			"role":    t.Role,
		})
	}
	return result
}
