<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NInput,
  NSelect,
  NSwitch,
  NButton,
  NSpace,
  NFormItem,
  NIcon,
} from 'naive-ui'
import { AddOutline, TrashOutline } from '@vicons/ionicons5'
import type { VariableConfig } from '@/types/dashboard'
import type { DataSource } from '@/types'

const { t } = useI18n()

const props = defineProps<{
  variable: VariableConfig
  datasources: DataSource[]
}>()

const emit = defineEmits<{
  'update:variable': [value: VariableConfig]
}>()

const typeOptions = computed(() => [
  { label: t('dashboardEditor.varTypeQuery'), value: 'query' },
  { label: t('dashboardEditor.varTypeCustom'), value: 'custom' },
  { label: t('dashboardEditor.varTypeInterval'), value: 'interval' },
  { label: t('dashboardEditor.varTypeDatasource'), value: 'datasource' },
  { label: t('dashboardEditor.varTypeTextbox'), value: 'textbox' },
  { label: t('dashboardEditor.varTypeConstant'), value: 'constant' },
  { label: t('dashboardEditor.varTypeAdhoc'), value: 'adhoc' },
])

const sortOptions = computed(() => [
  { label: t('dashboardEditor.sortDisabled'), value: 'disabled' },
  { label: t('dashboardEditor.sortAscending'), value: 'asc' },
  { label: t('dashboardEditor.sortDescending'), value: 'desc' },
  { label: t('dashboardEditor.sortNumAscending'), value: 'numerical-asc' },
  { label: t('dashboardEditor.sortNumDescending'), value: 'numerical-desc' },
])

const refreshOptions = computed(() => [
  { label: t('dashboardEditor.refreshOnLoad'), value: 'onLoad' },
  { label: t('dashboardEditor.refreshOnTimeChange'), value: 'onTimeRangeChange' },
  { label: t('dashboardEditor.refreshNever'), value: 'never' },
])

const datasourceOptions = computed(() =>
  props.datasources.map(ds => ({ label: ds.name, value: ds.id }))
)

// Helper to emit partial updates
function update<K extends keyof VariableConfig>(key: K, value: VariableConfig[K]) {
  emit('update:variable', { ...props.variable, [key]: value })
}

// Custom options management.
// options is a string[] so rows can't carry their own identity — keep a
// parallel array of stable ids, maintained positionally on add/remove and
// left UNTOUCHED on edit. Keying by the option value (or index) would
// regenerate the key on every keystroke and rebuild the input (focus loss).
let _nextOptId = 0
const optIds = ref<number[]>([])

// Resync when the options array is replaced externally (e.g. switching the
// edited variable). Length-based: grow with fresh ids, trim from the end.
watch(
  () => (props.variable.options || []).length,
  (len) => {
    while (optIds.value.length < len) optIds.value.push(++_nextOptId)
    if (optIds.value.length > len) optIds.value.length = len
  },
  { immediate: true },
)

function optKey(idx: number): number | string {
  // Fallback covers the interval type's implicit default options
  // (variable.options undefined) which have no tracked ids.
  return optIds.value[idx] ?? `d${idx}`
}

function addOption() {
  const opts = [...(props.variable.options || []), '']
  optIds.value.push(++_nextOptId)
  update('options', opts)
}

function removeOption(index: number) {
  const opts = [...(props.variable.options || [])]
  opts.splice(index, 1)
  optIds.value.splice(index, 1)
  update('options', opts)
}

function updateOption(index: number, value: string) {
  const opts = [...(props.variable.options || [])]
  opts[index] = value
  // ids untouched: same row, same key, input keeps focus
  update('options', opts)
}
</script>

