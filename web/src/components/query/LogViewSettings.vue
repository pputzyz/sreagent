<script setup lang="ts">
import { computed } from 'vue'
import { NPopover, NSwitch, NInputNumber, NIcon, NButton, NDivider } from 'naive-ui'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
import { SettingsOutline } from '@vicons/ionicons5'

export interface LogViewOptions {
  lineBreak: boolean
  showTime: boolean
  showLabels: boolean
  showLineNum: boolean
  jsonExpandLevel: number
}

const props = defineProps<{
  options: LogViewOptions
}>()

const emit = defineEmits<{
  'update:options': [value: LogViewOptions]
}>()

function update(patch: Partial<LogViewOptions>) {
  emit('update:options', { ...props.options, ...patch })
}

const lineBreak = computed({
  get: () => props.options.lineBreak,
  set: (v) => update({ lineBreak: v }),
})

const showTime = computed({
  get: () => props.options.showTime,
  set: (v) => update({ showTime: v }),
})

const showLabels = computed({
  get: () => props.options.showLabels,
  set: (v) => update({ showLabels: v }),
})

const showLineNum = computed({
  get: () => props.options.showLineNum,
  set: (v) => update({ showLineNum: v }),
})

const jsonExpandLevel = computed({
  get: () => props.options.jsonExpandLevel,
  set: (v) => update({ jsonExpandLevel: v ?? 1 }),
})
</script>

<template>
  <n-popover trigger="click" placement="bottom-end" :show-arrow="false">
    <template #trigger>
      <n-button quaternary size="small" style="padding: 0 6px;">
        <template #icon>
          <n-icon :size="18">
            <SettingsOutline />
          </n-icon>
        </template>
      </n-button>
    </template>

    <div class="log-view-settings">
      <div class="setting-row">
        <span class="setting-label">{{ t('explore.lineBreak') }}</span>
        <n-switch v-model:value="lineBreak" size="small" />
      </div>

      <div class="setting-row">
        <span class="setting-label">{{ t('explore.showTime') }}</span>
        <n-switch v-model:value="showTime" size="small" />
      </div>

      <div class="setting-row">
        <span class="setting-label">{{ t('explore.showLabels') }}</span>
        <n-switch v-model:value="showLabels" size="small" />
      </div>

      <n-divider style="margin: 4px 0;" />

      <div class="setting-row">
        <span class="setting-label">{{ t('explore.showLineNumbers') }}</span>
        <n-switch v-model:value="showLineNum" size="small" />
      </div>

      <n-divider style="margin: 4px 0;" />

      <div class="setting-row">
        <span class="setting-label">{{ t('explore.jsonExpandLevel') }}</span>
        <n-input-number
          v-model:value="jsonExpandLevel"
          :min="1"
          :max="5"
          size="small"
          style="width: 80px;"
        />
      </div>
    </div>
  </n-popover>
</template>

<style scoped>
.log-view-settings {
  min-width: 220px;
  padding: 4px 0;
}

.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 0;
}

.setting-label {
  font-size: 13px;
  color: var(--sre-text-secondary);
}
</style>
