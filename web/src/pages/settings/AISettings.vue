<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import {
  NButton, NIcon, NSwitch, NAlert, NCard, NDivider, NSpin,
  NSpace, NTag, useMessage,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { PulseOutline, SaveOutline, SparklesOutline } from '@vicons/ionicons5'
import { aiApi } from '@/api'
import { aiModuleApi } from '@/api/preset-rules'
import type { AIModuleConfig } from '@/types/preset-rule'
import { getErrorMessage } from '@/utils/format'

const message = useMessage()
const { t } = useI18n()

// ─── AI Config (read-only display) ───
const configLoading = ref(false)
const aiConfig = ref<{
  provider: string
  model: string
  base_url: string
  enabled: boolean
} | null>(null)

// ─── Module config ───
const moduleLoading = ref(false)
const saving = ref(false)
const testing = ref(false)
const modules = ref<AIModuleConfig | null>(null)

const moduleLabels: Record<keyof AIModuleConfig, { name: string; description: string }> = {
  platform: {
    name: '平台智能助手',
    description: '全局 AI 助手浮窗，支持自然语言问答、告警上下文对话',
  },
  chat: {
    name: 'AI 对话',
    description: '告警详情页的 AI 对话面板，支持告警分析和通用问答模式',
  },
  rule_gen: {
    name: '规则生成',
    description: '基于自然语言描述自动生成 PromQL/MetricsQL 告警规则表达式',
  },
  analysis: {
    name: '告警分析',
    description: '告警事件的 AI 根因分析报告和 SOP 建议生成',
  },
  agent: {
    name: 'AI Agent',
    description: '自主告警处理 Agent，支持自动诊断、关联分析和处理建议',
  },
}

const moduleKeys: (keyof AIModuleConfig)[] = ['platform', 'chat', 'rule_gen', 'analysis', 'agent']

// ─── Fetch AI config ───
async function fetchAIConfig() {
  configLoading.value = true
  try {
    const res = await aiApi.getConfig()
    const d = res.data.data
    aiConfig.value = {
      provider: d.provider || 'openai',
      model: d.model || '',
      base_url: d.base_url || '',
      enabled: d.enabled,
    }
  } catch {
    aiConfig.value = null
  } finally {
    configLoading.value = false
  }
}

// ─── Fetch module config ───
async function fetchModules() {
  moduleLoading.value = true
  try {
    const res = await aiModuleApi.getModules()
    modules.value = res.data.data
  } catch {
    modules.value = null
  } finally {
    moduleLoading.value = false
  }
}

// ─── Toggle module ───
function toggleModule(key: keyof AIModuleConfig, val: boolean) {
  if (!modules.value) return
  modules.value[key].enabled = val
}

// ─── Save ───
async function handleSave() {
  if (!modules.value) return
  saving.value = true
  try {
    await aiModuleApi.updateModules(modules.value)
    message.success('AI 模块配置已保存')
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

// ─── Test connection ───
async function handleTest() {
  testing.value = true
  try {
    const res = await aiApi.testConnection()
    const ok = !!res.data.data?.success
    ok
      ? message.success(res.data.data?.message || '连接测试成功')
      : message.error(res.data.data?.message || '连接测试失败')
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    testing.value = false
  }
}

// ─── Provider label ───
function providerLabel(p: string) {
  const map: Record<string, string> = {
    openai: 'OpenAI',
    azure: 'Azure OpenAI',
    ollama: 'Ollama (Local)',
    custom: 'Custom / Compatible',
  }
  return map[p] || p
}

onMounted(() => {
  fetchAIConfig()
  fetchModules()
})
</script>

<template>
  <NSpin :show="configLoading && moduleLoading">
    <div class="sre-config-page ai-settings-page">
      <header class="sre-config-header">
        <div>
          <h2 class="sre-config-header-title">
            <n-icon :component="SparklesOutline" :size="20" style="margin-right: 8px; vertical-align: -3px;" />
            AI 配置
          </h2>
          <p class="sre-config-header-sub">管理 AI 模块开关和连接配置</p>
        </div>
        <div class="sre-config-header-actions">
          <n-button size="small" :loading="testing" @click="handleTest">
            <template #icon><n-icon :component="PulseOutline" /></template>
            测试连接
          </n-button>
          <n-button type="primary" size="small" :loading="saving" @click="handleSave">
            <template #icon><n-icon :component="SaveOutline" /></template>
            保存
          </n-button>
        </div>
      </header>

      <!-- Warning: AI not configured -->
      <n-alert
        v-if="!configLoading && (!aiConfig || !aiConfig.enabled)"
        type="warning"
        :bordered="false"
        style="margin-bottom: 20px"
      >
        AI 功能尚未配置或已禁用。请先在
        <router-link to="/platform/settings/ai" class="alert-link">AI 基础配置</router-link>
        中设置 API Key 和模型信息。
      </n-alert>

      <div class="config-sections sre-stagger">
        <!-- Section 1: AI Provider Info (read-only) -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">AI 总开关</h3>
          <p class="sre-config-section-desc">当前 AI 服务连接信息，修改请前往 AI 基础配置页</p>
          <div class="ai-info-grid" v-if="aiConfig">
            <div class="ai-info-item">
              <span class="ai-info-label">状态</span>
              <n-tag :type="aiConfig.enabled ? 'success' : 'default'" size="small" :bordered="false">
                {{ aiConfig.enabled ? '已启用' : '已禁用' }}
              </n-tag>
            </div>
            <div class="ai-info-item">
              <span class="ai-info-label">提供商</span>
              <span class="ai-info-value">{{ providerLabel(aiConfig.provider) }}</span>
            </div>
            <div class="ai-info-item">
              <span class="ai-info-label">模型</span>
              <span class="ai-info-value mono">{{ aiConfig.model || '未配置' }}</span>
            </div>
            <div class="ai-info-item full-row">
              <span class="ai-info-label">Base URL</span>
              <span class="ai-info-value mono">{{ aiConfig.base_url || '默认' }}</span>
            </div>
          </div>
          <div v-else-if="!configLoading" class="ai-info-empty">
            无法加载 AI 配置信息
          </div>
        </section>

        <n-divider />

        <!-- Section 2: Module Toggles -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">模块开关</h3>
          <p class="sre-config-section-desc">独立控制各 AI 功能模块的启用状态</p>

          <div v-if="modules" class="module-list">
            <div
              v-for="key in moduleKeys"
              :key="key"
              class="module-item"
              :class="{ disabled: !modules[key].enabled }"
            >
              <div class="module-info">
                <div class="module-name">
                  {{ moduleLabels[key].name }}
                  <n-tag v-if="modules[key].enabled" type="success" size="tiny" :bordered="false">已启用</n-tag>
                  <n-tag v-else size="tiny" :bordered="false">已禁用</n-tag>
                </div>
                <div class="module-desc">{{ moduleLabels[key].description }}</div>
              </div>
              <n-switch
                :value="modules[key].enabled"
                @update:value="(val: boolean) => toggleModule(key, val)"
              />
            </div>
          </div>
          <div v-else-if="!moduleLoading" class="ai-info-empty">
            无法加载模块配置
          </div>
        </section>
      </div>
    </div>
  </NSpin>
</template>

<style scoped>
.ai-settings-page {
  max-width: 800px;
}

.alert-link {
  color: var(--sre-primary);
  text-decoration: underline;
  font-weight: 500;
}
.alert-link:hover {
  opacity: 0.8;
}

/* AI Info Grid */
.ai-info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px 24px;
  padding: 16px;
  background: var(--sre-bg-sunken, rgba(0,0,0,0.02));
  border-radius: 8px;
  border: 1px solid var(--sre-border);
}
.ai-info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.ai-info-item.full-row {
  grid-column: 1 / -1;
}
.ai-info-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.ai-info-value {
  font-size: 14px;
  color: var(--sre-text-primary);
  font-weight: 500;
}
.ai-info-value.mono {
  font-family: var(--sre-font-mono, monospace);
  font-size: 13px;
}
.ai-info-empty {
  font-size: 13px;
  color: var(--sre-text-tertiary);
  padding: 16px;
  text-align: center;
}

/* Module List */
.module-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.module-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  transition: opacity 200ms ease, border-color 200ms ease;
}
.module-item.disabled {
  opacity: 0.65;
}
.module-item:hover {
  border-color: var(--sre-primary-ring, var(--sre-border-strong));
}
.module-info {
  flex: 1;
  min-width: 0;
}
.module-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}
.module-desc {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  line-height: 1.5;
}
</style>
