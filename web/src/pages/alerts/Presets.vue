<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import {
  useMessage, useDialog,
  NButton, NIcon, NInput, NTag, NModal, NForm, NFormItem,
  NSelect, NSpace, NPagination, NSpin, NEmpty, NCard, NTooltip,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { presetRuleApi, datasourceApi } from '@/api'
import type { PresetRule, PresetRuleOverride } from '@/types/preset-rule'
import type { DataSource } from '@/types'
import { useCrudPage } from '@/composables/useCrudPage'
import type { CrudApiModule } from '@/composables/useCrudPage'
import PageHeader from '@/components/common/PageHeader.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import {
  SearchOutline, RefreshOutline, CloudUploadOutline,
  RocketOutline,
} from '@vicons/ionicons5'
import { getErrorMessage } from '@/utils/format'
import { severityLabel, severityType } from '@/utils/severity'

const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()

// ─── Category tabs ───
const activeCategory = ref('')
const categories = ref<string[]>([])

// ─── Search ───
const searchKeyword = ref('')

// ─── List via useCrudPage (no create/update for presets) ───
const presetCrudApi = {
  list: (params?: Record<string, unknown>) => presetRuleApi.list({
    ...params,
    category: activeCategory.value || undefined,
    search: searchKeyword.value.trim() || undefined,
  }),
  create: async () => { throw new Error('Not supported') },
  update: async () => { throw new Error('Not supported') },
  delete: presetRuleApi.delete,
} as unknown as CrudApiModule<PresetRule>

const {
  loading,
  items: presets,
  total,
  page,
  pageSize,
  fetchList,
  refresh,
} = useCrudPage<PresetRule>({
  api: presetCrudApi,
  defaultForm: () => ({} as any),
  i18nKeys: {},
  pageSize: 20,
})

// ─── Datasources for apply dialog ───
const datasources = ref<DataSource[]>([])
const datasourceOptions = computed(() =>
  datasources.value.map(ds => ({ label: `${ds.name} (${ds.type})`, value: ds.id })),
)

// ─── Apply dialog ───
const showApplyModal = ref(false)
const applyingPreset = ref<PresetRule | null>(null)
const applyForm = ref<PresetRuleOverride>({})
const applyLoading = ref(false)

function openApplyDialog(preset: PresetRule) {
  applyingPreset.value = preset
  applyForm.value = { severity: preset.severity }
  showApplyModal.value = true
}

async function handleApply() {
  if (!applyingPreset.value) return
  applyLoading.value = true
  try {
    await presetRuleApi.apply(applyingPreset.value.id, applyForm.value)
    message.success(t('preset.applySuccess'))
    showApplyModal.value = false
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    applyLoading.value = false
  }
}

// ─── Import YAML dialog ───
const showImportModal = ref(false)
const importYAML = ref('')
const importLoading = ref(false)

async function handleImportYAML() {
  if (!importYAML.value.trim()) {
    message.warning(t('preset.enterYaml'))
    return
  }
  importLoading.value = true
  try {
    const res = await presetRuleApi.importYAML(importYAML.value)
    const result = res.data.data
    message.success(t('preset.importResult', { imported: result.imported, skipped: result.skipped }))
    showImportModal.value = false
    importYAML.value = ''
    refresh()
    fetchCategories()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    importLoading.value = false
  }
}

// ─── Delete ───
function confirmDelete(preset: PresetRule) {
  dialog.warning({
    title: t('preset.confirmDeleteTitle'),
    content: t('preset.confirmDeleteMsg', { name: preset.display_name || preset.name }),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await presetRuleApi.delete(preset.id)
        message.success(t('preset.deleteSuccess'))
        fetchList()
      } catch (err: unknown) {
        message.error(getErrorMessage(err))
      }
    },
  })
}

// ─── Category fetch ───
async function fetchCategories() {
  try {
    const res = await presetRuleApi.categories()
    categories.value = res.data.data || []
  } catch { /* ignore */ }
}

// ─── Datasource fetch ───
async function fetchDatasources() {
  try {
    const res = await datasourceApi.list({ page: 1, page_size: 100 })
    datasources.value = res.data.data.list || []
  } catch { /* ignore */ }
}

// ─── Category change ───
function handleCategoryChange(cat: string) {
  activeCategory.value = cat
  refresh()
}

// ─── Search debounce ───
let searchTimer: ReturnType<typeof setTimeout> | null = null
watch(searchKeyword, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => refresh(), 300)
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})

onMounted(() => {
  fetchList()
  fetchCategories()
  fetchDatasources()
})
</script>

