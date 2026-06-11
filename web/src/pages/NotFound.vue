<script setup lang="ts">
import { useRouter } from 'vue-router'
import { NButton, NIcon, NResult } from 'naive-ui'
import { AlertCircleOutline, HomeOutline } from '@vicons/ionicons5'
import PageHeader from '@/components/common/PageHeader.vue'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const { t } = useI18n()

function goHome() {
  router.push('/')
}
</script>

<template>
  <div class="not-found-page">
    <NResult
      status="404"
      :title="t('notFound.title')"
      :description="t('notFound.description')"
    >
      <template #icon>
        <NIcon :component="AlertCircleOutline" :size="64" />
      </template>
      <template #footer>
        <NButton type="primary" @click="goHome">
          <template #icon><NIcon :component="HomeOutline" /></template>
          {{ t('notFound.goHome') }}
        </NButton>
      </template>
    </NResult>
  </div>
</template>

<style scoped>
.not-found-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
}
</style>
