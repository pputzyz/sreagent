<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { useMessage, NForm, NFormItem, NInput, NButton, NSpin, NTabs, NTabPane } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api'

const message = useMessage()
const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()

const loading = ref(false)
const saving = ref(false)

const form = reactive({
  username: '',
  display_name: '',
  email: '',
  phone: '',
})

// Password form
const passwordForm = reactive({ old_password: '', new_password: '', confirm_password: '' })
const changingPassword = ref(false)

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
})

async function handleSave() {
  saving.value = true
  try {
    await authApi.updateMe({
      display_name: form.display_name,
      email: form.email,
      phone: form.phone,
    })
    await authStore.fetchProfile()
    message.success(t('profile.saved'))
  } catch (err: any) {
    message.error(err.message || t('common.saveFailed'))
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
  } catch (err: any) {
    message.error(err.message || t('common.failed'))
  } finally {
    changingPassword.value = false
  }
}

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>

<template>
  <div class="page-container">
    <n-tabs type="line" animated>
      <!-- Tab 1: Basic Info -->
      <n-tab-pane name="basic" :tab="t('header.profile')">
        <div class="content-card">
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
