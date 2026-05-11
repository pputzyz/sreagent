<script setup lang="ts">
import { ref, onMounted } from 'vue'
import BizGroupManagement from '@/pages/settings/BizGroupManagement.vue'
import { userApi } from '@/api'
import type { User } from '@/types'

const allUsers = ref<User[]>([])

onMounted(async () => {
  try {
    const { data } = await userApi.list({ page: 1, page_size: 500 })
    allUsers.value = data.data.list || []
  } catch {
    // silently ignore — BizGroupManagement handles empty list
  }
})
</script>

<template>
  <BizGroupManagement :all-users="allUsers" />
</template>
