package handler

import (
	"github.com/gin-gonic/gin"

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

	role, _ := c.Get("user_role")
	roleStr, _ := role.(string)

	// Build permission list based on global role
	perms := buildPermissions(roleStr)

	// Get team-level roles
	teams := h.getTeamRoles(c, uid)

	Success(c, gin.H{
		"role":   roleStr,
		"perms":  perms,
		"teams":  teams,
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

func buildPermissions(role string) []string {
	switch role {
	case "admin":
		return []string{
			"users.manage", "teams.manage", "roles.view",
			"rules.manage", "rules.create", "rules.edit", "rules.delete",
			"events.manage", "events.ack", "events.assign",
			"schedules.manage", "channels.manage",
			"settings.manage", "audit.view",
			"datasources.manage", "dashboards.manage",
			"incidents.manage", "incidents.create",
			"notifications.view", "todos.manage",
		}
	case "team_lead":
		return []string{
			"teams.manage",
			"rules.manage", "rules.create", "rules.edit",
			"events.manage", "events.ack", "events.assign",
			"schedules.manage", "channels.manage",
			"datasources.view", "dashboards.manage",
			"incidents.manage", "incidents.create",
			"notifications.view", "todos.manage",
		}
	case "member":
		return []string{
			"rules.view", "rules.create",
			"events.ack", "events.assign",
			"schedules.view", "channels.view",
			"datasources.view", "dashboards.view",
			"incidents.view", "incidents.create",
			"notifications.view", "todos.manage",
		}
	case "viewer", "global_viewer":
		return []string{
			"rules.view", "events.view",
			"schedules.view", "channels.view",
			"datasources.view", "dashboards.view",
			"incidents.view",
			"notifications.view", "todos.view",
		}
	default:
		return []string{"notifications.view", "todos.view"}
	}
}
