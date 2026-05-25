<script setup lang="ts">
import { computed } from 'vue'
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

const props = defineProps<{
  variable: VariableConfig
  datasources: DataSource[]
}>()

const emit = defineEmits<{
  'update:variable': [value: VariableConfig]
}>()

const typeOptions = [
  { label: 'Query', value: 'query' },
  { label: 'Custom', value: 'custom' },
  { label: 'Interval', value: 'interval' },
  { label: 'Datasource', value: 'datasource' },
  { label: 'Textbox', value: 'textbox' },
  { label: 'Constant', value: 'constant' },
  { label: 'Adhoc', value: 'adhoc' },
]

const sortOptions = [
  { label: 'Disabled', value: 'disabled' },
  { label: 'Ascending', value: 'asc' },
  { label: 'Descending', value: 'desc' },
  { label: 'Numerical Asc', value: 'numerical-asc' },
  { label: 'Numerical Desc', value: 'numerical-desc' },
]

const refreshOptions = [
  { label: 'On Load', value: 'onLoad' },
  { label: 'On Time Range Change', value: 'onTimeRangeChange' },
  { label: 'Never', value: 'never' },
]

const datasourceOptions = computed(() =>
  props.datasources.map(ds => ({ label: ds.name, value: ds.id }))
)

// Helper to emit partial updates
function update<K extends keyof VariableConfig>(key: K, value: VariableConfig[K]) {
  emit('update:variable', { ...props.variable, [key]: value })
}

// Custom options management
function addOption() {
  const opts = [...(props.variable.options || []), '']
  update('options', opts)
}

function removeOption(index: number) {
  const opts = [...(props.variable.options || [])]
  opts.splice(index, 1)
  update('options', opts)
}

function updateOption(index: number, value: string) {
  const opts = [...(props.variable.options || [])]
  opts[index] = value
  update('options', opts)
}
</script>

<template>
  <div class="variable-editor-item">
    <!-- Name -->
    <NFormItem label="Name" required>
      <NInput
        :value="variable.name"
        placeholder="e.g. host"
        @update:value="(v: string) => update('name', v)"
      />
    </NFormItem>

    <!-- Label -->
    <NFormItem label="Label">
      <NInput
        :value="variable.label"
        placeholder="e.g. Host"
        @update:value="(v: string) => update('label', v)"
      />
    </NFormItem>

    <!-- Type -->
    <NFormItem label="Type" required>
      <NSelect
        :value="variable.type"
        :options="typeOptions"
        @update:value="(v: string) => update('type', v as VariableConfig['type'])"
      />
    </NFormItem>

    <!-- Query type fields -->
    <template v-if="variable.type === 'query'">
      <NFormItem label="Datasource">
        <NSelect
          :value="variable.datasourceId"
          :options="datasourceOptions"
          placeholder="Select datasource"
          clearable
          @update:value="(v: number) => update('datasourceId', v)"
        />
      </NFormItem>
      <NFormItem label="Query (PromQL)">
        <NInput
          type="textarea"
          :value="variable.query"
          placeholder='label_values(up, job)'
          :rows="3"
          @update:value="(v: string) => update('query', v)"
        />
      </NFormItem>
      <NFormItem label="Regex">
        <NInput
          :value="variable.regex"
          placeholder="Optional regex filter"
          @update:value="(v: string) => update('regex', v)"
        />
      </NFormItem>
      <NFormItem label="Sort">
        <NSelect
          :value="variable.sort"
          :options="sortOptions"
          @update:value="(v: string) => update('sort', v as VariableConfig['sort'])"
        />
      </NFormItem>
      <NFormItem label="Refresh">
        <NSelect
          :value="variable.refresh"
          :options="refreshOptions"
          @update:value="(v: string) => update('refresh', v as VariableConfig['refresh'])"
        />
      </NFormItem>
    </template>

    <!-- Custom type fields -->
    <template v-if="variable.type === 'custom'">
      <NFormItem label="Options">
        <div class="custom-options">
          <div v-for="(opt, idx) in (variable.options || [])" :key="idx" class="option-row">
            <NInput
              :value="opt"
              size="small"
              placeholder="Option value"
              @update:value="(v: string) => updateOption(idx, v)"
            />
            <NButton quaternary size="small" type="error" @click="removeOption(idx)">
              <template #icon><NIcon :component="TrashOutline" /></template>
            </NButton>
          </div>
          <NButton size="small" dashed @click="addOption">
            <template #icon><NIcon :component="AddOutline" /></template>
            Add Option
          </NButton>
        </div>
      </NFormItem>
    </template>

    <!-- Interval type fields -->
    <template v-if="variable.type === 'interval'">
      <NFormItem label="Interval Options">
        <div class="custom-options">
          <div v-for="(opt, idx) in (variable.options || ['1m', '5m', '10m', '30m', '1h'])" :key="idx" class="option-row">
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
            Add Interval
          </NButton>
        </div>
      </NFormItem>
    </template>

    <!-- Datasource type fields -->
    <template v-if="variable.type === 'datasource'">
      <NFormItem label="Datasource Filter">
        <NSelect
          :value="variable.datasourceId"
          :options="datasourceOptions"
          placeholder="Base datasource (optional)"
          clearable
          @update:value="(v: number) => update('datasourceId', v)"
        />
      </NFormItem>
      <NFormItem label="Regex">
        <NInput
          :value="variable.regex"
          placeholder="Filter datasource names"
          @update:value="(v: string) => update('regex', v)"
        />
      </NFormItem>
    </template>

    <!-- Textbox type fields -->
    <template v-if="variable.type === 'textbox'">
      <NFormItem label="Default Value">
        <NInput
          :value="variable.defaultValue"
          placeholder="Default text"
          @update:value="(v: string) => update('defaultValue', v)"
        />
      </NFormItem>
    </template>

    <!-- Constant type fields -->
    <template v-if="variable.type === 'constant'">
      <NFormItem label="Value">
        <NInput
          :value="variable.defaultValue"
          placeholder="Constant value"
          @update:value="(v: string) => update('defaultValue', v)"
        />
      </NFormItem>
    </template>

    <!-- Adhoc type fields -->
    <template v-if="variable.type === 'adhoc'">
      <NFormItem label="Datasource">
        <NSelect
          :value="variable.datasourceId"
          :options="datasourceOptions"
          placeholder="Select datasource for adhoc filters"
          clearable
          @update:value="(v: number) => update('datasourceId', v)"
        />
      </NFormItem>
    </template>

    <!-- Common fields: multi / includeAll / allValue / defaultValue -->
    <NFormItem v-if="variable.type !== 'constant' && variable.type !== 'adhoc'" label="Default Value">
      <NInput
        :value="variable.defaultValue"
        placeholder="Default value"
        @update:value="(v: string) => update('defaultValue', v)"
      />
    </NFormItem>

    <NFormItem label="Multi Select">
      <NSwitch
        :value="variable.multi"
        @update:value="(v: boolean) => update('multi', v)"
      />
    </NFormItem>

    <NFormItem label="Include All">
      <NSwitch
        :value="variable.includeAll"
        @update:value="(v: boolean) => update('includeAll', v)"
      />
    </NFormItem>

    <NFormItem v-if="variable.includeAll" label="All Value">
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
