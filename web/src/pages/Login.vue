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
    router.push((route.query.redirect as string) || '/dashboard')
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
    <!-- Top right: language + theme -->
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

    <!-- Brand side (60%) -->
    <aside class="login-brand">
      <div class="brand-eyebrow">
        <span class="brand-dot" />
        <span>{{ t('auth.brand.control') }}</span>
      </div>
      <h1 class="brand-title">SREAGENT</h1>
      <p class="brand-tagline">{{ t('auth.brand.tagline') }}</p>

      <ul class="brand-features">
        <li class="feature-item">{{ t('auth.brand.feature1') }}</li>
        <li class="feature-item">{{ t('auth.brand.feature2') }}</li>
        <li class="feature-item">{{ t('auth.brand.feature3') }}</li>
        <li class="feature-item">{{ t('auth.brand.feature4') }}</li>
      </ul>

      <div class="brand-foot">
        <span class="brand-version">v{{ appVersion }} &middot; build {{ new Date().getFullYear() }}</span>
        <span class="brand-status">
          <span class="status-pulse" /> {{ t('auth.brand.systemStatus') }}
        </span>
      </div>
    </aside>

    <!-- Form side (40%) -->
    <section class="login-form-side">
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

        <p class="form-default-hint" v-html="t('auth.defaultHint')" />
      </form>
    </section>
  </div>
</template>

<style scoped>
.login-layout {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 60% 40%;
  background: var(--sre-bg-base);
  font-family: var(--sre-font-display);
  color: var(--sre-text-primary);
  position: relative;
  overflow: hidden;
}

.login-controls {
  position: absolute;
  top: 20px;
  right: 24px;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* ===== Brand side (left 60%) ===== */
.login-brand {
  position: relative;
  padding: 80px 64px 56px;
  display: flex;
  flex-direction: column;
  background: var(--sre-bg-page);
  overflow: hidden;
}

.brand-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-family: var(--sre-font-mono);
  font-size: 11px;
  letter-spacing: 1.4px;
  color: var(--sre-text-tertiary);
  margin-bottom: 36px;
  text-transform: uppercase;
}
.brand-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--sre-primary);
  box-shadow: 0 0 12px var(--sre-primary);
}

.brand-title {
  font-family: var(--sre-font-display);
  font-size: clamp(40px, 5vw, 56px);
  font-weight: 800;
  letter-spacing: -2px;
  line-height: 1;
  margin: 0 0 18px;
  background: linear-gradient(135deg, var(--sre-text-primary) 0%, color-mix(in srgb, var(--sre-text-primary) 55%, transparent) 100%);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}

.login-layout.light .brand-title {
  background: linear-gradient(135deg, var(--sre-text-primary) 0%, color-mix(in srgb, var(--sre-text-primary) 55%, transparent) 100%);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}

.brand-tagline {
  font-size: clamp(18px, 2vw, 22px);
  color: var(--sre-text-secondary);
  margin: 0 0 56px;
  max-width: 520px;
  font-weight: 400;
  line-height: 1.5;
}

.brand-features {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin: 0 0 auto;
  padding: 0;
  list-style: none;
  max-width: 520px;
}
.feature-item {
  position: relative;
  font-size: 13px;
  color: var(--sre-text-secondary);
  padding-left: 18px;
  line-height: 1.5;
}
.feature-item::before {
  content: '\25B8';
  position: absolute;
  left: 0;
  top: 0;
  color: var(--sre-primary);
  font-size: 12px;
}

.brand-foot {
  margin-top: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding-top: 20px;
  border-top: var(--sre-hairline);
  font-family: var(--sre-font-mono);
  font-size: 11px;
  color: var(--sre-text-tertiary);
  letter-spacing: 0.5px;
}
.brand-status {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  text-transform: uppercase;
}
.status-pulse {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--sre-success);
  box-shadow: 0 0 0 0 color-mix(in srgb, var(--sre-success) 60%, transparent);
  animation: status-pulse 2.4s ease-out infinite;
}
@keyframes status-pulse {
  0%   { box-shadow: 0 0 0 0 color-mix(in srgb, var(--sre-success) 55%, transparent); }
  70%  { box-shadow: 0 0 0 8px transparent; }
  100% { box-shadow: 0 0 0 0 transparent; }
}

/* ===== Form side (right 40%) ===== */
.login-form-side {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
  background: var(--sre-bg-card);
  border-left: var(--sre-hairline);
}

.login-form {
  width: 100%;
  max-width: 360px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.login-form > * {
  opacity: 0;
  animation: login-rise 480ms var(--sre-ease-out) forwards;
}
.login-form > *:nth-child(1) { animation-delay: 60ms; }
.login-form > *:nth-child(2) { animation-delay: 120ms; }
.login-form > *:nth-child(3) { animation-delay: 180ms; }
.login-form > *:nth-child(4) { animation-delay: 240ms; }
.login-form > *:nth-child(5) { animation-delay: 300ms; }
.login-form > *:nth-child(6) { animation-delay: 360ms; }
.login-form > *:nth-child(7) { animation-delay: 420ms; }
.login-form > *:nth-child(8) { animation-delay: 480ms; }

@keyframes login-rise {
  from { opacity: 0; transform: translateY(12px); }
  to   { opacity: 1; transform: translateY(0); }
}

.form-header {
  margin-bottom: 6px;
}
.form-title {
  font-size: 24px;
  font-weight: 600;
  letter-spacing: -0.4px;
  margin: 0 0 6px;
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
  box-shadow: 0 8px 24px -8px var(--sre-primary-ring);
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
  color: #fff;
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
  color: var(--sre-warning);
  text-align: center;
  margin: 8px 0 0;
  padding: 10px 12px;
  background: var(--sre-warning-soft);
  border-radius: var(--sre-radius-sm);
  border: 1px solid color-mix(in srgb, var(--sre-warning) 22%, transparent);
  line-height: 1.5;
}
.form-default-hint code {
  font-family: var(--sre-font-mono);
  background: var(--sre-warning-soft);
  padding: 1px 6px;
  border-radius: 3px;
  color: var(--sre-warning);
  font-size: 11px;
}

/* ===== Responsive ===== */
@media (max-width: 920px) {
  .login-layout { grid-template-columns: 1fr; }
  .login-brand { display: none; }
  .login-form-side { border-left: none; padding: 24px; }
}
</style>