<template>
  <div class="variable-editor-item">
    <!-- Name -->
    <NFormItem :label="t('dashboardEditor.varName')" required>
      <NInput
        :value="variable.name"
        :placeholder="t('dashboardEditor.placeholderName')"
        @update:value="(v: string) => update('name', v)"
      />
    </NFormItem>

    <!-- Label -->
    <NFormItem :label="t('dashboardEditor.varLabel')">
      <NInput
        :value="variable.label"
        :placeholder="t('dashboardEditor.placeholderLabel')"
        @update:value="(v: string) => update('label', v)"
      />
    </NFormItem>

    <!-- Type -->
    <NFormItem :label="t('dashboardEditor.varType')" required>
      <NSelect
        :value="variable.type"
        :options="typeOptions"
        @update:value="(v: string) => update('type', v as VariableConfig['type'])"
      />
    </NFormItem>

    <!-- Query type fields -->
    <template v-if="variable.type === 'query'">
      <NFormItem :label="t('dashboardEditor.varDatasource')">
        <NSelect
          :value="variable.datasourceId"
          :options="datasourceOptions"
          :placeholder="t('dashboardEditor.placeholderDatasource')"
          clearable
          @update:value="(v: number) => update('datasourceId', v)"
        />
      </NFormItem>
      <NFormItem :label="t('dashboardEditor.varQuery')">
        <NInput
          type="textarea"
          :value="variable.query"
          :placeholder="t('dashboardEditor.placeholderQuery')"
          :rows="3"
          @update:value="(v: string) => update('query', v)"
        />
      </NFormItem>
      <NFormItem :label="t('dashboardEditor.varRegex')">
        <NInput
          :value="variable.regex"
          :placeholder="t('dashboardEditor.placeholderRegex')"
          @update:value="(v: string) => update('regex', v)"
        />
      </NFormItem>
      <NFormItem :label="t('dashboardEditor.varSort')">
        <NSelect
          :value="variable.sort"
          :options="sortOptions"
          @update:value="(v: string) => update('sort', v as VariableConfig['sort'])"
        />
      </NFormItem>
      <NFormItem :label="t('dashboardEditor.varRefresh')">
        <NSelect
          :value="variable.refresh"
          :options="refreshOptions"
          @update:value="(v: string) => update('refresh', v as VariableConfig['refresh'])"
        />
      </NFormItem>
    </template>

    <!-- Custom type fields -->
    <template v-if="variable.type === 'custom'">
      <NFormItem :label="t('dashboardEditor.varOptions')">
        <div class="custom-options">
          <div v-for="(opt, idx) in (variable.options || [])" :key="optKey(idx)" class="option-row">
            <NInput
              :value="opt"
              size="small"
              :placeholder="t('dashboardEditor.placeholderOptionValue')"
              @update:value="(v: string) => updateOption(idx, v)"
            />
            <NButton quaternary size="small" type="error" @click="removeOption(idx)">
              <template #icon><NIcon :component="TrashOutline" /></template>
            </NButton>
          </div>
          <NButton size="small" dashed @click="addOption">
            <template #icon><NIcon :component="AddOutline" /></template>
            {{ t('dashboardEditor.addOption') }}
          </NButton>
        </div>
      </NFormItem>
    </template>

    <!-- Interval type fields -->
    <template v-if="variable.type === 'interval'">
      <NFormItem :label="t('dashboardEditor.varIntervalOptions')">
        <div class="custom-options">
          <div v-for="(opt, idx) in (variable.options || ['1m', '5m', '10m', '30m', '1h'])" :key="optKey(idx)" class="option-row">
            <NInput
              :value="opt"
              size="small"
              placeholder="e.g. 5m"
              @update:value="(v: string) => updateOption(idx, v)"
            />
            <NButton quaternary size="small" type="error" @click="removeOption(idx)">
              <template #icon><NIcon :component="TrashOutline" /></template>
            </NButton>
          </div>
          <NButton size="small" dashed @click="addOption">
            <template #icon><NIcon :component="AddOutline" /></template>
            {{ t('dashboardEditor.addInterval') }}
          </NButton>
        </div>
      </NFormItem>
    </template>

    <!-- Datasource type fields -->
    <template v-if="variable.type === 'datasource'">
      <NFormItem :label="t('dashboardEditor.varDatasourceFilter')">
        <NSelect
          :value="variable.datasourceId"
          :options="datasourceOptions"
          :placeholder="t('dashboardEditor.placeholderBaseDatasource')"
          clearable
          @update:value="(v: number) => update('datasourceId', v)"
        />
      </NFormItem>
      <NFormItem :label="t('dashboardEditor.varRegex')">
        <NInput
          :value="variable.regex"
          :placeholder="t('dashboardEditor.placeholderFilterDatasource')"
          @update:value="(v: string) => update('regex', v)"
        />
      </NFormItem>
    </template>

    <!-- Textbox type fields -->
    <template v-if="variable.type === 'textbox'">
      <NFormItem :label="t('dashboardEditor.varDefaultValue')">
        <NInput
          :value="variable.defaultValue"
          :placeholder="t('dashboardEditor.placeholderDefaultText')"
          @update:value="(v: string) => update('defaultValue', v)"
        />
      </NFormItem>
    </template>

    <!-- Constant type fields -->
    <template v-if="variable.type === 'constant'">
      <NFormItem :label="t('dashboardEditor.varDefaultValue')">
        <NInput
          :value="variable.defaultValue"
          :placeholder="t('dashboardEditor.placeholderConstantValue')"
          @update:value="(v: string) => update('defaultValue', v)"
        />
      </NFormItem>
    </template>

    <!-- Adhoc type fields -->
    <template v-if="variable.type === 'adhoc'">
      <NFormItem :label="t('dashboardEditor.varDatasource')">
        <NSelect
          :value="variable.datasourceId"
          :options="datasourceOptions"
          :placeholder="t('dashboardEditor.placeholderAdhocDatasource')"
          clearable
          @update:value="(v: number) => update('datasourceId', v)"
        />
      </NFormItem>
    </template>

    <!-- Common fields: multi / includeAll / allValue / defaultValue -->
    <NFormItem v-if="variable.type !== 'constant' && variable.type !== 'adhoc'" :label="t('dashboardEditor.varDefaultValue')">
      <NInput
        :value="variable.defaultValue"
        :placeholder="t('dashboardEditor.placeholderDefaultValue')"
        @update:value="(v: string) => update('defaultValue', v)"
      />
    </NFormItem>

    <NFormItem :label="t('dashboardEditor.varMultiSelect')">
      <NSwitch
        :value="variable.multi"
        @update:value="(v: boolean) => update('multi', v)"
      />
    </NFormItem>

    <NFormItem :label="t('dashboardEditor.varIncludeAll')">
      <NSwitch
        :value="variable.includeAll"
        @update:value="(v: boolean) => update('includeAll', v)"
      />
    </NFormItem>

    <NFormItem v-if="variable.includeAll" :label="t('dashboardEditor.varAllValue')">
      <NInput
        :value="variable.allValue || '$__all'"
        placeholder="$__all"
        @update:value="(v: string) => update('allValue', v)"
      />
    </NFormItem>
  </div>
</template>

<style scoped>
.variable-editor-item {
  padding: 8px 0;
}
.custom-options {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}
.option-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