<template>
  <div class="presets-page">
    <PageHeader :title="t('preset.title')" :subtitle="t('preset.subtitle')">
      <template #actions>
        <n-button size="small" secondary @click="showImportModal = true">
          <template #icon><n-icon :component="CloudUploadOutline" /></template>
          {{ t('preset.importYaml') }}
        </n-button>
        <n-button size="small" secondary @click="refresh">
          <template #icon><n-icon :component="RefreshOutline" /></template>
          {{ t('common.refresh') }}
        </n-button>
      </template>
    </PageHeader>

    <div class="presets-layout">
      <!-- Sidebar: categories -->
      <aside class="cat-aside">
        <div class="sre-label-eyebrow cat-eyebrow">{{ t('preset.category') }}</div>
        <button
          type="button"
          class="cat-item"
          :class="{ active: activeCategory === '' }"
          @click="handleCategoryChange('')"
        >
          <span class="cat-name">{{ t('common.all') }}</span>
          <span class="cat-count tnum">{{ total }}</span>
        </button>
        <button
          v-for="cat in categories"
          :key="cat"
          type="button"
          class="cat-item"
          :class="{ active: activeCategory === cat }"
          @click="handleCategoryChange(cat)"
        >
          <span class="cat-name">{{ cat }}</span>
        </button>
      </aside>

      <!-- Main column -->
      <section class="presets-main">
        <!-- Toolbar -->
        <div class="toolbar">
          <n-input
            v-model:value="searchKeyword"
            size="small"
            :placeholder="t('preset.searchPlaceholder')"
            clearable
            class="toolbar-search"
          >
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
        </div>

        <!-- Loading -->
        <LoadingSkeleton v-if="loading && presets.length === 0" :rows="6" variant="row" />

        <!-- Empty -->
        <n-empty
          v-else-if="!loading && presets.length === 0"
          :description="t('preset.noData')"
          class="empty-state"
        >
          <template #extra>
            <n-button size="small" type="primary" @click="showImportModal = true">
              {{ t('preset.importYaml') }}
            </n-button>
          </template>
        </n-empty>

        <!-- Rule list -->
        <div v-else class="preset-list sre-stagger">
          <div
            v-for="preset in presets"
            :key="preset.id"
            class="sre-row-card preset-row"
            :data-severity="preset.severity === 'critical' || preset.severity === 'p0' ? 'critical' : preset.severity === 'warning' || preset.severity === 'p2' ? 'warning' : 'info'"
          >
            <div class="preset-main">
              <div class="preset-title">
                <span class="preset-name">{{ preset.display_name || preset.name }}</span>
                <n-tag v-if="preset.is_builtin" size="small" :bordered="false" type="success">{{ t('preset.builtin') }}</n-tag>
                <n-tag v-if="preset.source" size="small" :bordered="false">{{ preset.source }}</n-tag>
              </div>
              <div class="preset-desc" v-if="preset.description">{{ preset.description }}</div>
              <div class="preset-expr">{{ preset.expression }}</div>
              <div class="preset-meta">
                <n-tag :type="severityType(preset.severity)" size="small" :bordered="false">
                  {{ severityLabel(preset.severity) }}
                </n-tag>
                <span v-if="preset.category" class="preset-meta-item">{{ preset.category }}</span>
                <span v-if="preset.sub_category" class="preset-meta-item">/ {{ preset.sub_category }}</span>
                <span v-if="preset.component" class="preset-meta-item">| {{ preset.component }}</span>
                <span v-if="preset.for_duration" class="preset-meta-item tnum">{{ t('preset.forDuration', { duration: preset.for_duration }) }}</span>
                <span v-if="preset.usage_count > 0" class="preset-meta-item tnum">{{ t('preset.usageCount', { count: preset.usage_count }) }}</span>
              </div>
            </div>
            <div class="preset-actions">
              <n-tooltip trigger="hover">
                <template #trigger>
                  <n-button size="small" type="primary" @click="openApplyDialog(preset)">
                    <template #icon><n-icon :component="RocketOutline" /></template>
                    {{ t('preset.apply') }}
                  </n-button>
                </template>
                {{ t('preset.applyTooltip') }}
              </n-tooltip>
              <n-button
                v-if="!preset.is_builtin"
                size="small"
                quaternary
                type="error"
                @click="confirmDelete(preset)"
              >
                {{ t('common.delete') }}
              </n-button>
            </div>
          </div>
        </div>

        <!-- Pagination -->
        <div v-if="presets.length > 0" class="pagination-wrap">
          <n-pagination
            v-model:page="page"
            :page-size="pageSize"
            :item-count="total"
            :page-slot="7"
            @update:page="fetchList"
          />
        </div>
      </section>
    </div>

    <!-- Apply Dialog -->
    <n-modal
      v-model:show="showApplyModal"
      preset="card"
      :title="t('preset.applyModalTitle')"
      style="width: 520px"
      :bordered="false"
      :segmented="{ content: true, footer: true }"
    >
      <n-form v-if="applyingPreset" label-placement="left" label-width="80">
        <n-form-item :label="t('preset.ruleName')">
          <span class="form-readonly">{{ applyingPreset.display_name || applyingPreset.name }}</span>
        </n-form-item>
        <n-form-item :label="t('preset.expression')">
          <n-input
            :value="applyingPreset.expression"
            type="textarea"
            :rows="3"
            readonly
            class="mono-input"
          />
        </n-form-item>
        <n-form-item :label="t('preset.datasource')" required>
          <n-select
            v-model:value="applyForm.datasource_id"
            :options="datasourceOptions"
            :placeholder="t('preset.selectDatasource')"
            filterable
          />
        </n-form-item>
        <n-form-item :label="t('preset.severity')">
          <n-select
            v-model:value="applyForm.severity"
            :options="[
              { label: t('severity.critical'), value: 'critical' },
              { label: t('severity.warning'), value: 'warning' },
              { label: t('severity.info'), value: 'info' },
              { label: 'P0', value: 'p0' },
              { label: 'P1', value: 'p1' },
              { label: 'P2', value: 'p2' },
              { label: 'P3', value: 'p3' },
            ]"
            :placeholder="t('preset.severityPlaceholder')"
            clearable
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showApplyModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="applyLoading" :disabled="!applyForm.datasource_id" @click="handleApply">
            {{ t('preset.confirmApply') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Import YAML Dialog -->
    <n-modal
      v-model:show="showImportModal"
      preset="card"
      :title="t('preset.importModalTitle')"
      style="width: 640px"
      :bordered="false"
      :segmented="{ content: true, footer: true }"
    >
      <div class="import-hint">
        {{ t('preset.importHint') }}
      </div>
      <n-input
        v-model:value="importYAML"
        type="textarea"
        :rows="16"
        :placeholder="t('preset.importPlaceholder')"
        class="mono-input"
      />
      <template #footer>
        <n-space justify="end">
          <n-button @click="showImportModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="importLoading" @click="handleImportYAML">
            {{ t('preset.import') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.presets-page {
  max-width: 1400px;
  font-family: var(--sre-font-sans);
}

.presets-layout {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 24px;
  margin-top: 16px;
  align-items: start;
}

/* Sidebar */
.cat-aside {
  background: var(--sre-bg-card);
  border-right: var(--sre-hairline);
  border-radius: 8px 0 0 8px;
  padding: 16px 0;
  position: sticky;
  top: 16px;
}
.cat-eyebrow {
  padding: 0 16px 8px;
  color: var(--sre-text-tertiary);
}
.cat-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 8px 16px;
  font-size: 13px;
  color: var(--sre-text-secondary);
  cursor: pointer;
  position: relative;
  transition: background 120ms ease, color 120ms ease;
  border: none;
  border-left: 2px solid transparent;
  border-radius: 0;
  background: none;
  font-family: inherit;
  text-align: left;
}
.cat-item:hover {
  color: var(--sre-text-primary);
  background: var(--sre-bg-hover, rgba(255,255,255,0.03));
}
.cat-item.active {
  color: var(--sre-primary);
  background: var(--sre-primary-soft);
  border-left-color: var(--sre-primary);
  font-weight: 500;
}
.cat-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cat-count {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  font-family: var(--sre-font-mono, monospace);
}
.cat-item.active .cat-count {
  color: var(--sre-primary);
}

/* Main */
.presets-main {
  min-width: 0;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 0;
  margin-bottom: 4px;
}
.toolbar-search { width: 280px; }

/* List */
.preset-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.preset-row {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 14px 16px 14px 20px;
}

.preset-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.preset-title {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.preset-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
}

.preset-desc {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  line-height: 1.5;
}

.preset-expr {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  color: var(--sre-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 600px;
}

.preset-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--sre-text-tertiary);
  flex-wrap: wrap;
  row-gap: 4px;
  margin-top: 2px;
}
.preset-meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.preset-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  padding-top: 2px;
}

/* Pagination */
.pagination-wrap {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}

/* Form helpers */
.form-readonly {
  font-size: 14px;
  font-weight: 500;
  color: var(--sre-text-primary);
}

.mono-input :deep(textarea),
.mono-input :deep(input) {
  font-family: var(--sre-font-mono, monospace) !important;
  font-size: 12px !important;
}

.import-hint {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin-bottom: 12px;
}

.empty-state {
  margin-top: 80px;
}
</style>
