<script setup lang="ts">
/**
 * DispatchConfig.vue — Per-channel dispatch policy configuration.
 * Covers Phase 3: 3.1–3.6
 *   - Policy list with priority ordering
 *   - Trigger conditions (label match + time window)
 *   - Delay window (0–3600 s)
 *   - Repeat notification (interval + max repeats)
 *   - Notify mode (personal preference vs unified media)
 *   - Escalation policy binding
 *   - Label enhancement rules (JSON editor)
 */
import { ref, onMounted, computed } from 'vue'
import { useMessage, NButton, NSpace, NSwitch, NInputNumber, NSelect, NTag, NPopconfirm } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { dispatchApi, escalationApi } from '@/api'
import type { DispatchPolicy, EscalationPolicy } from '@/types'
import { getErrorMessage } from '@/utils/format'
import { AddOutline, ArrowUpOutline, ArrowDownOutline } from '@vicons/ionicons5'

const props = defineProps<{ channelId: number }>()
const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const policies = ref<DispatchPolicy[]>([])
const escalationPolicies = ref<EscalationPolicy[]>([])
const showModal = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)

const defaultForm = () => ({
  name: '',
  description: '',
  is_enabled: true,
  priority: 0,
  delay_seconds: 0,
  repeat_interval_seconds: 0,
  max_repeats: 0,
  notify_mode: 'personal_preference',
  unified_media_id: null as number | null,
  escalation_policy_id: null as number | null,
  match_conditions: '',
  active_time_enabled: false,
  active_timezone: 'Asia/Shanghai',
  active_days: [] as number[],
  active_start: '',
  active_end: '',
  label_enhancement_rules: '',
})

const form = ref(defaultForm())

const notifyModeOptions = [
  { label: t('channel.dispatchNotifyPersonal'), value: 'personal_preference' },
  { label: t('channel.dispatchNotifyUnified'), value: 'unified' },
]

const dayOptions = computed(() => [
  { label: t('channel.mon'), value: 1 },
  { label: t('channel.tue'), value: 2 },
  { label: t('channel.wed'), value: 3 },
  { label: t('channel.thu'), value: 4 },
  { label: t('channel.fri'), value: 5 },
  { label: t('channel.sat'), value: 6 },
  { label: t('channel.sun'), value: 0 },
])

const escalationOptions = computed(() => [
  { label: t('channel.dispatchEscalationHint'), value: null },
  ...escalationPolicies.value.map(p => ({ label: p.name, value: p.id })),
])

