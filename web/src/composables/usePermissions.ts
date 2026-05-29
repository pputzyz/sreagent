import { ref, computed } from 'vue'
import { permissionsApi } from '@/api'
import type { MyPermissions, TeamRole } from '@/api/center'

const permissions = ref<MyPermissions | null>(null)
const loaded = ref(false)

/** Reset module-level singleton state (call on logout) */
export function resetPermissions() {
  permissions.value = null
  loaded.value = false
}

// Role-based fallback — mirrors internal/pkg/rbac/rbac.go PermissionsByGlobalRole
const roleFallbackPerms: Record<string, string[]> = {
  admin: [
    'users.manage', 'teams.manage', 'roles.view',
    'rules.manage', 'rules.create', 'rules.edit', 'rules.delete', 'rules.write',
    'events.manage', 'events.ack', 'events.assign',
    'schedules.manage', 'channels.manage',
    'mute.write', 'inhibition.write',
    'notify.write', 'channels.write', 'dispatch.write',
    'datasource.write', 'integration.write',
    'team.write', 'user.write',
    'settings.manage', 'audit.view',
    'datasources.manage', 'dashboards.manage',
    'incidents.manage', 'incidents.create',
    'notifications.view', 'todos.manage',
    'metrics.write', 'metrics.manage',
  ],
  team_lead: [
    'teams.manage',
    'rules.manage', 'rules.create', 'rules.edit', 'rules.write',
    'events.manage', 'events.ack', 'events.assign',
    'schedules.manage', 'channels.manage',
    'mute.write', 'inhibition.write',
    'notify.write', 'channels.write', 'dispatch.write',
    'datasources.view', 'dashboards.manage',
    'incidents.manage', 'incidents.create',
    'notifications.view', 'todos.manage',
    'metrics.write', 'metrics.manage',
  ],
  member: [
    'rules.view', 'rules.create',
    'events.ack', 'events.assign',
    'schedules.view', 'channels.view',
    'datasources.view', 'dashboards.view',
    'incidents.view', 'incidents.create',
    'notifications.view', 'todos.manage',
  ],
}

export function usePermissions() {
  async function loadPermissions() {
    try {
      const { data } = await permissionsApi.getMy()
      permissions.value = data.data
    } catch (err) {
      console.warn('[usePermissions] Failed to load permissions:', err)
      permissions.value = null
    } finally {
      loaded.value = true
    }
  }

  function hasPerm(perm: string): boolean {
    // Fast path: use loaded permissions from API
    if (permissions.value) return permissions.value.perms.includes(perm)
    // Fallback: infer from localStorage role (set by fetchProfile)
    const role = localStorage.getItem('user_role') || ''
    return (roleFallbackPerms[role] || []).includes(perm)
  }

  function hasAnyPerm(...perms: string[]): boolean {
    return perms.some(p => hasPerm(p))
  }

  function isTeamRole(teamId: number, role: string): boolean {
    if (!permissions.value) return false
    return permissions.value.teams.some((t: TeamRole) => t.team_id === teamId && t.role === role)
  }

  function isTeamLead(teamId: number): boolean {
    return isTeamRole(teamId, 'lead')
  }

  const globalRole = computed(() => permissions.value?.role || '')
  const teamRoles = computed(() => permissions.value?.teams || [])

  return {
    permissions,
    loaded,
    loadPermissions,
    hasPerm,
    hasAnyPerm,
    isTeamRole,
    isTeamLead,
    globalRole,
    teamRoles,
  }
}
