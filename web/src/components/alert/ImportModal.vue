<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  useMessage, NDrawer, NDrawerContent, NTabs, NTabPane, NSpace,
  NUpload, NUploadDragger, NIcon, NFormItem, NSelect, NButton,
  NRadioGroup, NRadioButton,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { alertRuleApi } from '@/api'
import type { DataSource } from '@/types'
import { CloudUploadOutline, CloudDownloadOutline } from '@vicons/ionicons5'

const props = defineProps<{
  show: boolean
  datasources: DataSource[]
  categories: string[]
}>()

const emit = defineEmits<{
  close: []
  imported: []
}>()

const message = useMessage()
const { t } = useI18n()

const importFile = ref<File | null>(null)

function onUploadChange(payload: { file: { file: File | null } }) {
  importFile.value = payload.file?.file || null
}
const importDatasourceId = ref<number | null>(null)
const importing = ref(false)
const exportFormat = ref('yaml')
const exportCategory = ref('')

const datasourceOptions = computed(() =>
  props.datasources.map(ds => ({ label: `${ds.name} (${ds.type})`, value: ds.id })),
)

const exportCategoryOptions = computed(() => [
  { label: t('alert.allCategories'), value: '' },
  ...props.categories.map(c => ({ label: c, value: c })),
])

async function handleImport() {
  if (!importFile.value) return
  importing.value = true
  try {
    const { data } = await alertRuleApi.importRules(importFile.value, importDatasourceId.value || undefined)
    const result = data.data
    message.success(t('alert.rulesImported', { success: result.success, total: result.total }))
    if (result.errors && result.errors.length > 0) {
      message.warning(result.errors.join('\n'))
    }
    importFile.value = null
    emit('imported')
  } catch (err: unknown) { message.error((err as Error).message) } finally { importing.value = false }
}

async function handleExport() {
  try {
    const params: Record<string, string> = { format: exportFormat.value }
    if (exportCategory.value) params.category = exportCategory.value
    const response = await alertRuleApi.exportRules(params)
    const blob = new Blob([response.data as BlobPart])
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `alert-rules.${exportFormat.value}`
    a.click()
    URL.revokeObjectURL(url)
  } catch (err: unknown) { message.error((err as Error).message) }
}
</script>

<template>
  <n-drawer :show="show" :width="480" placement="right" @update:show="(v: boolean) => { if (!v) emit('close') }">
    <n-drawer-content :title="t('alert.importExport')">
      <n-tabs type="line">
        <n-tab-pane name="import" :tab="t('alert.importFile')">
          <n-space vertical size="large">
            <n-upload
              :max="1"
              accept=".yaml,.yml,.json"
              :default-upload="false"
              @change="onUploadChange"
            >
              <n-upload-dragger>
                <div class="im-upload-drop">
                  <n-icon :component="CloudUploadOutline" :size="36" class="im-upload-icon" />
                  <div class="im-upload-hint">
                    {{ t('alert.dragOrClick') }}
                  </div>
                </div>
              </n-upload-dragger>
            </n-upload>
            <n-form-item :label="t('alert.dataSource')">
              <n-select
                v-model:value="importDatasourceId"
                :options="datasourceOptions"
                :placeholder="t('alert.selectDataSource')"
                clearable
              />
            </n-form-item>
            <n-button type="primary" block :loading="importing" :disabled="!importFile" @click="handleImport">
              {{ t('alert.importFile') }}
            </n-button>
          </n-space>
        </n-tab-pane>
        <n-tab-pane name="export" :tab="t('alert.exportRules')">
          <n-space vertical size="large">
            <n-form-item :label="t('alert.exportFormat')">
              <n-radio-group v-model:value="exportFormat">
                <n-radio-button value="yaml">YAML</n-radio-button>
                <n-radio-button value="json">JSON</n-radio-button>
              </n-radio-group>
            </n-form-item>
            <n-form-item :label="t('alert.category')">
              <n-select
                v-model:value="exportCategory"
                :options="exportCategoryOptions"
                :placeholder="t('alert.selectCategory')"
              />
            </n-form-item>
            <n-button type="primary" block @click="handleExport">
              <template #icon><n-icon :component="CloudDownloadOutline" /></template>
              {{ t('alert.exportRules') }}
            </n-button>
          </n-space>
        </n-tab-pane>
      </n-tabs>
    </n-drawer-content>
  </n-drawer>
</template>

<style scoped>
.im-upload-drop {
  padding: 20px;
  text-align: center;
}

.im-upload-icon {
  color: var(--sre-text-secondary);
}

.im-upload-hint {
  margin-top: 8px;
  color: var(--sre-text-secondary);
  font-size: 13px;
}
</style>
