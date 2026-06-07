// Package rbac provides role-based access control helpers.
package rbac

// PermissionsByGlobalRole returns the permission set for a given global role.
func PermissionsByGlobalRole(role string) map[string]bool {
	switch role {
	case "admin":
		return map[string]bool{
			"users.manage": true, "teams.manage": true, "roles.view": true,
			"rules.manage": true, "rules.create": true, "rules.edit": true, "rules.delete": true,
			"rules.write": true,
			"events.manage": true, "events.ack": true, "events.assign": true,
			"schedules.manage": true, "schedule.write": true, "escalation.write": true,
			"channels.manage": true, "pipeline.write": true,
			"mute.write": true, "inhibition.write": true,
			"notify.write": true, "channels.write": true, "dispatch.write": true,
			"datasource.write": true, "integration.write": true,
			"team.write": true, "user.write": true,
			"settings.manage": true, "audit.view": true,
			"datasources.manage": true, "dashboards.manage": true,
			"incidents.manage": true, "incidents.create": true, "incident.write": true,
			"notifications.view": true, "todos.manage": true,
			"recording.write": true, "template.write": true,
			"mcp.write": true, "skill.write": true, "llm.write": true,
			"task.write": true, "inspection.write": true,
		}
	case "team_lead":
		return map[string]bool{
			"teams.manage": true,
			"rules.manage": true, "rules.create": true, "rules.edit": true,
			"rules.write": true,
			"events.manage": true, "events.ack": true, "events.assign": true,
			"schedules.manage": true, "schedule.write": true, "escalation.write": true,
			"channels.manage": true, "pipeline.write": true,
			"mute.write": true, "inhibition.write": true,
			"notify.write": true, "channels.write": true, "dispatch.write": true,
			"datasources.view": true, "dashboards.manage": true,
			"incidents.manage": true, "incidents.create": true, "incident.write": true,
			"notifications.view": true, "todos.manage": true,
			"recording.write": true, "template.write": true,
			"mcp.write": true, "skill.write": true, "llm.write": true,
			"task.write": true, "inspection.write": true,
		}
	case "member":
		return map[string]bool{
			"rules.view": true, "rules.create": true,
			"events.ack": true, "events.assign": true,
			"schedules.view": true, "channels.view": true,
			"datasources.view": true, "dashboards.view": true,
			"incidents.view": true, "incidents.create": true,
			"notifications.view": true, "todos.manage": true,
		}
	case "viewer", "global_viewer":
		return map[string]bool{
			"rules.view": true, "events.view": true,
			"schedules.view": true, "channels.view": true,
			"datasources.view": true, "dashboards.view": true,
			"incidents.view": true,
			"notifications.view": true, "todos.view": true,
		}
	default:
		return map[string]bool{"notifications.view": true, "todos.view": true}
	}
}

// HasPerm checks if the given global role grants the specified permission.
func HasPerm(role, perm string) bool {
	return PermissionsByGlobalRole(role)[perm]
}

// RoleLevel returns a numeric level for role comparison (higher = more privileged).
func RoleLevel(role string) int {
	switch role {
	case "admin":
		return 4
	case "team_lead":
		return 3
	case "member":
		return 2
	case "viewer", "global_viewer":
		return 1
	default:
		return 0
	}
}

// HighestTeamRole returns the highest role from a list of team roles.
// Returns empty string if no roles provided.
func HighestTeamRole(teamRoles []string) string {
	best := ""
	bestLevel := 0
	for _, r := range teamRoles {
		if lvl := RoleLevel(r); lvl > bestLevel {
			bestLevel = lvl
			best = r
		}
	}
	return best
}

// EffectivePerms returns the merged permission set considering both global role
// and team-level roles. Team roles can only elevate permissions (not restrict).
func EffectivePerms(globalRole string, teamRoles []string) map[string]bool {
	perms := PermissionsByGlobalRole(globalRole)
	highestTeam := HighestTeamRole(teamRoles)
	if highestTeam == "" {
		return perms
	}
	teamPerms := PermissionsByGlobalRole(highestTeam)
	for k, v := range teamPerms {
		if v {
			perms[k] = true
		}
	}
	return perms
}
