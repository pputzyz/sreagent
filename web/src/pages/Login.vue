<script setup lang="ts">
import { ref, computed, inject, onMounted, watch } from 'vue'
import type { Ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import { getErrorMessage } from '@/utils/format'
import { useI18n } from 'vue-i18n'
import { authApi } from '@/api'
import { SunnyOutline, MoonOutline } from '@vicons/ionicons5'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const authStore = useAuthStore()
const { t, locale } = useI18n()
const appVersion = __APP_VERSION__

const isDark = inject<Ref<boolean>>('isDark', ref(true))
const toggleTheme = inject<() => void>('toggleTheme', () => {})

const form = ref({ username: '', password: '' })
const loading = ref(false)
const loginError = ref('')

const oidcEnabled = ref(false)
const oidcLoginUrl = ref('')
const oidcLoading = ref(false)

const langOptions = computed(() => [
  { label: t('language.zh'), value: 'zh-CN' },
  { label: t('language.en'), value: 'en' },
])

function handleLangChange(val: string) {
  locale.value = val
  localStorage.setItem('locale', val)
}

async function handleLogin() {
  loginError.value = ''
  if (!form.value.username || !form.value.password) {
    loginError.value = t('auth.pleaseEnter')
    return
  }
  loading.value = true
  try {
    await authStore.login(form.value.username, form.value.password)
    message.success(t('auth.loginSuccess'))
    const raw = (route.query.redirect as string) || ''
    const safeRedirect = raw.startsWith('/') && !raw.startsWith('//') ? raw : '/'
    router.push(safeRedirect)
  } catch (err: unknown) {
    loginError.value = getErrorMessage(err) || t('auth.loginFailed')
  } finally {
    loading.value = false
  }
}

function handleSSOLogin() {
  if (oidcLoginUrl.value) {
    oidcLoading.value = true
    window.location.href = oidcLoginUrl.value
  }
}

async function checkOIDCConfig() {
  try {
    const { data } = await authApi.getOIDCConfig()
    if (data.data.enabled && data.data.login_url) {
      oidcEnabled.value = true
      oidcLoginUrl.value = data.data.login_url
    }
  } catch {
    /* OIDC not configured */
  }
}

onMounted(() => {
  checkOIDCConfig()
})

watch([() => form.value.username, () => form.value.password], () => {
  if (loginError.value) loginError.value = ''
})
</script>

<template>
  <div class="login-layout" :class="{ light: !isDark }">
    <!-- Top right controls -->
    <div class="login-controls">
      <n-select
        :value="locale"
        :options="langOptions"
        size="small"
        style="width: 116px"
        @update:value="handleLangChange"
      />
      <n-button text @click="toggleTheme" style="padding: 4px 8px">
        <n-icon :component="isDark ? SunnyOutline : MoonOutline" :size="18" />
      </n-button>
    </div>

    <!-- Centered card -->
    <div class="login-center">
      <div class="login-card">
        <!-- Brand header inside card -->
        <div class="card-brand">
          <div class="brand-logo-row">
            <img src="/logo.svg" alt="SREAgent" class="brand-logo" />
            <span class="brand-name">SREAgent</span>
          </div>
          <p class="brand-tagline">{{ t('auth.brand.tagline') }}</p>
        </div>

        <!-- Form -->
        <form class="login-form" @submit.prevent="handleLogin">
          <header class="form-header">
            <h2 class="form-title">{{ t('auth.signIn') }}</h2>
            <p class="form-subtitle">{{ t('auth.welcomeBack') }}</p>
          </header>

          <label class="field">
            <span class="field-label">{{ t('auth.username') }}</span>
            <n-input
              v-model:value="form.username"
              :placeholder="t('auth.enterUsername')"
              size="large"
              :autofocus="true"
            />
          </label>

          <label class="field">
            <span class="field-label">{{ t('auth.password') }}</span>
            <n-input
              v-model:value="form.password"
              type="password"
              :placeholder="t('auth.enterPassword')"
              size="large"
              show-password-on="click"
              @keyup.enter="handleLogin"
            />
          </label>

          <n-button
            type="primary"
            block
            size="large"
            :loading="loading"
            class="submit-btn"
            @click="handleLogin"
          >
            {{ t('auth.signIn') }} &rarr;
          </n-button>

          <div v-if="loginError" class="error-banner">
            <span class="error-mark">!</span>
            <span>{{ loginError }}</span>
          </div>

          <template v-if="oidcEnabled">
            <div class="form-divider">{{ t('auth.orContinueWith') }}</div>
            <n-button
              block
              size="large"
              quaternary
              :loading="oidcLoading"
              class="sso-btn"
              @click="handleSSOLogin"
            >
              {{ t('auth.ssoLogin') }}
            </n-button>
          </template>

          <p class="form-default-hint">{{ t('auth.defaultHint') }}</p>
        </form>

        <!-- Footer -->
        <div class="card-footer">
          <span class="footer-version">v{{ appVersion }}</span>
          <span class="footer-status">
            <span class="status-dot" /> {{ t('auth.brand.systemStatus') }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-layout {
  min-height: 100vh;
  position: relative;
  overflow: hidden;
  font-family: var(--sre-font-sans);
  color: var(--sre-text-primary);
  background: var(--sre-bg-base);
}

/* ===== Mesh Blobs — warmer palette ===== */
.login-layout::before {
  content: '';
  position: absolute;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background:
    radial-gradient(ellipse 50% 50% at 15% 20%, rgba(13,148,136,0.12), transparent 60%),
    radial-gradient(ellipse 50% 50% at 80% 15%, rgba(20,184,166,0.10), transparent 55%),
    radial-gradient(ellipse 45% 45% at 25% 80%, rgba(6,182,212,0.10), transparent 55%),
    radial-gradient(ellipse 45% 50% at 75% 70%, rgba(139,92,246,0.08), transparent 55%);
  animation: blob-drift 20s ease-in-out infinite alternate;
}

@keyframes blob-drift {
  0%   { transform: translate(0, 0) scale(1); }
  33%  { transform: translate(20px, -15px) scale(1.03); }
  66%  { transform: translate(-15px, 10px) scale(0.98); }
  100% { transform: translate(10px, -5px) scale(1.01); }
}

/* ===== Controls ===== */
.login-controls {
  position: absolute;
  top: 20px;
  right: 24px;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* ===== Centered card ===== */
.login-center {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 24px;
}

.login-card {
  width: 100%;
  max-width: 420px;
  position: relative;
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 16px;
  padding: 40px 36px 28px;
  box-shadow: var(--sre-shadow-lg);
  animation: card-in 400ms var(--sre-ease-out) both;
  background-clip: padding-box;
}

.login-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  border-radius: 16px 16px 0 0;
  background: linear-gradient(135deg, #0D9488, #14B8A6, #06B6D4);
}

@keyframes card-in {
  from { opacity: 0; transform: translateY(16px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* ===== Brand inside card ===== */
.card-brand {
  text-align: center;
  margin-bottom: 32px;
  padding-bottom: 28px;
  border-bottom: 1px solid var(--sre-border);
}

.brand-logo-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  margin-bottom: 10px;
}

.brand-logo {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  filter: drop-shadow(0 0 12px rgba(13, 148, 136, 0.3));
  animation: brand-logo-breathe 4s ease-in-out infinite;
}

@keyframes brand-logo-breathe {
  0%, 100% { filter: drop-shadow(0 0 12px rgba(13, 148, 136, 0.2)); }
  50% { filter: drop-shadow(0 0 18px rgba(13, 148, 136, 0.45)); }
}

.brand-name {
  font-family: var(--sre-font-display);
  font-size: 28px;
  font-weight: 700;
  letter-spacing: -1px;
}

.brand-tagline {
  font-size: 14px;
  color: var(--sre-text-secondary);
  margin: 0;
  line-height: 1.5;
}

/* ===== Form ===== */
.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.login-form > * {
  opacity: 0;
  animation: form-rise 480ms var(--sre-ease-out) forwards;
}
.login-form > *:nth-child(1) { animation-delay: 120ms; }
.login-form > *:nth-child(2) { animation-delay: 180ms; }
.login-form > *:nth-child(3) { animation-delay: 240ms; }
.login-form > *:nth-child(4) { animation-delay: 300ms; }
.login-form > *:nth-child(5) { animation-delay: 360ms; }
.login-form > *:nth-child(6) { animation-delay: 420ms; }
.login-form > *:nth-child(7) { animation-delay: 480ms; }
.login-form > *:nth-child(8) { animation-delay: 540ms; }

@keyframes form-rise {
  from { opacity: 0; transform: translateY(8px); }
  to   { opacity: 1; transform: translateY(0); }
}

.form-header {
  margin-bottom: 4px;
}
.form-title {
  font-size: 22px;
  font-weight: 700;
  letter-spacing: -0.3px;
  margin: 0 0 4px;
  color: var(--sre-text-primary);
}
.form-subtitle {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin: 0;
  line-height: 1.5;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--sre-text-secondary);
}

.login-form :deep(.n-input--focus) {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--sre-primary) 20%, transparent);
  transition: box-shadow 200ms var(--sre-ease-out);
}

.login-form :deep(.n-input) {
  transition: box-shadow 200ms var(--sre-ease-out);
}

.submit-btn {
  height: 44px;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.2px;
  margin-top: 4px;
  background: linear-gradient(135deg, #0D9488, #14B8A6) !important;
  border: none !important;
  transition:
    box-shadow var(--sre-duration-base) var(--sre-ease-out),
    transform 100ms var(--sre-ease-out),
    background 200ms var(--sre-ease-out);
}
.submit-btn:hover {
  box-shadow: var(--sre-shadow-md);
  background: linear-gradient(135deg, #0F766E, #0D9488) !important;
}
.submit-btn:active {
  transform: scale(0.97);
}

.error-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: var(--sre-radius-sm);
  background: var(--sre-critical-soft);
  border: 1px solid color-mix(in srgb, var(--sre-critical) 30%, transparent);
  font-size: 12px;
  color: var(--sre-critical);
  line-height: 1.4;
}
.error-mark {
  flex: 0 0 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--sre-critical);
  color: var(--sre-text-inverse);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  font-family: var(--sre-font-mono);
}

.form-divider {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: var(--sre-text-tertiary);
  margin: 4px 0;
}
.form-divider::before,
.form-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--sre-border);
}

