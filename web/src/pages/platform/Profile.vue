<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { useMessage, NForm, NFormItem, NInput, NButton, NSpin } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api'

const message = useMessage()
const { t } = useI18n()
const authStore = useAuthStore()

const loading = ref(false)
const saving = ref(false)

const form = reactive({
  username: '',
  display_name: '',
  email: '',
  phone: '',
})

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
</script>

<template>
  <div class="page-container">
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
  </div>
</template>
