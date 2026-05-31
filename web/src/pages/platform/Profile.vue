<script setup lang="ts">
import { reactive, ref, computed, onMounted, watch } from 'vue'
import { useMessage, NForm, NFormItem, NInput, NButton, NSpin, NTabs, NTabPane, NSelect, NSwitch } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePreferencesStore, ACCENT_COLORS } from '@/stores/preferences'
import { authApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import UserAvatar from '@/components/common/UserAvatar.vue'
import type { UserPreferences } from '@/types'

const message = useMessage()
const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const prefsStore = usePreferencesStore()

const loading = ref(false)
const saving = ref(false)

const form = reactive({
  username: '',
  display_name: '',
  email: '',
  phone: '',
})

// Avatar preset
const presetAvatars = [
  'engineer', 'firefighter', 'detective', 'pilot', 'scientist', 'wizard',
  'ninja', 'chef', 'astronaut', 'artist', 'doctor', 'pirate',
]

const avatarPreset = computed({
  get: () => {
    const uid = authStore.user?.id
    if (!uid) return ''
    return localStorage.getItem(`sre-avatar-preset-${uid}`) || ''
  },
  set: (val: string) => {
    const uid = authStore.user?.id
    if (!uid) return
    if (val) {
      localStorage.setItem(`sre-avatar-preset-${uid}`, val)
    } else {
      localStorage.removeItem(`sre-avatar-preset-${uid}`)
    }
  },
})

const displayName = computed(() =>
  authStore.user?.display_name || authStore.user?.username || 'U',
)

function selectPreset(preset: string) {
  avatarPreset.value = avatarPreset.value === preset ? '' : preset
}

// Password form
const passwordForm = reactive({ old_password: '', new_password: '', confirm_password: '' })
const changingPassword = ref(false)

// Preferences
const prefsLoading = ref(false)
const prefsSaving = ref(false)
const prefs = reactive<UserPreferences>({
  user_id: 0,
  theme: 'auto',
  language: 'zh-CN',
  timezone: 'Asia/Shanghai',
  default_time_range: '24h',
  notification_severities: '["critical","warning"]',
  ai_chat_mode: 'sidebar',
})

const themeOptions = [
  { label: 'Auto', value: 'auto' },
  { label: 'Light', value: 'light' },
  { label: 'Dark', value: 'dark' },
]
const langOptions = [
  { label: '中文', value: 'zh-CN' },
  { label: 'English', value: 'en' },
]
const timeRangeOptions = [
  { label: '1h', value: '1h' },
  { label: '6h', value: '6h' },
  { label: '24h', value: '24h' },
  { label: '7d', value: '7d' },
  { label: '30d', value: '30d' },
]
const chatModeOptions = [
  { label: 'Sidebar', value: 'sidebar' },
  { label: 'Modal', value: 'modal' },
  { label: 'Inline', value: 'inline' },
]

const accentColorOptions = Object.entries(ACCENT_COLORS).map(([key, val]) => ({
  label: key.charAt(0).toUpperCase() + key.slice(1),
  value: key,
  color: val.primary,
}))

onMounted(async () => {
  loading.value = true
  try {
    if (!authStore.user) {
      await authStore.fetchProfile()
    }
    if (authStore.user) {
      form.username = authStore.user.username
      form.display_name = authStore.user.display_name || ''
      form.email = authStore.user.email || ''
      form.phone = authStore.user.phone || ''
    }
  } finally {
    loading.value = false
  }

  // Load preferences
  prefsLoading.value = true
  try {
    const res = await authApi.getPreferences()
    const p = res.data?.data
    if (p) {
      Object.assign(prefs, p)
    }
  } catch {
    // Use defaults
  } finally {
    prefsLoading.value = false
  }
})

async function handleSave() {
  if (form.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
    message.error(t('profile.invalidEmail'))
    return
  }
  saving.value = true
  try {
    await authApi.updateMe({
      display_name: form.display_name,
      email: form.email,
      phone: form.phone,
    })
    await authStore.fetchProfile()
    message.success(t('profile.saved'))
  } catch (err: unknown) {
    message.error(getErrorMessage(err) || t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function handleChangePassword() {
  if (passwordForm.new_password !== passwordForm.confirm_password) {
    message.error(t('profile.passwordMismatch'))
    return
  }
  if (!passwordForm.new_password) {
    message.error(t('common.required'))
    return
  }
  changingPassword.value = true
  try {
    await authApi.changeMyPassword({
      old_password: passwordForm.old_password,
      new_password: passwordForm.new_password,
    })
    message.success(t('profile.passwordChanged'))
    passwordForm.old_password = ''
    passwordForm.new_password = ''
    passwordForm.confirm_password = ''
  } catch (err: unknown) {
    message.error(getErrorMessage(err) || t('common.failed'))
  } finally {
    changingPassword.value = false
  }
}

function handleLogout() {
  authStore.logout()
  router.push('/login')
}

async function handleSavePrefs() {
  prefsSaving.value = true
  try {
    await authApi.updatePreferences({ ...prefs })
    prefsStore.applyAccentColor()
    prefsStore.applyTheme()
    prefsStore.applyLanguage({ value: prefs.language })
    message.success(t('common.saved'))
  } catch (err: unknown) {
    message.error(getErrorMessage(err) || t('common.saveFailed'))
  } finally {
    prefsSaving.value = false
  }
}
</script>

<template>
  <div class="page-container">
    <n-tabs type="line" animated>
      <!-- Tab 1: Basic Info -->
      <n-tab-pane name="basic" :tab="t('header.profile')">
        <div class="content-card">
          <!-- Avatar Selection -->
          <div class="avatar-section">
            <div class="avatar-current">
              <UserAvatar
                :src="authStore.user?.avatar || undefined"
                :preset-id="avatarPreset"
                :name="displayName"
                :size="64"
                :show-ring="true"
              />
            </div>
            <div class="avatar-presets">
              <div class="avatar-presets-label">{{ t('profile.selectAvatar') }}</div>
              <div class="avatar-presets-grid">
                <button
                  v-for="preset in presetAvatars"
                  :key="preset"
                  class="avatar-preset-btn"
                  :class="{ 'avatar-preset-btn--active': avatarPreset === preset }"
                  @click="selectPreset(preset)"
                >
                  <UserAvatar :preset-id="preset" :size="36" />
                </button>
              </div>
            </div>
          </div>

          <NSpin :show="loading">
            <NForm label-placement="left" label-width="100" :model="form" style="max-width: 480px;">
              <NFormItem :label="t('auth.username')">
                <NInput v-model:value="form.username" disabled />
              </NFormItem>
              <NFormItem :label="t('settings.displayName')">
                <NInput v-model:value="form.display_name" :placeholder="t('settings.displayName')" />
              </NFormItem>
              <NFormItem :label="t('profile.email')">
                <NInput v-model:value="form.email" placeholder="name@example.com" />
              </NFormItem>
              <NFormItem :label="t('settings.phone')">
                <NInput v-model:value="form.phone" placeholder="+86 ..." />
              </NFormItem>
              <NFormItem>
                <NButton type="primary" :loading="saving" @click="handleSave">
                  {{ t('common.save') }}
                </NButton>
              </NFormItem>
            </NForm>
          </NSpin>
        </div>
      </n-tab-pane>

      <!-- Tab 2: Change Password -->
      <n-tab-pane name="password" :tab="t('header.changePassword')">
        <div class="content-card" style="max-width: 520px;">
          <n-form :model="passwordForm" label-placement="left" label-width="auto">
            <n-form-item :label="t('profile.oldPassword')">
              <n-input v-model:value="passwordForm.old_password" type="password" show-password-on="click" />
            </n-form-item>
            <n-form-item :label="t('profile.newPassword')">
              <n-input v-model:value="passwordForm.new_password" type="password" show-password-on="click" />
            </n-form-item>
            <n-form-item :label="t('profile.confirmPassword')">
              <n-input v-model:value="passwordForm.confirm_password" type="password" show-password-on="click" @keyup.enter="handleChangePassword" />
            </n-form-item>
            <n-form-item>
              <n-button type="primary" :loading="changingPassword" @click="handleChangePassword">
                {{ t('profile.changePassword') }}
              </n-button>
            </n-form-item>
          </n-form>
        </div>
      </n-tab-pane>

      <!-- Tab 3: Preferences -->
      <n-tab-pane name="preferences" :tab="t('profile.preferences')">
        <div class="content-card" style="max-width: 520px;">
          <n-spin :show="prefsLoading">
            <n-form :model="prefs" label-placement="left" label-width="auto">
              <n-form-item :label="t('profile.theme')">
                <n-select v-model:value="prefs.theme" :options="themeOptions" />
              </n-form-item>
              <n-form-item :label="t('profile.language')">
                <n-select v-model:value="prefs.language" :options="langOptions" />
              </n-form-item>
              <n-form-item :label="t('profile.timezone')">
                <n-input v-model:value="prefs.timezone" placeholder="Asia/Shanghai" />
              </n-form-item>
              <n-form-item :label="t('profile.defaultTimeRange')">
                <n-select v-model:value="prefs.default_time_range" :options="timeRangeOptions" />
              </n-form-item>
              <n-form-item :label="t('profile.aiChatMode')">
                <n-select v-model:value="prefs.ai_chat_mode" :options="chatModeOptions" />
              </n-form-item>
              <!-- FE8-10: Accent color picker -->
              <n-form-item :label="t('profile.accentColor')">
                <div class="accent-color-grid">
                  <button
                    v-for="opt in accentColorOptions"
                    :key="opt.value"
                    class="accent-color-btn"
                    :class="{ 'accent-color-btn--active': (prefs.accent_color || 'teal') === opt.value }"
                    :style="{ '--swatch-color': opt.color }"
                    :title="opt.label"
                    @click="prefs.accent_color = opt.value"
                  >
                    <span class="accent-swatch" />
                    <span class="accent-label">{{ opt.label }}</span>
                  </button>
                </div>
              </n-form-item>
              <n-form-item>
                <n-button type="primary" :loading="prefsSaving" @click="handleSavePrefs">
                  {{ t('common.save') }}
                </n-button>
              </n-form-item>
            </n-form>
          </n-spin>
        </div>
      </n-tab-pane>
    </n-tabs>

    <!-- Danger Zone: Logout -->
    <div class="content-card" style="margin-top: 24px; border-color: var(--sre-critical-soft);">
      <div style="display: flex; align-items: center; justify-content: space-between;">
        <div>
          <h3 style="margin: 0 0 4px; font-size: 14px; font-weight: 600; color: var(--sre-text-primary);">{{ t('header.logout') }}</h3>
          <p style="margin: 0; font-size: 12px; color: var(--sre-text-tertiary);">{{ t('auth.logout') }}</p>
        </div>
        <n-button type="error" @click="handleLogout">{{ t('header.logout') }}</n-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.avatar-section {
  display: flex;
  align-items: flex-start;
  gap: 20px;
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--sre-border);
}

.avatar-current {
  flex-shrink: 0;
}

.avatar-presets {
  flex: 1;
  min-width: 0;
}

.avatar-presets-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  margin-bottom: 10px;
}

.avatar-presets-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 8px;
}

.avatar-preset-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 6px;
  border: 2px solid transparent;
  border-radius: var(--sre-radius-md);
  background: var(--sre-bg-elevated);
  cursor: pointer;
  transition: all 150ms var(--sre-ease-out);
}

.avatar-preset-btn:hover {
  border-color: var(--sre-border-strong);
  transform: scale(1.08);
}

.avatar-preset-btn--active {
  border-color: var(--sre-primary);
  background: var(--sre-primary-soft);
}

.avatar-preset-btn:active {
  transform: scale(0.95);
}

@media (max-width: 600px) {
  .avatar-section {
    flex-direction: column;
    align-items: center;
  }
  .avatar-presets-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

/* FE8-10: Accent color picker */
.accent-color-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.accent-color-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border: 2px solid transparent;
  border-radius: var(--sre-radius-md);
  background: var(--sre-bg-elevated);
  cursor: pointer;
  transition: all 150ms var(--sre-ease-out);
}
.accent-color-btn:hover {
  border-color: var(--sre-border-strong);
}
.accent-color-btn--active {
  border-color: var(--swatch-color);
  background: var(--sre-primary-soft);
}
.accent-swatch {
  display: inline-block;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--swatch-color);
  flex-shrink: 0;
}
.accent-label {
  font-size: 12px;
  color: var(--sre-text-secondary);
  text-transform: capitalize;
}
</style>
