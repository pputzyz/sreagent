<script setup lang="ts">
/**
 * QueryRow — vmui-style single query row.
 * Q index + datasource selector + expression input + enable/disable + remove.
 */
import { computed } from 'vue'
import { NSelect, NIcon, NTooltip } from 'naive-ui'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
import PromQLEditor from './PromQLEditor.vue'
import LogsQLEditor from './LogsQLEditor.vue'
import type { DataSource } from '@/types'

const props = defineProps<{
  index: number
  dsId: number | null
  expression: string
  enabled: boolean
  datasources: DataSource[]
  activeTab: 'metrics' | 'logs'
  canRemove: boolean
}>()

const emit = defineEmits<{
  (e: 'update:dsId', v: number | null): void
  (e: 'update:expression', v: string): void
  (e: 'update:enabled', v: boolean): void
  (e: 'remove'): void
  (e: 'execute'): void
}>()

const isLogs = computed(() => props.activeTab === 'logs')

const metricDatasources = computed(() =>
  props.datasources.filter(d => d.supports_query && d.type !== 'victorialogs' && d.type !== 'elasticsearch')
)
const logDatasources = computed(() =>
  props.datasources.filter(d => d.type === 'victorialogs' || d.type === 'elasticsearch')
)

const datasourceOptions = computed(() => {
  const list = isLogs.value ? logDatasources.value : metricDatasources.value
  return list.map(d => ({
    label: `${d.name} (${typeBadge(d.type)})`,
    value: d.id,
  }))
})

function typeBadge(tp: string): string {
  const m: Record<string, string> = { prometheus: 'Prom', victoriametrics: 'VM', victorialogs: 'VL', zabbix: 'Zbx', elasticsearch: 'ES' }
  return m[tp] || tp
}
</script>

<template>
  <div class="query-row" :class="{ 'query-row--disabled': !enabled }">
    <div class="query-idx">Q{{ index + 1 }}</div>
    <NSelect
      :value="dsId"
      :options="datasourceOptions"
      :placeholder="isLogs ? t('explore.logDatasource') : t('explore.metricDatasource')"
      filterable
      size="small"
      class="query-ds-select"
      @update:value="(v: number | null) => emit('update:dsId', v)"
    />
    <div class="query-editor-wrap">
      <LogsQLEditor
        v-if="isLogs"
        :model-value="expression"
        :datasource-id="dsId"
        :placeholder="t('explore.logsqlPlaceholder')"
        @update:model-value="(v: string) => emit('update:expression', v)"
        @execute="emit('execute')"
      />
      <PromQLEditor
        v-else
        :model-value="expression"
        :datasource-id="dsId"
        :placeholder="t('explore.promqlPlaceholder')"
        @update:model-value="(v: string) => emit('update:expression', v)"
        @execute="emit('execute')"
      />
    </div>
    <NTooltip>
      <template #trigger>
        <button
          class="query-toggle-btn"
          :class="{ 'toggle-on': enabled }"
          @click="emit('update:enabled', !enabled)"
        >
          {{ enabled ? '●' : '○' }}
        </button>
      </template>
      {{ enabled ? t('explore.hideSeries') : t('explore.showSeries') }}
    </NTooltip>
    <NTooltip>
      <template #trigger>
        <button
          class="query-remove-btn"
          :disabled="!canRemove"
          @click="emit('remove')"
        >
          &times;
        </button>
      </template>
      {{ t('explore.removeQuery') }}
    </NTooltip>
  </div>
</template>

<style scoped>
.query-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
  transition: opacity 0.15s;
}
.query-row--disabled {
  opacity: 0.5;
}
.query-idx {
  width: 28px;
  text-align: center;
  font-size: 12px;
  color: var(--sre-text-tertiary, #94a3b8);
  font-family: var(--sre-font-mono, monospace);
  flex-shrink: 0;
}
.query-ds-select {
  width: 200px;
  flex-shrink: 0;
}
.query-editor-wrap {
  flex: 1;
  min-width: 0;
}
.query-toggle-btn {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--sre-text-tertiary, #94a3b8);
  cursor: pointer;
  font-size: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, color 0.15s;
  flex-shrink: 0;
}
.query-toggle-btn:hover {
  background: var(--sre-bg-hover, rgba(0,0,0,0.04));
  color: var(--sre-text-primary);
}
.query-toggle-btn.toggle-on {
  color: var(--sre-primary, #0D9488);
}
.query-remove-btn {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--sre-text-tertiary, #94a3b8);
  cursor: pointer;
  font-size: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, color 0.15s;
  flex-shrink: 0;
}
.query-remove-btn:hover:not(:disabled) {
  background: var(--sre-bg-hover, rgba(0,0,0,0.04));
  color: var(--sre-critical, #ef4444);
}
.query-remove-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}
</style>
