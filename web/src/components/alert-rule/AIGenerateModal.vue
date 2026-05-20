<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import {
  NModal, NInput, NSelect, NButton, NIcon, NSpace, NAlert, NTag,
  NText, NCollapse, NCollapseItem, NDataTable, useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { SparklesOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { aiRuleApi, datasourceApi, alertRuleApi } from '@/api'
import type { RuleGenerateResult, MuteRuleGenerateResult } from '@/types/ai-module'
import type { AlertSeverity } from '@/types'
import { getErrorMessage } from '@/utils/format'

export interface AIGenerateModalProps {
  visible: boolean
  ruleType: 'rule' | 'mute' | 'inhibition'
  datasourceId?: number
  datasourceOptions?: Array<{ label: string; value: number }>
}

const props = withDefaults(defineProps<AIGenerateModalProps>(), {
  datasourceId: undefined,
  datasourceOptions: () => [],
})

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'generated', result: RuleGenerateResult | MuteRuleGenerateResult): void
  (e: 'saved', payload: { draft: boolean }): void
}>()

const message = useMessage()

const { t } = useI18n()

const description = ref('')
const selectedDatasourceId = ref<number | null>(null)
const generating = ref(false)
const saving = ref(false)
const result = ref<RuleGenerateResult | MuteRuleGenerateResult | null>(null)
const error = ref('')

// Dry-run state
const dryRunResult = ref<{
  series_count: number
  sample_series: Array<Record<string, string>>
  would_fire: boolean
  eval_duration_ms: number
} | null>(null)
const dryRunLoading = ref(false)

// Label preview state
const labelHits = ref<Array<{ key: string; matched: boolean }>>([])
const labelLoading = ref(false)

const show = computed({
  get: () => props.visible,
  set: (v: boolean) => emit('update:visible', v),
})

const showDatasourceSelect = computed(() => props.ruleType === 'rule')

watch(() => props.visible, (v) => {
  if (v) {
    description.value = ''
    selectedDatasourceId.value = props.datasourceId ?? null
    result.value = null
    error.value = ''
    dryRunResult.value = null
    labelHits.value = []
  }
})

async function handleGenerate() {
  if (!description.value.trim()) return
  generating.value = true
  result.value = null
  error.value = ''
  try {
    let res
    if (props.ruleType === 'rule') {
      res = await aiRuleApi.generate({
        description: description.value,
        datasource_id: selectedDatasourceId.value ?? undefined,
        rule_type: 'alert',
      })
    } else if (props.ruleType === 'mute') {
      res = await aiRuleApi.generateMute({ description: description.value })
    } else {
      res = await aiRuleApi.generateInhibition({ description: description.value })
    }
    result.value = res.data.data
  } catch (err: unknown) {
    error.value = getErrorMessage(err) || t('alert.aiGenerateFailed')
  } finally {
    generating.value = false
  }
}

function handleRegenerate() {
  handleGenerate()
}

function handleApply() {
  if (!result.value) return
  emit('generated', result.value)
  show.value = false
}

