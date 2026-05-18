import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import type { User } from '@/types'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem('token') || '')
  const user = ref<User | null>(null)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  /** admin or team_lead — can manage alert rules, schedules, notification config, teams */
  const canManage = computed(() => ['admin', 'team_lead'].includes(user.value?.role || ''))
  /** admin, team_lead, or member — can operate on alert events (ack, assign, etc.) */
  const canOperate = computed(() => ['admin', 'team_lead', 'member'].includes(user.value?.role || ''))

  /** Standard username/password login */
  async function login(username: string, password: string) {
    const { data } = await authApi.login({ username, password })
    token.value = data.data.token
    localStorage.setItem('token', data.data.token)
    await fetchProfile()
  }

  /** Accept a token from OIDC callback redirect */
  function setToken(oidcToken: string) {
    token.value = oidcToken
    localStorage.setItem('token', oidcToken)
  }

  async function fetchProfile() {
    try {
      const { data } = await authApi.getProfile()
      user.value = data.data
      // Persist role for route guard (sync check before store is hydrated)
      if (data.data?.role) {
        localStorage.setItem('user_role', data.data.role)
      }
    } catch (err: unknown) {
      // Only logout on 401 (invalid/expired token), not on network/5xx errors
      const msg = getErrorMessage(err)
      if (msg.includes('401')) {
        logout()
      }
      // For network errors, keep the session — the user may reconnect
    }
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user_role')
  }

  return {
    token,
    user,
    isLoggedIn,
    isAdmin,
    canManage,
    canOperate,
    login,
    setToken,
    fetchProfile,
    logout,
  }
})
