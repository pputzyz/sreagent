<script setup lang="ts">
/**
 * NoiseConfig.vue — Per-channel noise reduction configuration panel.
 * Covers:
 *  - Aggregation rules (unified / fine-grained + window + storm thresholds)
 *  - Flapping detection (off / notify_only / notify_then_silence)
 *  - Exclusion rules (CRUD)
 */
import { ref, onMounted, reactive } from 'vue'
import { useMessage, NButton, NSpace, NSwitch, NInputNumber, NSelect, NTag, NPopconfirm } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { channelV2Api } from '@/api'
import { AddOutline } from '@vicons/ionicons5'

const props = defineProps<{ channelId: number }>()
const { t } = useI18n()
const message = useMessage()

// ---- Aggregation config ----
const aggConfig = reactive({
  enabled: false,
  mode: 'unified',          // unified | fine_grained
  dimensions: [] as string[],
  window_enabled: false,
  window_origin: 'triggered',
  window_minutes: 5,
  storm_thresholds: [] as number[],
  strict_mode: false,
})

// ---- Flapping config ----
const flapConfig = reactive({
  mode: 'off',              // off | notify_only | notify_then_silence
  max_changes: 5,
  window_minutes: 30,
  mute_minutes: 60,
})

const saving = ref(false)
const loadingExclusion = ref(false)

const dimensionInput = ref('')
const stormThresholdInput = ref<number | null>(null)

// ---- Exclusion rules ----
const exclusionRules = ref<any[]>([])
const showAddExclusionModal = ref(false)
const addingExclusion = ref(false)
const newExclusionForm = ref({
  name: '',
  description: '',
  conditions: '[]',
  is_enabled: true,
  priority: 0,
})

const modeOptions = [
  { label: t('channel.aggregationUnified'), value: 'unified' },
  { label: t('channel.aggregationFineGrained'), value: 'fine_grained' },
]
const flappingModeOptions = [
  { label: t('channel.flappingModeOff'), value: 'off' },
  { label: t('channel.flappingModeNotify'), value: 'notify_only' },
  { label: t('channel.flappingModeSilence'), value: 'notify_then_silence' },
]
const windowOriginOptions = [
  { label: t('channel.aggregationWindowOriginTriggered'), value: 'triggered' },
  { label: t('channel.aggregationWindowOriginMerged'), value: 'alert_merged' },
]

async function load() {
  try {
    const ch = await channelV2Api.get(props.channelId)
    const data = ch.data.data
    if (!data) return

    if (data.aggregation_config) {
      try {
        const parsed = JSON.parse(data.aggregation_config)
        Object.assign(aggConfig, {
          enabled: parsed.enabled ?? false,
          mode: parsed.mode ?? 'unified',
          dimensions: parsed.dimensions ?? [],
          window_enabled: parsed.window_enabled ?? false,
          window_origin: parsed.window_origin ?? 'triggered',
          window_minutes: parsed.window_minutes ?? 5,
          storm_thresholds: parsed.storm_thresholds ?? [],
          strict_mode: parsed.strict_mode ?? false,
        })
      } catch {}
    }

    if (data.flapping_config) {
      try {
        const parsed = JSON.parse(data.flapping_config)
        Object.assign(flapConfig, {
          mode: parsed.mode ?? 'off',
          max_changes: parsed.max_changes ?? 5,
          window_minutes: parsed.window_minutes ?? 30,
          mute_minutes: parsed.mute_minutes ?? 60,
        })
      } catch {}
    }
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
  }

  await loadExclusionRules()
}

async function loadExclusionRules() {
  loadingExclusion.value = true
  try {
    const res = await channelV2Api.listExclusionRules(props.channelId)
    exclusionRules.value = res.data.data ?? []
  } catch {
    exclusionRules.value = []
  } finally {
    loadingExclusion.value = false
  }
}