async function load() {
  loading.value = true
  try {
    const [polRes, escRes] = await Promise.all([
      dispatchApi.list(props.channelId),
      escalationApi.list(),
    ])
    policies.value = polRes.data.data ?? []
    const escData = escRes.data.data
    escalationPolicies.value = Array.isArray(escData) ? escData : (escData as { list?: EscalationPolicy[] })?.list ?? []
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  Object.assign(form.value, defaultForm())
  showModal.value = true
}

function openEdit(policy: DispatchPolicy) {
  editingId.value = policy.id
  const atc = policy.active_time_config ? JSON.parse(policy.active_time_config) : null
  Object.assign(form.value, {
    name: policy.name,
    description: policy.description ?? '',
    is_enabled: policy.is_enabled,
    priority: policy.priority,
    delay_seconds: policy.delay_seconds ?? 0,
    repeat_interval_seconds: policy.repeat_interval_seconds ?? 0,
    max_repeats: policy.max_repeats ?? 0,
    notify_mode: policy.notify_mode ?? 'personal_preference',
    unified_media_id: policy.unified_media_id ?? null,
    escalation_policy_id: policy.escalation_policy_id ?? null,
    match_conditions: policy.match_conditions ?? '',
    label_enhancement_rules: policy.label_enhancement_rules ?? '',
    active_time_enabled: atc?.enabled ?? false,
    active_timezone: atc?.timezone ?? 'Asia/Shanghai',
    active_days: atc?.days_of_week ?? [],
    active_start: atc?.start_time ?? '',
    active_end: atc?.end_time ?? '',
  })
  showModal.value = true
}

function buildPayload() {
  const atc = form.value.active_time_enabled ? JSON.stringify({
    enabled: true,
    timezone: form.value.active_timezone,
    days_of_week: form.value.active_days,
    start_time: form.value.active_start,
    end_time: form.value.active_end,
  }) : ''

  return {
    name: form.value.name,
    description: form.value.description,
    is_enabled: form.value.is_enabled,
    priority: form.value.priority,
    delay_seconds: form.value.delay_seconds,
    repeat_interval_seconds: form.value.repeat_interval_seconds,
    max_repeats: form.value.max_repeats,
    notify_mode: form.value.notify_mode,
    unified_media_id: form.value.unified_media_id ?? undefined,
    escalation_policy_id: form.value.escalation_policy_id ?? undefined,
    match_conditions: form.value.match_conditions,
    active_time_config: atc,
    label_enhancement_rules: form.value.label_enhancement_rules,
  }
}

async function save() {
  if (!form.value.name.trim()) return
  saving.value = true
  try {
    if (editingId.value) {
      await dispatchApi.update(editingId.value, buildPayload())
    } else {
      await dispatchApi.create(props.channelId, buildPayload())
    }
    message.success(t('common.savedSuccess'))
    showModal.value = false
    await load()
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(policy: DispatchPolicy) {
  try {
    await dispatchApi.update(policy.id, { ...policy, is_enabled: !policy.is_enabled })
    policy.is_enabled = !policy.is_enabled
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.failed'))
  }
}

async function deletePolicy(id: number) {
  try {
    await dispatchApi.delete(id)
    message.success(t('common.deleteSuccess'))
    await load()
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.deleteFailed'))
  }
}

async function movePriority(idx: number, dir: -1 | 1) {
  const target = idx + dir
  if (target < 0 || target >= policies.value.length) return
  const a = policies.value[idx]
  const b = policies.value[target]
  try {
    await Promise.all([
      dispatchApi.update(a.id, { ...a, priority: b.priority }),
      dispatchApi.update(b.id, { ...b, priority: a.priority }),
    ])
    await load()
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.failed'))
  }
}

onMounted(load)
</script>

<template>
  <div class="dispatch-config">

    <!-- Header -->
    <div class="section-header">
      <div class="section-title">{{ t('channel.dispatchPolicies') }}</div>
      <n-button type="primary" size="small" @click="openCreate">
        <template #icon><n-icon :component="AddOutline" /></template>
        {{ t('channel.dispatchCreate') }}
      </n-button>
    </div>

    <n-spin :show="loading">
      <!-- Empty state -->
      <div v-if="policies.length === 0 && !loading" class="empty-wrap">
        <n-empty :description="t('channel.dispatchNoPolicy')" size="small" />
      </div>

      <!-- Policy list -->
      <div v-else class="policy-list">
        <div v-for="(policy, idx) in policies" :key="policy.id" class="policy-item">
          <div class="policy-main">
            <div class="policy-header">
              <span class="policy-name">{{ policy.name }}</span>
              <n-tag :type="policy.is_enabled ? 'success' : 'default'" size="small">
                {{ policy.is_enabled ? t('common.enabled') : t('common.disabled') }}
              </n-tag>
              <n-tag size="small" type="info">P{{ policy.priority }}</n-tag>
            </div>
            <div class="policy-meta">
              <span v-if="policy.delay_seconds > 0">{{ t('channel.delaySeconds', { n: policy.delay_seconds }) }}</span>
              <span v-if="policy.repeat_interval_seconds > 0">{{ t('channel.repeatInterval', { n: policy.repeat_interval_seconds }) }}</span>
              <span v-if="policy.escalation_policy_id">{{ t('channel.escalationBound') }}</span>
              <span>{{ policy.notify_mode === 'personal_preference' ? t('channel.dispatchNotifyPersonal') : t('channel.dispatchNotifyUnified') }}</span>
            </div>
          </div>
          <div class="policy-actions">
            <n-button size="tiny" circle quaternary @click="movePriority(idx, -1)" :disabled="idx === 0">
              <template #icon><n-icon :component="ArrowUpOutline" /></template>
            </n-button>
            <n-button size="tiny" circle quaternary @click="movePriority(idx, 1)" :disabled="idx === policies.length - 1">
              <template #icon><n-icon :component="ArrowDownOutline" /></template>
            </n-button>
            <n-switch :value="policy.is_enabled" size="small" @update:value="toggleEnabled(policy)" />
            <n-button size="tiny" @click="openEdit(policy)">{{ t('common.edit') }}</n-button>
            <n-popconfirm @positive-click="deletePolicy(policy.id)">
              <template #trigger>
                <n-button size="tiny" quaternary type="error">{{ t('common.delete') }}</n-button>
              </template>
              {{ t('common.confirmDeleteMsg') }}
            </n-popconfirm>
          </div>
        </div>
      </div>
    </n-spin>

    <!-- Create/Edit Modal -->
    <n-modal
      v-model:show="showModal"
      :title="editingId ? t('common.edit') : t('channel.dispatchCreate')"
      preset="card"
      class="ch-modal-lg"
      :bordered="false"
      :segmented="{ content: true }"
    >
      <n-scrollbar class="ch-form-scroll">
        <n-form label-placement="top" size="small" class="ch-form-pad">

          <!-- Basic -->
          <n-form-item :label="t('common.name')" required>
            <n-input v-model:value="form.name" :placeholder="t('common.name')" />
          </n-form-item>
          <n-form-item :label="t('common.description')">
            <n-input v-model:value="form.description" type="textarea" :rows="2" />
          </n-form-item>
          <n-grid :cols="2" :x-gap="12">
            <n-form-item-gi :label="t('channel.dispatchPriority')">
              <n-input-number v-model:value="form.priority" :min="0" :max="9999" />
            </n-form-item-gi>
            <n-form-item-gi>
              <n-checkbox v-model:checked="form.is_enabled" class="ch-checkbox-top">{{ t('common.enabled') }}</n-checkbox>
            </n-form-item-gi>
          </n-grid>

          <n-divider title-placement="left" class="ch-divider-label">{{ t('channel.triggerAndDelay') }}</n-divider>

          <!-- Delay -->
          <n-form-item :label="t('channel.dispatchDelay')">
            <n-input-number v-model:value="form.delay_seconds" :min="0" :max="3600" class="ch-input-120" />
            <span class="ch-form-hint">{{ t('channel.dispatchDelayHint') }}</span>
          </n-form-item>

          <!-- Repeat -->
          <n-grid :cols="2" :x-gap="12">
            <n-form-item-gi :label="t('channel.dispatchRepeatInterval')">
              <n-input-number v-model:value="form.repeat_interval_seconds" :min="0" class="ch-input-100" />
            </n-form-item-gi>
            <n-form-item-gi :label="t('channel.dispatchMaxRepeats')">
              <n-input-number v-model:value="form.max_repeats" :min="0" class="ch-input-100" />
            </n-form-item-gi>
          </n-grid>

          <!-- Match conditions -->
          <n-form-item :label="t('channel.dispatchMatchConditions')">
            <n-input
              v-model:value="form.match_conditions"
              type="textarea"
              :rows="2"
              :placeholder='t("channel.dispatchMatchConditionsHint")'
            />
          </n-form-item>

          <!-- Active time -->
          <n-form-item>
            <n-checkbox v-model:checked="form.active_time_enabled">
              {{ t('channel.dispatchActiveTimeEnabled') }}
            </n-checkbox>
          </n-form-item>

          <template v-if="form.active_time_enabled">
            <n-form-item :label="t('channel.dispatchDaysOfWeek')">
              <n-checkbox-group v-model:value="form.active_days">
                <n-space>
                  <n-checkbox v-for="d in dayOptions" :key="d.value" :value="d.value" :label="d.label" />
                </n-space>
              </n-checkbox-group>
            </n-form-item>
            <n-grid :cols="2" :x-gap="12">
              <n-form-item-gi :label="t('channel.dispatchStartTime')">
                <n-input v-model:value="form.active_start" placeholder="09:00" class="ch-input-100" />
              </n-form-item-gi>
              <n-form-item-gi :label="t('channel.dispatchEndTime')">
                <n-input v-model:value="form.active_end" placeholder="18:00" class="ch-input-100" />
              </n-form-item-gi>
            </n-grid>
          </template>

          <n-divider title-placement="left" class="ch-divider-label">{{ t('channel.notifyAndEscalation') }}</n-divider>

          <!-- Notify mode -->
          <n-form-item :label="t('channel.dispatchNotifyMode')">
            <n-radio-group v-model:value="form.notify_mode">
              <n-radio value="personal_preference">{{ t('channel.dispatchNotifyPersonal') }}</n-radio>
              <n-radio value="unified">{{ t('channel.dispatchNotifyUnified') }}</n-radio>
            </n-radio-group>
          </n-form-item>

          <!-- Escalation policy -->
          <n-form-item :label="t('channel.dispatchEscalationPolicy')">
            <n-select
              v-model:value="form.escalation_policy_id"
              :options="escalationOptions"
              clearable
              class="ch-select-260"
            />
          </n-form-item>

          <n-divider title-placement="left" class="ch-divider-label">{{ t('channel.labelEnhancement') }}</n-divider>

          <!-- Label enhancement rules -->
          <n-form-item :label="t('channel.dispatchLabelRules')">
            <n-input
              v-model:value="form.label_enhancement_rules"
              type="textarea"
              :rows="4"
              :placeholder='t("channel.dispatchLabelRulesHint")'
            />
          </n-form-item>

        </n-form>
      </n-scrollbar>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="save">{{ t('common.save') }}</n-button>
        </n-space>
      </template>
    </n-modal>

  </div>
</template>

<style scoped>
.dispatch-config { max-width: 800px; }

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
}

.empty-wrap { padding: 24px 0; }

.policy-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.policy-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  background: var(--sre-bg-page);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  gap: 12px;
}

.policy-main { flex: 1; min-width: 0; }

.policy-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.policy-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-primary);
}

.policy-meta {
  display: flex;
  gap: 12px;
  font-size: 11px;
  color: var(--sre-text-secondary);
}

.policy-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

</style>

<style>
@import '@/styles/channels.css';
</style>
