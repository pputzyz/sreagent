<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NDrawer, NDrawerContent, NTabs, NTabPane, NButton, NSpace } from 'naive-ui'

const { t } = useI18n()
import type { PanelConfig, PanelOptions } from '@/types/dashboard'
import type { DataSource } from '@/types'
import PanelEditorGeneral from './PanelEditorGeneral.vue'
import PanelEditorQuery from './PanelEditorQuery.vue'
import PanelEditorVisualization from './PanelEditorVisualization.vue'
import PanelPreview from './PanelPreview.vue'

const props = defineProps<{
  show: boolean
  panel: PanelConfig
  datasources: DataSource[]
  timeRange: { start: number; end: number }
  variableOptions?: { label: string; value: string }[]
}>()

const emit = defineEmits<{
  (e: 'update:show', val: boolean): void
  (e: 'save', panel: PanelConfig): void
  (e: 'cancel'): void
}>()

// Deep clone for editing
const editPanel = ref<PanelConfig>(clonePanel(props.panel))

watch(() => props.panel, (newPanel) => {
  editPanel.value = clonePanel(newPanel)
}, { deep: true })

watch(() => props.show, (show) => {
  if (show) {
    editPanel.value = clonePanel(props.panel)
  }
})

function clonePanel(p: PanelConfig): PanelConfig {
  return JSON.parse(JSON.stringify(p))
}

function handleSave() {
  emit('save', clonePanel(editPanel.value))
  emit('update:show', false)
}

function handleCancel() {
  emit('cancel')
  emit('update:show', false)
}

function updateTitle(val: string) { editPanel.value.title = val }
function updateDescription(val: string) { editPanel.value.description = val }
function updateType(val: PanelConfig['type']) { editPanel.value.type = val }
function updateTransparent(val: boolean) { editPanel.value.transparent = val }
function updateRepeatByVariable(val: string | undefined) { editPanel.value.repeatByVariable = val }
function updateTargets(targets: PanelConfig['targets']) { editPanel.value.targets = targets }
function updateOptions(options: PanelOptions) { editPanel.value.options = options }

const varOpts = computed(() => props.variableOptions ?? [])
</script>

<template>
  <NDrawer
    :show="show"
    :width="800"
    placement="right"
    @update:show="(v: boolean) => { if (!v) handleCancel() }"
  >
    <NDrawerContent title="Panel Editor" closable>
      <NTabs type="line" animated>
        <NTabPane name="general" tab="General">
          <PanelEditorGeneral
            :title="editPanel.title"
            :description="editPanel.description ?? ''"
            :type="editPanel.type"
            :transparent="editPanel.transparent ?? false"
            :repeat-by-variable="editPanel.repeatByVariable"
            :variable-options="varOpts"
            @update:title="updateTitle"
            @update:description="updateDescription"
            @update:type="updateType"
            @update:transparent="updateTransparent"
            @update:repeat-by-variable="updateRepeatByVariable"
          />
        </NTabPane>

        <NTabPane name="queries" tab="Queries">
          <PanelEditorQuery
            :targets="editPanel.targets"
            :datasources="datasources"
            @update="updateTargets"
          />
        </NTabPane>

        <NTabPane name="visualization" tab="Visualization">
          <PanelEditorVisualization
            :type="editPanel.type"
            :options="editPanel.options"
            @update="updateOptions"
          />
        </NTabPane>
      </NTabs>

      <!-- Preview -->
      <PanelPreview :panel="editPanel" :time-range="timeRange" />

      <template #footer>
        <NSpace justify="end">
          <NButton @click="handleCancel">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" @click="handleSave">{{ t('common.save') }}</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>
