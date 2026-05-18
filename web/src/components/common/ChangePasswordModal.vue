<script setup lang="ts">
import { reactive, ref } from 'vue'
import { NModal, NForm, NFormItem, NInput, NButton, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { authApi } from '@/api'
import { getErrorMessage } from '@/utils/format'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ 'update:show': [value: boolean] }>()

const { t } = useI18n()
const message = useMessage()
const loading = ref(false)

const form = reactive({
  old_password: '',
  new_password: '',
  confirm_password: '',
})

async function handleSubmit() {
  if (form.new_password !== form.confirm_password) {
    message.error(t('profile.passwordMismatch'))
    return
  }
  if (!form.new_password) {
    message.error(t('common.required'))
    return
  }
  loading.value = true
  try {
    await authApi.changeMyPassword({
      old_password: form.old_password,
      new_password: form.new_password,
    })
    message.success(t('profile.passwordChanged'))
    emit('update:show', false)
    form.old_password = ''
    form.new_password = ''
    form.confirm_password = ''
  } catch (err: unknown) {
    message.error(getErrorMessage(err) || t('common.failed'))
  } finally {
    loading.value = false
  }
}

function handleClose() {
  emit('update:show', false)
  form.old_password = ''
  form.new_password = ''
  form.confirm_password = ''
}
</script>

<template>
  <n-modal
    :show="show"
    @update:show="emit('update:show', $event)"
    :title="t('header.changePassword')"
    preset="card"
    style="max-width: 420px;"
    :bordered="false"
    :segmented="{ content: true, footer: true }"
    @mask-click="handleClose"
  >
    <n-form :model="form" label-placement="left" label-width="auto">
      <n-form-item :label="t('profile.oldPassword')">
        <n-input
          v-model:value="form.old_password"
          type="password"
          show-password-on="click"
          :placeholder="t('profile.oldPassword')"
        />
      </n-form-item>
      <n-form-item :label="t('profile.newPassword')">
        <n-input
          v-model:value="form.new_password"
          type="password"
          show-password-on="click"
          :placeholder="t('profile.newPassword')"
        />
      </n-form-item>
      <n-form-item :label="t('profile.confirmPassword')">
        <n-input
          v-model:value="form.confirm_password"
          type="password"
          show-password-on="click"
          :placeholder="t('profile.confirmPassword')"
          @keyup.enter="handleSubmit"
        />
      </n-form-item>
    </n-form>
    <template #footer>
      <div style="display: flex; justify-content: flex-end; gap: 8px;">
        <n-button @click="handleClose">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" :loading="loading" @click="handleSubmit">
          {{ t('profile.changePassword') }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>
