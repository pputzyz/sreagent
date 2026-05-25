<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  NDrawer,
  NDrawerContent,
  NButton,
  NCard,
  NIcon,
  NSpace,
  NScrollbar,
  NEmpty,
} from 'naive-ui'
import { AddOutline, TrashOutline } from '@vicons/ionicons5'
import type { VariableConfig } from '@/types/dashboard'
import type { DataSource } from '@/types'
import VariableEditorItem from './VariableEditorItem.vue'

const props = defineProps<{
  show: boolean
  variables: VariableConfig[]
  datasources: DataSource[]
}>()

const emit = defineEmits<{
  'update:variables': [value: VariableConfig[]]
  close: []
}>()

const selectedIndex = ref(0)

const selectedVariable = computed(() => {
  if (props.variables.length === 0) return null
  const idx = Math.min(selectedIndex.value, props.variables.length - 1)
  return props.variables[idx] || null
})

function addVariable() {
  const newVar: VariableConfig = {
    name: `var${props.variables.length + 1}`,
    label: '',
    type: 'query',
    query: '',
    regex: '',
    options: [],
    defaultValue: '',
    multi: false,
    includeAll: false,
    allValue: '$__all',
    refresh: 'onLoad',
    sort: 'disabled',
  }
  const updated = [...props.variables, newVar]
  emit('update:variables', updated)
  selectedIndex.value = updated.length - 1
}

function removeVariable(index: number) {
  const updated = [...props.variables]
  updated.splice(index, 1)
  emit('update:variables', updated)
  if (selectedIndex.value >= updated.length) {
    selectedIndex.value = Math.max(0, updated.length - 1)
  }
}

function updateVariable(index: number, value: VariableConfig) {
  const updated = [...props.variables]
  updated[index] = value
  emit('update:variables', updated)
}

function handleClose() {
  emit('close')
}

function getVarTypeLabel(type: string): string {
  const map: Record<string, string> = {
    query: 'Query',
    custom: 'Custom',
    interval: 'Interval',
    datasource: 'Datasource',
    textbox: 'Textbox',
    constant: 'Constant',
    adhoc: 'Adhoc',
  }
  return map[type] || type
}
</script>

<template>
  <NDrawer
    :show="show"
    :width="600"
    placement="right"
    @update:show="(v: boolean) => { if (!v) handleClose() }"
  >
    <NDrawerContent title="Manage Variables" closable>
      <div class="variable-editor">
        <!-- Variable list (left panel) -->
        <div class="var-list-section">
          <div class="var-list-header">
            <span class="var-list-title">Variables ({{ variables.length }})</span>
            <NButton size="small" type="primary" @click="addVariable">
              <template #icon><NIcon :component="AddOutline" /></template>
              Add
            </NButton>
          </div>
          <NScrollbar style="max-height: calc(100vh - 220px)">
            <div class="var-list">
              <NCard
                v-for="(v, idx) in variables"
                :key="idx"
                size="small"
                class="var-list-item"
                :class="{ 'var-list-item--active': idx === selectedIndex }"
                hoverable
                @click="selectedIndex = idx"
              >
                <div class="var-list-item-content">
                  <div class="var-list-item-info">
                    <span class="var-name">{{ v.name || '(unnamed)' }}</span>
                    <span class="var-type">{{ getVarTypeLabel(v.type) }}</span>
                  </div>
                  <NButton
                    quaternary
                    size="tiny"
                    type="error"
                    @click.stop="removeVariable(idx)"
                  >
                    <template #icon><NIcon :component="TrashOutline" /></template>
                  </NButton>
                </div>
              </NCard>
              <NEmpty v-if="variables.length === 0" description="No variables" style="padding: 24px 0" />
            </div>
          </NScrollbar>
        </div>

        <!-- Variable detail (right panel) -->
        <div class="var-detail-section">
          <NScrollbar style="max-height: calc(100vh - 180px)">
            <VariableEditorItem
              v-if="selectedVariable"
              :variable="selectedVariable"
              :datasources="datasources"
              @update:variable="(v: VariableConfig) => updateVariable(selectedIndex, v)"
            />
            <NEmpty v-else description="Select a variable to edit" style="padding: 60px 0" />
          </NScrollbar>
        </div>
      </div>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="handleClose">Close</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped>
.variable-editor {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
}

.var-list-section {
  border-bottom: 1px solid var(--n-border-color);
  padding-bottom: 12px;
}

.var-list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.var-list-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--n-text-color);
}

.var-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.var-list-item {
  cursor: pointer;
  transition: border-color 0.2s;
}

.var-list-item--active {
  border-color: var(--n-primary-color) !important;
}

.var-list-item-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.var-list-item-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.var-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--n-text-color);
}

.var-type {
  font-size: 11px;
  padding: 1px 6px;
  background: var(--n-color-info-suppl);
  color: var(--n-color-info);
  border-radius: 4px;
}

.var-detail-section {
  flex: 1;
  min-height: 0;
}
</style>
