<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton, NIcon, NInput, NCard, NSpin, NTag, NEmpty,
  NTimeline, NTimelineItem, NSpace, NAlert, useMessage,
} from 'naive-ui'
import {
  SparklesOutline, PlayOutline, CheckmarkCircleOutline,
  CloseCircleOutline, TimeOutline, SyncOutline,
} from '@vicons/ionicons5'
import { aiAgentApi } from '@/api'
import type { AgentTask, AgentStep } from '@/api'
import { getErrorMessage } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()

// 状态
const query = ref('')
const loading = ref(false)
const task = ref<AgentTask | null>(null)
let eventSource: EventSource | null = null

// 是否正在接收 SSE 流（任务执行中）
const isPolling = computed(() => {
  return !!eventSource && !!task.value && (task.value.status === 'planning' || task.value.status === 'executing')
})

// 防止主动关闭后 onerror 误触轮询回退
let stopped = false

// 执行 Agent
async function handleRun() {
  if (!query.value.trim()) return
  loading.value = true
  task.value = null
  stopSSE()

  try {
    const res = await aiAgentApi.run(query.value.trim())
    task.value = res.data.data ?? null

    // 如果任务已完成，不需要 SSE
    if (task.value && (task.value.status === 'completed' || task.value.status === 'failed')) {
      return
    }

    // 启动 SSE 流
    if (task.value) startSSE(task.value.id)
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

// SSE 实时推送（替代 2s 轮询）
function startSSE(taskId: string) {
  stopSSE()
  stopped = false
  const url = `/api/v1/ai/agent/stream/${taskId}`
  const es = new EventSource(url)

  es.addEventListener('task', (e: MessageEvent) => {
    try {
      const updated = JSON.parse(e.data) as AgentTask
      task.value = updated
      if (updated.status === 'completed' || updated.status === 'failed') {
        stopSSE()
      }
    } catch {
      // 解析失败不中断流
    }
  })

  es.onerror = () => {
    stopSSE()
    // 主动关闭时不回退到轮询
    if (stopped) return
    // 连接断开时回退到轮询
    if (task.value && (task.value.status === 'planning' || task.value.status === 'executing')) {
      startPollingFallback(task.value.id)
    }
  }

  eventSource = es
}

// 回退轮询（SSE 断开时使用）
let pollingTimer: ReturnType<typeof setInterval> | null = null
let pollingLock = false
function startPollingFallback(taskId: string) {
  stopPollingFallback()
  pollingLock = false
  pollingTimer = setInterval(async () => {
    if (pollingLock || !task.value) {
      if (!task.value) stopPollingFallback()
      return
    }
    pollingLock = true
    try {
      const res = await aiAgentApi.getTask(taskId)
      const updated = res.data.data
      if (updated) {
        task.value = updated
        if (updated.status === 'completed' || updated.status === 'failed') {
          stopPollingFallback()
        }
      }
    } catch {
      // 轮询失败不停止
    } finally {
      pollingLock = false
    }
  }, 2000)
}

function stopPollingFallback() {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
}

function stopSSE() {
  stopped = true
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
  stopPollingFallback()
}

// 步骤状态颜色
function stepStatusType(status: string): 'success' | 'error' | 'warning' | 'info' {
  switch (status) {
    case 'completed': return 'success'
    case 'failed': return 'error'
    case 'running': return 'warning'
    default: return 'info'
  }
}

// 步骤状态图标
function stepStatusIcon(status: string) {
  switch (status) {
    case 'completed': return CheckmarkCircleOutline
    case 'failed': return CloseCircleOutline
    case 'running': return SyncOutline
    default: return TimeOutline
  }
}

// 步骤状态文案
function stepStatusText(status: string): string {
  const map: Record<string, string> = {
    pending: t('agent.statusPending'),
    running: t('agent.statusRunning'),
    completed: t('agent.statusCompleted'),
    failed: t('agent.statusFailed'),
  }
  return map[status] || status
}

// 格式化耗时
function formatDuration(ms: number): string {
  if (ms <= 0) return '-'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

// 任务状态文案
function taskStatusText(status: string): string {
  const map: Record<string, string> = {
    planning: t('agent.taskPlanning'),
    executing: t('agent.taskExecuting'),
    completed: t('agent.taskCompleted'),
    failed: t('agent.taskFailed'),
  }
  return map[status] || status
}

function taskStatusType(status: string): 'success' | 'error' | 'warning' | 'info' {
  switch (status) {
    case 'completed': return 'success'
    case 'failed': return 'error'
    case 'planning': return 'info'
    case 'executing': return 'warning'
    default: return 'info'
  }
}

// 回车执行
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleRun()
  }
}

onUnmounted(() => {
  stopSSE()
})
</script>

<template>
  <div class="sre-config-page agent-page">
    <header class="sre-config-header">
      <div>
        <h2 class="sre-config-header-title">
          <n-icon :component="SparklesOutline" :size="20" style="margin-right: 8px; vertical-align: -3px;" />
          {{ t('agent.title') }}
        </h2>
        <p class="sre-config-header-sub">{{ t('agent.subtitle') }}</p>
      </div>
    </header>

    <!-- 查询输入区 -->
    <n-card class="agent-input-card" :bordered="false">
      <div class="agent-input-row">
        <n-input
          v-model:value="query"
          :placeholder="t('agent.inputPlaceholder')"
          :disabled="loading || isPolling"
          size="large"
          @keydown="handleKeydown"
        />
        <n-button
          v-if="!isPolling"
          type="primary"
          size="large"
          :loading="loading"
          :disabled="!query.trim()"
          @click="handleRun"
        >
          <template #icon><n-icon :component="PlayOutline" /></template>
          {{ t('agent.run') }}
        </n-button>
        <n-button
          v-else
          type="error"
          size="large"
          @click="stopSSE"
        >
          {{ t('common.cancel') }}
        </n-button>
      </div>
    </n-card>

    <!-- 任务状态 -->
    <div v-if="task" class="agent-result-area">
      <!-- 任务概览 -->
      <n-alert :type="taskStatusType(task.status)" :bordered="false" class="agent-status-alert">
        <template #header>
          {{ t('agent.taskStatus') }}:
          <n-tag :type="taskStatusType(task.status)" size="small" :bordered="false" style="margin-left: 8px;">
            {{ taskStatusText(task.status) }}
          </n-tag>
        </template>
        <div class="agent-task-meta">
          <span>{{ t('agent.taskId') }}: <code>{{ task.id.slice(0, 8) }}</code></span>
          <span v-if="task.steps.length > 0">
            {{ t('agent.totalSteps') }}: {{ task.steps.length }}
          </span>
          <span v-if="task.completed_at">
            {{ t('agent.duration') }}: {{ formatDuration(
              new Date(task.completed_at).getTime() - new Date(task.created_at).getTime()
            ) }}
          </span>
        </div>
      </n-alert>

      <!-- 执行步骤列表 -->
      <n-card :title="t('agent.stepsTitle')" :bordered="false" class="agent-steps-card">
        <n-spin :show="isPolling">
          <n-timeline v-if="task.steps.length > 0">
            <n-timeline-item
              v-for="step in task.steps"
              :key="step.index"
              :type="stepStatusType(step.status)"
              :icon="stepStatusIcon(step.status)"
            >
              <div class="agent-step-item">
                <div class="agent-step-header">
                  <span class="agent-step-index">#{{ step.index }}</span>
                  <span class="agent-step-desc">{{ step.description }}</span>
                  <n-tag :type="stepStatusType(step.status)" size="tiny" :bordered="false">
                    {{ stepStatusText(step.status) }}
                  </n-tag>
                </div>
                <div class="agent-step-meta">
                  <span class="agent-step-tool">
                    {{ t('agent.tool') }}: <code>{{ step.tool }}</code>
                  </span>
                  <span v-if="step.duration_ms > 0" class="agent-step-duration">
                    <n-icon :component="TimeOutline" :size="12" />
                    {{ formatDuration(step.duration_ms) }}
                  </span>
                </div>
                <div v-if="step.result" class="agent-step-result">
                  <pre>{{ step.result }}</pre>
                </div>
              </div>
            </n-timeline-item>
          </n-timeline>
          <n-empty v-else :description="t('agent.noSteps')" />
        </n-spin>
      </n-card>

      <!-- 最终结果 -->
      <n-card
        v-if="task.result"
        :title="t('agent.resultTitle')"
        :bordered="false"
        class="agent-result-card"
      >
        <div class="agent-result-content">{{ task.result }}</div>
      </n-card>

      <!-- 错误信息 -->
      <n-alert
        v-if="task.error"
        type="error"
        :bordered="false"
        class="agent-error-alert"
      >
        {{ task.error }}
      </n-alert>
    </div>

    <!-- 空状态 -->
    <n-empty
      v-else-if="!loading"
      :description="t('agent.emptyDesc')"
      class="agent-empty"
    />
  </div>
</template>

<style scoped>
.agent-page {
  max-width: 880px;
}

.agent-input-card {
  margin-bottom: 20px;
}

.agent-input-row {
  display: flex;
  gap: 12px;
  align-items: center;
}

.agent-input-row .n-input {
  flex: 1;
}

.agent-result-area {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.agent-status-alert {
  margin-bottom: 4px;
}

.agent-task-meta {
  display: flex;
  gap: 20px;
  font-size: 13px;
  color: var(--sre-text-tertiary);
  margin-top: 4px;
}

.agent-task-meta code {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  background: var(--sre-bg-hover, rgba(0,0,0,0.04));
  padding: 1px 5px;
  border-radius: 3px;
}

.agent-steps-card {
  margin-bottom: 4px;
}

.agent-step-item {
  padding-bottom: 4px;
}

.agent-step-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.agent-step-index {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-tertiary);
  min-width: 24px;
}

.agent-step-desc {
  font-size: 13px;
  font-weight: 500;
  color: var(--sre-text-primary);
  flex: 1;
}

.agent-step-meta {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 12px;
  color: var(--sre-text-tertiary);
  padding-left: 32px;
}

.agent-step-tool code {
  font-family: var(--sre-font-mono, monospace);
  font-size: 11px;
  background: var(--sre-bg-hover, rgba(0,0,0,0.04));
  padding: 1px 5px;
  border-radius: 3px;
}

.agent-step-duration {
  display: flex;
  align-items: center;
  gap: 3px;
}

.agent-step-result {
  margin-top: 8px;
  padding-left: 32px;
}

.agent-step-result pre {
  font-size: 12px;
  font-family: var(--sre-font-mono, monospace);
  color: var(--sre-text-secondary);
  background: var(--sre-bg-hover, rgba(0,0,0,0.04));
  padding: 8px 12px;
  border-radius: 6px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 200px;
  overflow-y: auto;
  margin: 0;
}

.agent-result-card {
  margin-bottom: 4px;
}

.agent-result-content {
  font-size: 14px;
  line-height: 1.7;
  color: var(--sre-text-primary);
  white-space: pre-wrap;
}

.agent-error-alert {
  margin-top: 4px;
}

.agent-empty {
  padding: 60px 0;
}
</style>
