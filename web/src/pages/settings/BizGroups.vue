<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import BizGroupManagement from '@/pages/settings/BizGroupManagement.vue'
import { userApi } from '@/api'
import type { User } from '@/types'

const message = useMessage()
const { t } = useI18n()

const allUsers = ref<User[]>([])

onMounted(async () => {
  try {
    const { data } = await userApi.list({ page: 1, page_size: 500 })
    allUsers.value = data.data.list || []
  } catch {
    message.error(t('common.loadFailed'))
  }
})
</script>

<template>
  <BizGroupManagement :all-users="allUsers" />
</template>
