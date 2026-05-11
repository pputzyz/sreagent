<script setup lang="ts">
import { ref, inject, onMounted, watch } from 'vue'
import type { Ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
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

const langOptions = [
  { label: t('language.zh'), value: 'zh-CN' },
  { label: t('language.en'), value: 'en' },
]

function handleLangChange(val: string) {
  locale.value = val
  localStorage.setItem('locale', val)
}

async function handleLogin() {
  loginError.value = ''
  if (!form.value.username || !form.value.password) {
    loginError.value = t('auth.pleaseEnter') || 'Please enter username and password'
    return
  }
  loading.value = true
  try {
    await authStore.login(form.value.username, form.value.password)
    message.success(t('auth.loginSuccess'))
    router.push((route.query.redirect as string) || '/oncall/overview')
  } catch (err: any) {
    loginError.value = err.message || t('auth.loginFailed')
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
    <!-- Animated mesh background -->
    <div class="mesh-bg">
      <div class="mesh-blob mesh-blob--teal" />
      <div class="mesh-blob mesh-blob--blue" />
      <div class="mesh-blob mesh-blob--amber" />
    </div>

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

    <!-- Centered glass card -->
    <div class="login-center">
      <div class="login-card">
        <!-- Brand header inside card -->
        <div class="card-brand">
          <div class="brand-logo-row">
            <img src="/logo.svg" alt="SREAgent" class="brand-logo" />
            <span class="brand-name"><span class="gradient-text">SRE</span>Agent</span>
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
              :placeholder="t('auth.enterUsername') || 'Enter username'"
              size="large"
              :autofocus="true"
            />
          </label>

          <label class="field">
            <span class="field-label">{{ t('auth.password') }}</span>
            <n-input
              v-model:value="form.password"
              type="password"
              :placeholder="t('auth.enterPassword') || 'Enter password'"
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
            <div class="form-divider">{{ t('auth.orContinueWith') || 'or' }}</div>
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
}

/* ===== Animated mesh background ===== */
.mesh-bg {
  position: fixed;
  inset: 0;
  z-index: 0;
  background: var(--sre-bg-base);
  overflow: hidden;
}

.mesh-blob {
  position: absolute;
  border-radius: 50%;
  filter: blur(100px);
  opacity: 0.4;
  animation: mesh-float 20s ease-in-out infinite;
}

.mesh-blob--teal {
  width: 500px;
  height: 500px;
  background: rgba(13, 148, 136, 0.35);
  top: -10%;
  left: -5%;
  animation-delay: 0s;
}

.mesh-blob--blue {
  width: 400px;
  height: 400px;
  background: rgba(59, 130, 246, 0.25);
  top: 50%;
  right: -8%;
  animation-delay: -7s;
  animation-duration: 25s;
}

.mesh-blob--amber {
  width: 350px;
  height: 350px;
  background: rgba(245, 158, 11, 0.20);
  bottom: -10%;
  left: 30%;
  animation-delay: -14s;
  animation-duration: 22s;
}

@keyframes mesh-float {
  0%, 100% { transform: translate(0, 0) scale(1); }
  25%      { transform: translate(60px, -40px) scale(1.1); }
  50%      { transform: translate(-30px, 50px) scale(0.95); }
  75%      { transform: translate(40px, 20px) scale(1.05); }
}

.login-layout.light .mesh-blob--teal { background: rgba(13, 148, 136, 0.15); }
.login-layout.light .mesh-blob--blue { background: rgba(59, 130, 246, 0.10); }
.login-layout.light .mesh-blob--amber { background: rgba(245, 158, 11, 0.10); }
.login-layout.light .mesh-blob { opacity: 0.5; }

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

/* ===== Centered glass card ===== */
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
  background: var(--sre-glass-bg);
  -webkit-backdrop-filter: blur(24px) saturate(160%);
  backdrop-filter: blur(24px) saturate(160%);
  border: 1px solid var(--sre-glass-border);
  border-radius: 20px;
  padding: 40px 36px 28px;
  box-shadow:
    0 24px 80px -12px rgba(0, 0, 0, 0.50),
    0 0 0 1px rgba(148, 163, 184, 0.06),
    0 0 60px -20px rgba(13, 148, 136, 0.15);
  animation: card-rise 600ms var(--sre-ease-out) both;
}

@keyframes card-rise {
  from { opacity: 0; transform: translateY(24px) scale(0.97); }
  to   { opacity: 1; transform: translateY(0) scale(1); }
}

/* ===== Brand inside card ===== */
.card-brand {
  text-align: center;
  margin-bottom: 32px;
  padding-bottom: 28px;
  border-bottom: 1px solid var(--sre-glass-border);
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
}

.brand-name {
  font-family: var(--sre-font-display);
  font-size: 28px;
  font-weight: 800;
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
  from { opacity: 0; transform: translateY(10px); }
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
  font-family: var(--sre-font-mono);
  font-size: 11px;
  letter-spacing: 1px;
  text-transform: uppercase;
  color: var(--sre-text-tertiary);
}

.submit-btn {
  height: 44px;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.2px;
  margin-top: 4px;
  transition: box-shadow var(--sre-duration-base) var(--sre-ease-out),
              transform var(--sre-duration-base) var(--sre-ease-out);
}
.submit-btn:hover {
  box-shadow: 0 8px 24px -8px var(--sre-primary-ring), var(--sre-shadow-glow);
}
.submit-btn:active {
  transform: translateY(1px);
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
  color: #fafaf9;
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
  font-family: var(--sre-font-mono);
  font-size: 11px;
  color: var(--sre-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 1.2px;
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
  border-top: 1px solid var(--sre-glass-border);
  font-family: var(--sre-font-mono);
  font-size: 10px;
  color: var(--sre-text-tertiary);
  letter-spacing: 0.5px;
}

.footer-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  text-transform: uppercase;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--sre-success);
  box-shadow: 0 0 8px var(--sre-success);
  animation: status-pulse 2.4s ease-out infinite;
}

@keyframes status-pulse {
  0%   { box-shadow: 0 0 0 0 color-mix(in srgb, var(--sre-success) 55%, transparent); }
  70%  { box-shadow: 0 0 0 6px transparent; }
  100% { box-shadow: 0 0 0 0 transparent; }
}

/* ===== Light mode adjustments ===== */
.login-layout.light .login-card {
  background: rgba(255, 255, 255, 0.75);
  box-shadow:
    0 24px 80px -12px rgba(0, 0, 0, 0.12),
    0 0 0 1px rgba(0, 0, 0, 0.06);
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
</style>
