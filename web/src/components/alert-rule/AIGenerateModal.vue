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
interface LabelHit {
  key: string
  matched: boolean
  ruleValue: string
  registryValues: string[]
}
const labelHits = ref<LabelHit[]>([])
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
  const ruleResult = result.value as RuleGenerateResult
  const dsId = selectedDatasourceId.value
  if (!dsId) {
    message.warning(t('aiGenerate.selectDatasourceFirst'))
    return
  }
  saving.value = true
  try {
    await alertRuleApi.create({
      name: ruleResult.name,
      expression: ruleResult.expression,
      datasource_id: dsId,
      for_duration: ruleResult.for_duration || '5m',
      severity: ruleResult.severity as AlertSeverity | undefined,
      labels: ruleResult.labels || {},
      annotations: ruleResult.annotations || {},
      description: ruleResult.description,
      status: 'draft',
      enabled: false,
    })
    message.success(t('aiGenerate.savedAsDraft'))
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
  const dsId = selectedDatasourceId.value
  if (!dsId) {
    message.warning(t('aiGenerate.selectDatasourceFirst'))
    return
  }
  saving.value = true
  try {
    await alertRuleApi.create({
      name: ruleResult.name,
      expression: ruleResult.expression,
      datasource_id: dsId,
      for_duration: ruleResult.for_duration || '5m',
      severity: ruleResult.severity as AlertSeverity | undefined,
      labels: ruleResult.labels || {},
      annotations: ruleResult.annotations || {},
      description: ruleResult.description,
      status: 'active',
      enabled: true,
    })
    message.success(t('aiGenerate.ruleCreatedEnabled'))
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
  const ruleResult = result.value as RuleGenerateResult
  const dsId = selectedDatasourceId.value
  if (!dsId) {
    message.warning(t('aiGenerate.selectDatasourceFirst'))
    return
  }
  dryRunLoading.value = true
  try {
    const resp = await aiRuleApi.dryRun({
      datasource_id: dsId,
      expr: ruleResult.expression || '',
      for: ruleResult.for_duration || '5m',
      labels: ruleResult.labels || {},
      annotations: ruleResult.annotations || {},
      severity: ruleResult.severity || 'warning',
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
  if (!result.value) return
  const ruleResult = result.value as RuleGenerateResult
  const dsId = selectedDatasourceId.value
  if (!dsId) {
    message.warning(t('aiGenerate.selectDatasourceFirst'))
    return
  }
  labelLoading.value = true
  try {
    const ruleLabels = ruleResult.labels || {}
    const hits: LabelHit[] = []
    for (const [key, val] of Object.entries(ruleLabels)) {
      try {
        const resp = await datasourceApi.labelValues(dsId, key)
        const registryValues: string[] = (resp.data.data ?? []).slice(0, 10)
        hits.push({
          key,
          matched: registryValues.includes(String(val)),
          ruleValue: String(val),
          registryValues,
        })
      } catch {
        hits.push({ key, matched: false, ruleValue: String(val), registryValues: [] })
      }
    }
    labelHits.value = hits
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
        <NCollapseItem :title="t('aiGenerate.dryRun')" name="dry-run">
          <NSpace vertical>
            <NButton size="small" :loading="dryRunLoading" @click="handleDryRun">
              {{ t('aiGenerate.runDryRun') }}
            </NButton>
            <template v-if="dryRunResult">
              <NSpace>
                <NTag :type="dryRunResult.would_fire ? 'error' : 'success'" size="small">
                  {{ dryRunResult.would_fire ? t('aiGenerate.wouldFire') : t('aiGenerate.wouldNotFire') }}
                </NTag>
                <NText>{{ t('aiGenerate.matchedSeries', { count: dryRunResult.series_count }) }}</NText>
                <NText>{{ t('aiGenerate.evalDuration', { ms: dryRunResult.eval_duration_ms }) }}</NText>
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

        <NCollapseItem :title="t('aiGenerate.labelPreview')" name="labels">
          <NSpace vertical>
            <NButton size="small" :loading="labelLoading" @click="handleLabelPreview">
              {{ t('aiGenerate.queryMatches') }}
            </NButton>
            <template v-if="labelHits.length">
              <div v-for="hit in labelHits" :key="hit.key" class="label-hit-item">
                <NTag :type="hit.matched ? 'success' : 'warning'" size="small">
                  {{ hit.matched ? '✓' : '⚠' }} {{ hit.key }}
                </NTag>
                <span class="label-hit-detail">{{ t('aiGenerate.ruleValue') }}: {{ hit.ruleValue }}</span>
                <span v-if="hit.registryValues.length" class="label-hit-detail">
                  {{ t('aiGenerate.existing') }}: {{ hit.registryValues.join(', ') }}
                </span>
              </div>
            </template>
          </NSpace>
        </NCollapseItem>
      </NCollapse>

      <NSpace justify="end" style="margin-top: 16px">
        <NButton :loading="generating" @click="handleRegenerate">
          {{ t('aiGenerate.regenerate') }}
        </NButton>
        <template v-if="ruleType === 'rule'">
          <NButton :loading="saving" @click="handleSaveAsDraft">
            {{ t('aiGenerate.saveAsDraft') }}
          </NButton>
          <NButton type="primary" :loading="saving" @click="handleSaveAsActive">
            {{ t('aiGenerate.saveAndEnable') }}
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
.label-hit-detail {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  margin-left: 4px;
}
</style>
