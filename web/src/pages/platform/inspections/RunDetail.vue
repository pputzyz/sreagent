<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NButton, NIcon, NCard, NTag, NSpace, NDataTable, NSpin,
  useMessage,
} from 'naive-ui'
import { ArrowBackOutline } from '@vicons/ionicons5'
import { inspectionApi } from '@/api/inspection'
import type { InspectionRun, InspectionFinding } from '@/api/inspection'
import { getErrorMessage } from '@/utils/format'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const message = useMessage()

const loading = ref(true)
const run = ref<InspectionRun | null>(null)

const findings = computed<InspectionFinding[]>(() => {
  if (!run.value?.findings_json) return []
  try {
    return JSON.parse(run.value.findings_json)
  } catch {
    return []
  }
})

const statusType = computed(() => {
  if (!run.value) return 'info'
  return run.value.status === 'success' ? 'success' : run.value.status === 'failed' ? 'error' : 'info'
})

const findingColumns = [
  {
    title: '等级',
    key: 'severity',
    width: 80,
    render: (row: InspectionFinding) => {
      const type = row.severity === 'critical' ? 'error' : row.severity === 'warning' ? 'warning' : 'info'
      return h(NTag, { type, size: 'small' }, { default: () => row.severity.toUpperCase() })
    },
  },
  { title: '分类', key: 'category', width: 120 },
  { title: '对象', key: 'object', width: 150 },
  { title: '详情', key: 'detail', ellipsis: { tooltip: true } },
]

async function fetchRun() {
  const id = Number(route.params.id)
  if (!id) {
    message.error('无效的运行 ID')
    return
  }
  loading.value = true
  try {
    const { data } = await inspectionApi.getRun(id)
    run.value = data.data || null
  } catch (e) {
    message.error(getErrorMessage(e))
  } finally {
    loading.value = false
  }
}

onMounted(fetchRun)
</script>

<template>
  <div style="padding: 16px; display: flex; flex-direction: column; gap: 16px;">
    <!-- Header -->
    <div style="display: flex; align-items: center; gap: 12px">
      <NButton text @click="router.back()">
        <template #icon><NIcon><ArrowBackOutline /></NIcon></template>
      </NButton>
      <h2 style="margin: 0; font-size: 18px">巡检报告 #{{ route.params.id }}</h2>
      <NTag v-if="run" :type="statusType" size="small">{{ run.status }}</NTag>
    </div>

    <NSpin v-if="loading" />

    <template v-else-if="run">
      <!-- Summary -->
      <NCard title="摘要" size="small">
        <div style="font-size: 14px">{{ run.report_summary || '无摘要' }}</div>
        <div style="margin-top: 8px; font-size: 12px; color: #999">
          任务 ID: {{ run.task_id }} | 开始: {{ new Date(run.started_at).toLocaleString() }}
          <template v-if="run.finished_at">
            | 结束: {{ new Date(run.finished_at).toLocaleString() }}
          </template>
        </div>
      </NCard>

      <!-- Findings -->
      <NCard v-if="findings.length > 0" title="发现项" size="small">
        <NDataTable
          :columns="findingColumns"
          :data="findings"
          :bordered="false"
          size="small"
        />
      </NCard>

      <!-- Error -->
      <NCard v-if="run.error_msg" title="错误信息" size="small">
        <div style="color: #d03050; font-size: 13px; white-space: pre-wrap">{{ run.error_msg }}</div>
      </NCard>

      <!-- Full report markdown -->
      <NCard v-if="run.report_markdown" title="完整报告" size="small">
        <div style="white-space: pre-wrap; font-size: 13px; line-height: 1.6">{{ run.report_markdown }}</div>
      </NCard>
    </template>
  </div>
</template>