async function handleSaveAsDraft() {
  if (!result.value) return
  saving.value = true
  try {
    await aiRuleApi.generate({
      description: description.value,
      datasource_id: selectedDatasourceId.value ?? undefined,
      rule_type: 'alert',
      save_as_draft: true,
    })
    message.success('已保存为草稿')
    emit('saved', { draft: true })
    show.value = false
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

async function handleSaveAsActive() {
  if (!result.value) return
  const ruleResult = result.value as RuleGenerateResult
  saving.value = true
  try {
    await alertRuleApi.create({
      name: ruleResult.name,
      expression: ruleResult.expression,
      for_duration: ruleResult.for_duration,
      severity: ruleResult.severity as AlertSeverity | undefined,
      labels: ruleResult.labels,
      annotations: ruleResult.annotations,
      description: ruleResult.description,
      datasource_id: selectedDatasourceId.value ?? undefined,
      status: 'active',
    })
    message.success('规则已创建并启用')
    emit('saved', { draft: false })
    show.value = false
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

async function handleDryRun() {
  if (!result.value) return
  dryRunLoading.value = true
  try {
    const resp = await aiRuleApi.dryRun({
      description: description.value,
      datasource_id: selectedDatasourceId.value ?? undefined,
      rule_type: 'alert',
    })
    const data = resp.data.data
    dryRunResult.value = {
      series_count: data.series_count,
      sample_series: data.sample_series ?? [],
      would_fire: data.would_fire,
      eval_duration_ms: data.eval_duration_ms,
    }
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    dryRunLoading.value = false
  }
}

async function handleLabelPreview() {
  if (!selectedDatasourceId.value || !result.value) return
  const ruleResult = result.value as RuleGenerateResult
  labelLoading.value = true
  try {
    const resp = await datasourceApi.labelKeys(selectedDatasourceId.value)
    const keys: string[] = resp.data.data ?? []
    const ruleKeys = Object.keys(ruleResult.labels || {})
    labelHits.value = keys.map((k: string) => ({
      key: k,
      matched: ruleKeys.includes(k),
    }))
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    labelLoading.value = false
  }
}

const sampleColumns = computed(() => {
  if (!dryRunResult.value?.sample_series?.length) return [] as DataTableColumns<Record<string, string>>
  const keys = Object.keys(dryRunResult.value.sample_series[0])
  return keys.map(k => ({ title: k, key: k, ellipsis: { tooltip: true } })) as DataTableColumns<Record<string, string>>
})

const isRuleResult = (r: RuleGenerateResult | MuteRuleGenerateResult): r is RuleGenerateResult =>
  'expression' in r

const isMuteResult = (r: RuleGenerateResult | MuteRuleGenerateResult): r is MuteRuleGenerateResult =>
  'match_labels' in r
</script>

<template>
  <NModal
    v-model:show="show"
    :title="t('alert.aiGenerate')"
    preset="card"
    :mask-closable="false"
    :bordered="false"
    style="max-width: 680px"
  >
    <div class="ai-gen-form">
      <div class="ai-gen-field">
        <label class="ai-gen-label">{{ t('alert.aiDescription') }}</label>
        <NInput
          v-model:value="description"
          type="textarea"
          :rows="3"
          :placeholder="ruleType === 'mute' ? t('alert.aiMutePlaceholder') : ruleType === 'inhibition' ? t('alert.aiInhibitionPlaceholder') : t('alert.aiDescriptionPlaceholder')"
        />
      </div>
      <div v-if="showDatasourceSelect && datasourceOptions.length > 0" class="ai-gen-field">
        <label class="ai-gen-label">{{ t('alert.dataSource') }} ({{ t('common.optional') }})</label>
        <NSelect
          v-model:value="selectedDatasourceId"
          :options="datasourceOptions"
          :placeholder="t('alert.selectDatasource')"
          clearable
        />
      </div>
      <NButton type="primary" :loading="generating" :disabled="!description.trim()" @click="handleGenerate">
        <template #icon><NIcon :component="SparklesOutline" /></template>
        {{ t('alert.aiGenerateBtn') }}
      </NButton>
    </div>

    <!-- Error -->
    <NAlert v-if="error" type="error" style="margin-top: 16px">
      {{ error }}
    </NAlert>

    <!-- Result Preview -->
    <div v-if="result" class="ai-gen-preview">
      <div class="ai-gen-preview-header">
        <span class="ai-gen-preview-title">{{ result.name }}</span>
        <template v-if="isRuleResult(result)">
          <NTag v-if="result.severity" :type="result.severity === 'critical' ? 'error' : result.severity === 'warning' ? 'warning' : 'info'" size="small">
            {{ result.severity }}
          </NTag>
        </template>
        <NTag
          size="small"
          :bordered="false"
          :type="result.confidence >= 0.8 ? 'success' : result.confidence >= 0.5 ? 'warning' : 'error'"
        >
          {{ Math.round(result.confidence * 100) }}%
        </NTag>
      </div>

      <!-- Rule type preview -->
      <template v-if="isRuleResult(result)">
        <div v-if="result.expression" class="ai-gen-expr">{{ result.expression }}</div>
        <div v-if="result.description" class="ai-gen-desc">{{ result.description }}</div>
        <div v-if="result.for_duration" class="ai-gen-meta">
          <span class="ai-gen-meta-label">{{ t('alert.aiGenDuration') }}:</span> {{ result.for_duration }}
        </div>
        <div v-if="result.labels && Object.keys(result.labels).length > 0" class="ai-gen-meta">
          <span class="ai-gen-meta-label">{{ t('alert.aiGenLabels') }}:</span>
          <NTag v-for="(v, k) in result.labels" :key="k" size="small" style="margin-right: 4px">{{ k }}={{ v }}</NTag>
        </div>
        <div v-if="result.annotations?.summary" class="ai-gen-meta">
          <span class="ai-gen-meta-label">{{ t('alert.aiGenSummary') }}:</span> {{ result.annotations.summary }}
        </div>
        <div v-if="result.source_labels?.length" class="ai-gen-meta">
          <span class="ai-gen-meta-label">{{ t('inhibition.sourceLabel') }}:</span>
          <NTag v-for="l in result.source_labels" :key="l" size="small" style="margin-right: 4px">{{ l }}</NTag>
          <span v-if="result.source_value"> = {{ result.source_value }}</span>
        </div>
        <div v-if="result.target_labels?.length" class="ai-gen-meta">
          <span class="ai-gen-meta-label">{{ t('inhibition.targetLabel') }}:</span>
          <NTag v-for="l in result.target_labels" :key="l" size="small" style="margin-right: 4px">{{ l }}</NTag>
        </div>
        <div v-if="result.equal_labels?.length" class="ai-gen-meta">
          <span class="ai-gen-meta-label">{{ t('inhibition.equalLabel') }}:</span>
          <NTag v-for="l in result.equal_labels" :key="l" size="small" style="margin-right: 4px">{{ l }}</NTag>
        </div>
      </template>

      <!-- Mute type preview -->
      <template v-if="isMuteResult(result)">
        <div v-if="result.description" class="ai-gen-desc">{{ result.description }}</div>
        <div v-if="result.match_labels && Object.keys(result.match_labels).length" class="ai-gen-meta">
          <span class="ai-gen-meta-label">{{ t('mute.matchLabel') }}:</span>
          <NTag v-for="(v, k) in result.match_labels" :key="k" size="small" style="margin-right: 4px">{{ k }}={{ v }}</NTag>
        </div>
        <div v-if="result.severities?.length" class="ai-gen-meta">
          <span class="ai-gen-meta-label">{{ t('mute.severities') }}:</span>
          {{ result.severities.join(', ') }}
        </div>
        <div v-if="result.periodic_start" class="ai-gen-meta">
          <span class="ai-gen-meta-label">{{ t('mute.periodicMute') }}:</span>
          {{ result.periodic_start }} → {{ result.periodic_end }}
        </div>
      </template>

      <NAlert v-if="result.warnings?.length" type="warning" style="margin-top: 12px">
        <div v-for="w in result.warnings" :key="w">{{ w }}</div>
      </NAlert>

      <!-- Dry-run & Label Preview (rule type only) -->
      <NCollapse v-if="result && ruleType === 'rule'" class="mt-3">
        <NCollapseItem title="试算（最近 1h）" name="dry-run">
          <NSpace vertical>
            <NButton size="small" :loading="dryRunLoading" @click="handleDryRun">
              运行试算
            </NButton>
            <template v-if="dryRunResult">
              <NSpace>
                <NTag :type="dryRunResult.would_fire ? 'error' : 'success'" size="small">
                  {{ dryRunResult.would_fire ? '会触发' : '不会触发' }}
                </NTag>
                <NText>命中 series: {{ dryRunResult.series_count }} 条</NText>
                <NText>评估耗时: {{ dryRunResult.eval_duration_ms }}ms</NText>
              </NSpace>
              <NDataTable
                v-if="dryRunResult.sample_series?.length"
                :columns="sampleColumns"
                :data="dryRunResult.sample_series"
                size="small"
                :max-height="200"
              />
            </template>
          </NSpace>
        </NCollapseItem>

        <NCollapseItem title="标签命中预览" name="labels">
          <NSpace vertical>
            <NButton size="small" :loading="labelLoading" @click="handleLabelPreview">
              查询命中
            </NButton>
            <template v-if="labelHits.length">
              <div v-for="hit in labelHits" :key="hit.key" class="label-hit-item">
                <NTag :type="hit.matched ? 'success' : 'default'" size="small">
                  {{ hit.key }}
                </NTag>
              </div>
            </template>
          </NSpace>
        </NCollapseItem>
      </NCollapse>

      <NSpace justify="end" style="margin-top: 16px">
        <NButton :loading="generating" @click="handleRegenerate">
          重新生成
        </NButton>
        <template v-if="ruleType === 'rule'">
          <NButton :loading="saving" @click="handleSaveAsDraft">
            保存为草稿
          </NButton>
          <NButton type="primary" :loading="saving" @click="handleSaveAsActive">
            直接启用并保存
          </NButton>
        </template>
        <template v-else>
          <NButton type="primary" @click="handleApply">{{ t('common.apply') }}</NButton>
        </template>
      </NSpace>
    </div>
  </NModal>
</template>

<style scoped>
.ai-gen-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.ai-gen-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.ai-gen-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--sre-text-secondary);
}
.ai-gen-preview {
  margin-top: 20px;
  padding: 16px;
  background: var(--sre-bg-elevated, rgba(255,255,255,0.04));
  border: var(--sre-hairline);
  border-radius: 8px;
}
.ai-gen-preview-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}
.ai-gen-preview-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--sre-text-primary);
}
.ai-gen-expr {
  font-family: var(--sre-font-mono, monospace);
  font-size: 13px;
  color: var(--sre-text-secondary);
  background: var(--sre-bg-card, rgba(0,0,0,0.15));
  padding: 10px 12px;
  border-radius: 6px;
  margin-bottom: 10px;
  word-break: break-all;
}
.ai-gen-desc {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin-bottom: 8px;
}
.ai-gen-meta {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  margin-bottom: 4px;
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.ai-gen-meta-label {
  font-weight: 600;
  color: var(--sre-text-secondary);
}
.mt-3 {
  margin-top: 12px;
}
.label-hit-item {
  display: inline-block;
  margin-right: 6px;
  margin-bottom: 4px;
}
</style>