async function saveNoise() {
  saving.value = true
  try {
    await channelV2Api.updateNoiseConfig(props.channelId, {
      aggregation_config: JSON.stringify(aggConfig),
      flapping_config: JSON.stringify(flapConfig),
    })
    message.success(t('common.savedSuccess'))
  } catch (e: any) {
    message.error(e?.message ?? t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

function addDimension() {
  const val = dimensionInput.value.trim()
  if (!val || aggConfig.dimensions.includes(val)) return
  if (aggConfig.dimensions.length >= 5) {
    message.warning('最多 5 个维度')
    return
  }
  aggConfig.dimensions.push(val)
  dimensionInput.value = ''
}

function removeDimension(dim: string) {
  aggConfig.dimensions = aggConfig.dimensions.filter(d => d !== dim)
}

function addStormThreshold() {
  const v = stormThresholdInput.value
  if (!v || v < 2 || aggConfig.storm_thresholds.includes(v)) return
  if (aggConfig.storm_thresholds.length >= 5) {
    message.warning('最多 5 个阈值')
    return
  }
  aggConfig.storm_thresholds.push(v)
  aggConfig.storm_thresholds.sort((a, b) => a - b)
  stormThresholdInput.value = null
}

function removeStormThreshold(v: number) {
  aggConfig.storm_thresholds = aggConfig.storm_thresholds.filter(t => t !== v)
}

async function addExclusionRule() {
  if (!newExclusionForm.value.name.trim()) return
  addingExclusion.value = true
  try {
    await channelV2Api.createExclusionRule(props.channelId, newExclusionForm.value)
    message.success(t('common.createSuccess'))
    showAddExclusionModal.value = false
    newExclusionForm.value = { name: '', description: '', conditions: '[]', is_enabled: true, priority: 0 }
    await loadExclusionRules()
  } catch (e: any) {
    message.error(e?.message ?? t('common.failed'))
  } finally {
    addingExclusion.value = false
  }
}

async function deleteExclusionRule(ruleId: number) {
  try {
    await channelV2Api.deleteExclusionRule(ruleId)
    message.success(t('common.deleteSuccess'))
    await loadExclusionRules()
  } catch (e: any) {
    message.error(e?.message ?? t('common.deleteFailed'))
  }
}

async function toggleExclusionRule(rule: any) {
  try {
    await channelV2Api.updateExclusionRule(rule.id, { is_enabled: !rule.is_enabled })
    rule.is_enabled = !rule.is_enabled
  } catch (e: any) {
    message.error(e?.message ?? t('common.failed'))
  }
}

onMounted(load)
</script>

<template>
  <div class="noise-config">

    <!-- ===== Section 1: Aggregation ===== -->
    <div class="section">
      <div class="section-title">{{ t('channel.aggregation') }}</div>
      <n-form label-placement="left" :label-width="160" size="small">

        <n-form-item :label="t('channel.aggregationEnabled')">
          <n-switch v-model:value="aggConfig.enabled" />
        </n-form-item>

        <template v-if="aggConfig.enabled">
          <n-form-item :label="t('channel.aggregationMode')">
            <n-radio-group v-model:value="aggConfig.mode">
              <n-radio value="unified">{{ t('channel.aggregationUnified') }}</n-radio>
              <n-radio value="fine_grained">{{ t('channel.aggregationFineGrained') }}</n-radio>
            </n-radio-group>
          </n-form-item>

          <n-form-item :label="t('channel.aggregationDimensions')">
            <div class="tag-input-group">
              <n-space wrap style="margin-bottom:6px">
                <n-tag
                  v-for="dim in aggConfig.dimensions"
                  :key="dim"
                  closable
                  size="small"
                  @close="removeDimension(dim)"
                >{{ dim }}</n-tag>
              </n-space>
              <n-input-group>
                <n-input
                  v-model:value="dimensionInput"
                  :placeholder="t('channel.aggregationDimensionsHint')"
                  size="small"
                  style="width:200px"
                  @keyup.enter="addDimension"
                />
                <n-button size="small" @click="addDimension">+</n-button>
              </n-input-group>
            </div>
          </n-form-item>

          <n-form-item :label="t('channel.aggregationWindowEnabled')">
            <n-switch v-model:value="aggConfig.window_enabled" />
          </n-form-item>

          <template v-if="aggConfig.window_enabled">
            <n-form-item :label="t('channel.aggregationWindowOrigin')">
              <n-select v-model:value="aggConfig.window_origin" :options="windowOriginOptions" style="width:200px" />
            </n-form-item>
            <n-form-item :label="t('channel.aggregationWindowMinutes')">
              <n-input-number v-model:value="aggConfig.window_minutes" :min="1" :max="1440" style="width:120px" />
            </n-form-item>
          </template>

          <n-form-item :label="t('channel.stormThresholds')">
            <div class="tag-input-group">
              <n-space wrap style="margin-bottom:6px">
                <n-tag
                  v-for="v in aggConfig.storm_thresholds"
                  :key="v"
                  closable
                  size="small"
                  type="warning"
                  @close="removeStormThreshold(v)"
                >{{ v }}</n-tag>
              </n-space>
              <n-input-group>
                <n-input-number
                  v-model:value="stormThresholdInput"
                  :min="2"
                  :max="10000"
                  :placeholder="t('channel.stormThresholdsHint')"
                  size="small"
                  style="width:140px"
                />
                <n-button size="small" @click="addStormThreshold">+</n-button>
              </n-input-group>
            </div>
          </n-form-item>

          <n-form-item :label="t('channel.strictMode')">
            <n-space align="center">
              <n-switch v-model:value="aggConfig.strict_mode" />
              <span style="font-size:12px;color:var(--sre-text-secondary)">
                {{ t('channel.strictModeHint') }}
              </span>
            </n-space>
          </n-form-item>
        </template>
      </n-form>
    </div>

    <!-- ===== Section 2: Flapping detection ===== -->
    <div class="section">
      <div class="section-title">{{ t('channel.flapping') }}</div>
      <n-form label-placement="left" :label-width="160" size="small">

        <n-form-item :label="t('channel.flappingMode')">
          <n-select v-model:value="flapConfig.mode" :options="flappingModeOptions" style="width:200px" />
        </n-form-item>

        <template v-if="flapConfig.mode !== 'off'">
          <n-form-item :label="t('channel.flappingMaxChanges')">
            <n-input-number v-model:value="flapConfig.max_changes" :min="2" :max="100" style="width:100px" />
          </n-form-item>
          <n-form-item :label="t('channel.flappingWindowMinutes')">
            <n-input-number v-model:value="flapConfig.window_minutes" :min="1" :max="1440" style="width:100px" />
          </n-form-item>
          <n-form-item v-if="flapConfig.mode === 'notify_then_silence'" :label="t('channel.flappingMuteMinutes')">
            <n-input-number v-model:value="flapConfig.mute_minutes" :min="30" :max="1440" style="width:100px" />
          </n-form-item>
        </template>
      </n-form>
    </div>

    <!-- Save button -->
    <div class="save-row">
      <n-button type="primary" :loading="saving" @click="saveNoise">
        {{ t('common.save') }}
      </n-button>
    </div>

    <!-- ===== Section 3: Exclusion rules ===== -->
    <div class="section">
      <div class="section-header">
        <div class="section-title">{{ t('channel.exclusionRules') }}</div>
        <n-button size="small" type="primary" @click="showAddExclusionModal = true">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('channel.addExclusionRule') }}
        </n-button>
      </div>
      <p class="section-hint">{{ t('channel.exclusionRulesHint') }}</p>

      <div v-if="exclusionRules.length === 0 && !loadingExclusion" class="empty-rules">
        <n-empty :description="t('channel.noExclusionRules')" size="small" />
      </div>

      <div v-else class="exclusion-list">
        <div
          v-for="rule in exclusionRules"
          :key="rule.id"
          class="exclusion-item"
        >
          <div class="excl-info">
            <span class="excl-name">{{ rule.name }}</span>
            <span v-if="rule.description" class="excl-desc">{{ rule.description }}</span>
          </div>
          <div class="excl-actions">
            <n-switch
              :value="rule.is_enabled"
              size="small"
              @update:value="toggleExclusionRule(rule)"
            />
            <n-popconfirm @positive-click="deleteExclusionRule(rule.id)">
              <template #trigger>
                <n-button size="tiny" quaternary type="error">{{ t('common.delete') }}</n-button>
              </template>
              {{ t('common.confirmDeleteMsg') }}
            </n-popconfirm>
          </div>
        </div>
      </div>
    </div>

    <!-- Add exclusion rule modal -->
    <n-modal
      v-model:show="showAddExclusionModal"
      :title="t('channel.addExclusionRule')"
      preset="card"
      style="width:440px"
      :bordered="false"
    >
      <n-form label-placement="top" size="small">
        <n-form-item :label="t('channel.exclusionRuleName')" required>
          <n-input v-model:value="newExclusionForm.name" :placeholder="t('channel.exclusionRuleName')" />
        </n-form-item>
        <n-form-item :label="t('common.description')">
          <n-input v-model:value="newExclusionForm.description" :placeholder="t('common.description')" />
        </n-form-item>
        <n-form-item :label="t('channel.exclusionRuleConditions')">
          <n-input
            v-model:value="newExclusionForm.conditions"
            type="textarea"
            :rows="4"
            placeholder='[{"field":"severity","operator":"eq","value":"info"}]'
          />
          <template #feedback>
            <span style="font-size:11px;color:var(--sre-text-secondary)">
              JSON 数组，field 支持 severity/alertname/labels.xxx，operator 支持 eq/ne/contains/regex/in
            </span>
          </template>
        </n-form-item>
        <n-form-item label="Priority">
          <n-input-number v-model:value="newExclusionForm.priority" :min="0" style="width:100px" />
        </n-form-item>
        <n-form-item>
          <n-checkbox v-model:checked="newExclusionForm.is_enabled">{{ t('common.enabled') }}</n-checkbox>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showAddExclusionModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="addingExclusion" @click="addExclusionRule">
            {{ t('common.create') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.noise-config { max-width: 680px; }

.section {
  margin-bottom: 28px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--sre-border);
}

.section:last-child { border-bottom: none; }

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  margin-bottom: 14px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.section-hint {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin-bottom: 12px;
}

.tag-input-group { display: flex; flex-direction: column; }

.save-row {
  margin-bottom: 32px;
}

.empty-rules {
  padding: 16px 0;
}

.exclusion-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.exclusion-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  background: var(--sre-bg-page);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
}

.excl-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.excl-name {
  font-size: 13px;
  font-weight: 500;
}

.excl-desc {
  font-size: 11px;
  color: var(--sre-text-secondary);
}

.excl-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
