import { ref, computed } from 'vue'
import { permissionsApi } from '@/api'
import type { MyPermissions, TeamRole } from '@/api/center'

const permissions = ref<MyPermissions | null>(null)
const loaded = ref(false)

export function usePermissions() {
  async function loadPermissions() {
    try {
      const { data } = await permissionsApi.getMy()
      permissions.value = data.data
    } catch {
      permissions.value = null
    } finally {
      loaded.value = true
    }
  }

  function hasPerm(perm: string): boolean {
    if (!permissions.value) return false
    return permissions.value.perms.includes(perm)
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
