<script setup lang="ts">
import { ref, onMounted, computed, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NButton, NIcon, NCard, NTag, NSpace, NDataTable, NSpin,
  useMessage,
} from 'naive-ui'
import { ArrowBackOutline, DownloadOutline } from '@vicons/ionicons5'
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

// FE7-6: Severity badge counts from findings
const severityCounts = computed(() => {
  const counts = { critical: 0, warning: 0, info: 0 }
  for (const f of findings.value) {
    if (f.severity === 'critical') counts.critical++
    else if (f.severity === 'warning') counts.warning++
    else counts.info++
  }
  return counts
})

const statusType = computed(() => {
  if (!run.value) return 'info'
  return run.value.status === 'success' ? 'success' : run.value.status === 'failed' ? 'error' : 'info'
})

const findingColumns = [
  {
    title: t('inspection.severity'),
    key: 'severity',
    width: 80,
    render: (row: InspectionFinding) => {
      const type = row.severity === 'critical' ? 'error' : row.severity === 'warning' ? 'warning' : 'info'
      return h(NTag, { type, size: 'small' }, { default: () => row.severity.toUpperCase() })
    },
  },
  { title: t('inspection.category'), key: 'category', width: 120 },
  { title: t('inspection.object'), key: 'object', width: 150 },
  { title: t('inspection.detail'), key: 'detail', ellipsis: { tooltip: true } },
]

async function fetchRun() {
  const id = Number(route.params.id)
  if (!id) {
    message.error(t('inspection.invalidRunId'))
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

// Export report as markdown file (FE7-5)
function exportReport() {
  if (!run.value?.report_markdown) return
  const blob = new Blob([run.value.report_markdown], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `inspection-report-${run.value.id}.md`
  a.click()
  URL.revokeObjectURL(url)
  message.success(t('common.success'))
}
</script>

<template>
  <div style="padding: 16px; display: flex; flex-direction: column; gap: 16px;">
    <!-- Header -->
    <div style="display: flex; align-items: center; gap: 12px">
      <NButton text @click="router.back()">
        <template #icon><NIcon><ArrowBackOutline /></NIcon></template>
      </NButton>
      <h2 style="margin: 0; font-size: 18px">{{ t('inspection.reportTitle', { id: route.params.id }) }}</h2>
      <NTag v-if="run" :type="statusType" size="small">{{ run.status }}</NTag>
      <NSpace v-if="findings.length > 0" size="small" style="margin-left: 8px">
        <NTag v-if="severityCounts.critical > 0" type="error" size="small" :bordered="false">
          {{ severityCounts.critical }} {{ t('inspection.critical') || 'critical' }}
        </NTag>
        <NTag v-if="severityCounts.warning > 0" type="warning" size="small" :bordered="false">
          {{ severityCounts.warning }} {{ t('inspection.warning') || 'warning' }}
        </NTag>
        <NTag v-if="severityCounts.info > 0" type="info" size="small" :bordered="false">
          {{ severityCounts.info }} {{ t('inspection.info') || 'info' }}
        </NTag>
      </NSpace>
      <div style="margin-left: auto;">
        <NButton
          v-if="run?.report_markdown"
          size="small"
          quaternary
          @click="exportReport"
        >
          <template #icon><NIcon><DownloadOutline /></NIcon></template>
          {{ t('common.export') || 'Export' }}
        </NButton>
      </div>
    </div>

    <NSpin v-if="loading" />

    <template v-else-if="run">
      <!-- Summary -->
      <NCard :title="t('inspection.summary')" size="small">
        <div style="font-size: 14px">{{ run.report_summary || t('inspection.noSummary') }}</div>
        <div style="margin-top: 8px; font-size: 12px; color: #999">
          {{ t('inspection.taskLabel') }}: {{ run.task_id }} | {{ t('inspection.started') }}: {{ new Date(run.started_at).toLocaleString() }}
          <template v-if="run.finished_at">
            | {{ t('inspection.finished') }}: {{ new Date(run.finished_at).toLocaleString() }}
          </template>
        </div>
      </NCard>

      <!-- Findings -->
      <NCard v-if="findings.length > 0" :title="t('inspection.findings')" size="small">
        <NDataTable
          :columns="findingColumns"
          :data="findings"
          :bordered="false"
          size="small"
        />
      </NCard>

      <!-- Error -->
      <NCard v-if="run.error_msg" :title="t('inspection.errorMsg')" size="small">
        <div style="color: #d03050; font-size: 13px; white-space: pre-wrap">{{ run.error_msg }}</div>
      </NCard>

      <!-- Full report markdown -->
      <NCard v-if="run.report_markdown" :title="t('inspection.fullReport')" size="small">
        <div style="white-space: pre-wrap; font-size: 13px; line-height: 1.6">{{ run.report_markdown }}</div>
      </NCard>
    </template>
  </div>
</template>
