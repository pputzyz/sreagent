import axios from 'axios'
import type { ApiResponse } from '@/types'
import i18n from '@/i18n'

const t = i18n.global.t

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
})

// Map backend error codes to i18n keys
const errorCodeMap: Record<number, string> = {
  10102: 'errorCode.invalidCredentials',
  10101: 'errorCode.sessionExpired',
  10100: 'errorCode.unauthorized',
  10200: 'errorCode.insufficientPermissions',
  10300: 'errorCode.resourceNotFound',
  10400: 'errorCode.nameTaken',
}

function localizeError(code: number, fallback: string): string {
  const key = errorCodeMap[code]
  if (!key) return fallback
  return t(key)
}

// Request interceptor - attach JWT token
request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) config.headers.Authorization = `Bearer ${token}`
    return config
  },
  (error) => Promise.reject(error)
)

// Prevent multiple simultaneous 401 redirects / refresh attempts
const REDIRECT_DEBOUNCE_MS = 2000
let isRedirecting = false
let refreshPromise: Promise<string> | null = null

function redirectToLogin() {
  if (isRedirecting) return
  isRedirecting = true
  localStorage.removeItem('token')
  localStorage.removeItem('user_role')
  import('@/router').then(({ default: router }) => {
    router.push({ name: 'Login', query: { redirect: router.currentRoute.value.fullPath } })
  }).finally(() => {
    setTimeout(() => { isRedirecting = false }, REDIRECT_DEBOUNCE_MS)
  })
}

// Response interceptor — auto-refresh token on 401 before giving up
request.interceptors.response.use(
  (response) => {
    const data = response.data as ApiResponse
    if (data.code !== 0) {
      const msg = localizeError(data.code, data.message || 'Unknown error')
      return Promise.reject(new Error(msg))
    }
    return response
  },
  async (error) => {
    const originalRequest = error.config
    const data = error.response?.data as ApiResponse | undefined
    const code = data?.code || 0

    if (error.response?.status === 401 && !originalRequest._retried) {
      // If the backend returned a specific error code (e.g. 10102 invalid credentials),
      // surface the localized message directly — don't attempt token refresh.
      if (code && code !== 10101) {
        const fallback = data?.message || error.message || 'Unauthorized'
        return Promise.reject(new Error(localizeError(code, fallback)))
      }

      originalRequest._retried = true
      const storedToken = localStorage.getItem('token')
      if (storedToken && !isRedirecting) {
        try {
          // Deduplicate concurrent refresh calls
          if (!refreshPromise) {
            refreshPromise = (async () => {
              const res = await axios.post('/api/v1/auth/refresh', { token: storedToken })
              const newToken: string = res.data?.data?.token
              if (!newToken) throw new Error('empty token')
              return newToken
            })().finally(() => { refreshPromise = null })
          }
          const newToken = await refreshPromise
          localStorage.setItem('token', newToken)
          originalRequest.headers.Authorization = `Bearer ${newToken}`
          return request(originalRequest)
        } catch {
          redirectToLogin()
          return Promise.reject(error)
        }
      }
      redirectToLogin()
      return Promise.reject(error)
    }
    const fallback = data?.message || error.message || 'Network error'
    return Promise.reject(new Error(localizeError(code, fallback)))
  }
)

export default request