.sso-btn {
  height: 44px;
  font-size: 13px;
  font-weight: 500;
  border: 1px solid var(--sre-border-strong);
}

.form-default-hint {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  text-align: center;
  margin: 8px 0 0;
  line-height: 1.5;
}

/* ===== Card footer ===== */
.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid var(--sre-border);
  font-size: 11px;
  color: var(--sre-text-tertiary);
}

.footer-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--sre-warning, #F59E0B);
  box-shadow: 0 0 8px rgba(245, 158, 11, 0.5);
  animation: status-pulse 2s ease-in-out infinite;
}

@keyframes status-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(245, 158, 11, 0.4); }
  50%      { box-shadow: 0 0 0 6px rgba(245, 158, 11, 0); }
}

/* ===== Light mode adjustments ===== */
.login-layout.light .login-card {
  background: var(--sre-bg-card);
  box-shadow: var(--sre-shadow-lg);
}

/* ===== Responsive ===== */
@media (max-width: 520px) {
  .login-card {
    padding: 28px 20px 20px;
    border-radius: 16px;
  }
  .brand-name { font-size: 24px; }
  .brand-logo { width: 28px; height: 28px; }
}

@media (prefers-reduced-motion: reduce) {
  .brand-logo {
    animation: none;
  }
  .submit-btn:active {
    transform: none;
  }
  .login-form :deep(.n-input--focus) {
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--sre-primary) 25%, transparent);
  }
}
</style>
