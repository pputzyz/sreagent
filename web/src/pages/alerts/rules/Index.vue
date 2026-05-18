<script setup lang="ts">
import { h, ref, computed, onMounted, watch } from 'vue'
import { useMessage, useDialog, NButton, NIcon, NDropdown, NInput, NSelect, NPagination, NSwitch, NModal, NForm, NFormItem, NSpace, NSpin, NAlert, NTag } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { alertRuleApi, datasourceApi, aiRuleApi } from '@/api'
import type { AlertRule, DataSource } from '@/types'
import type { RuleGenerateResult } from '@/types/preset-rule'
import { usePaginatedList, useAIModule } from '@/composables'
import { DynamicScroller, DynamicScrollerItem } from 'vue-virtual-scroller'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import RuleFormModal from '@/components/alert/RuleFormModal.vue'
import ImportModal from '@/components/alert/ImportModal.vue'
import BatchOperations from '@/components/alert/BatchOperations.vue'
import {
  AddOutline,
  CloudUploadOutline,
  SearchOutline,
  EllipsisHorizontalOutline,
  CreateOutline,
  CopyOutline,
  TrashOutline,
  PowerOutline,
  DocumentTextOutline,
  SparklesOutline,
} from '@vicons/ionicons5'

const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()
const router = useRouter()

// ─── List state ───
const datasources = ref<DataSource[]>([])
const isFirstLoad = ref(true)

const {
  loading,
  items: rules,
  total,
  page,
  pageSize,
  fetchList,
  refresh,
} = usePaginatedList<AlertRule>({
  apiFn: alertRuleApi.list,
  pageSize: 50,
  extraParams: () => {
    const params: Record<string, unknown> = {}
    if (activeCategory.value) params.category = activeCategory.value
    return params
  },
  onError: (err: unknown) => {
    message.error((err as Error).message)
  },
})

// ─── Filters ───
const searchKeyword = ref('')
const filterDatasource = ref<number | null>(null)
const filterSeverity = ref<string | null>(null)
const filterStatus = ref<string | null>(null)

// ─── Batch selection ───
const selectedKeys = ref<number[]>([])
const batchLoading = ref(false)

// ─── Category sidebar ───
const activeCategory = ref('')
const categories = ref<string[]>([])
const categoryCounts = ref<Record<string, number>>({})

// ─── Modals ───
const showFormModal = ref(false)
const currentRule = ref<AlertRule | null>(null)
const duplicateFrom = ref<AlertRule | null>(null)
const showImportModal = ref(false)

// ─── AI Rule Generation ───
const { isEnabled: isAIModuleEnabled, loadModules } = useAIModule()
const showAIModal = ref(false)
const aiDescription = ref('')
const aiDatasourceId = ref<number | null>(null)
const aiGenerating = ref(false)
const aiResult = ref<RuleGenerateResult | null>(null)
const aiError = ref('')

function openAIGenerate() {
  aiDescription.value = ''
  aiDatasourceId.value = null
  aiResult.value = null
  aiError.value = ''
  showAIModal.value = true
}

async function handleAIGenerate() {
  if (!aiDescription.value.trim()) return
  aiGenerating.value = true
  aiResult.value = null
  aiError.value = ''
  try {
    const { data } = await aiRuleApi.generate({
      description: aiDescription.value,
      datasource_id: aiDatasourceId.value ?? undefined,
      rule_type: 'alert',
    })
    aiResult.value = data.data
  } catch (err: unknown) {
    aiError.value = (err as Error).message || 'AI 生成失败'
  } finally {
    aiGenerating.value = false
  }
}

async function handleAIConfirmCreate() {
  if (!aiResult.value) return
  try {
    await alertRuleApi.create({
      name: aiResult.value.name,
      expression: aiResult.value.expression || '',
      for_duration: aiResult.value.for_duration || '0s',
      severity: (aiResult.value.severity as AlertRule['severity']) || 'warning',
      labels: aiResult.value.labels || {},
      annotations: aiResult.value.annotations || {},
      datasource_id: aiDatasourceId.value,
    })
    message.success(t('common.createSuccess'))
    showAIModal.value = false
    fetchList()
  } catch (err: unknown) {
    message.error((err as Error).message)
  }
}

// ─── Computed ───
const severityFilterOptions = computed(() => [
  { label: t('alert.critical'), value: 'critical' },
  { label: t('alert.warning'), value: 'warning' },
  { label: t('alert.info'), value: 'info' },
])

const statusFilterOptions = computed(() => [
  { label: t('common.enabled'), value: 'enabled' },
  { label: t('common.disabled'), value: 'disabled' },
])

const datasourceOptions = computed(() =>
  datasources.value.map(ds => ({ label: `${ds.name} (${ds.type})`, value: ds.id })),
)

const filteredRules = computed(() => {
  let arr = rules.value
  if (searchKeyword.value.trim()) {
    const kw = searchKeyword.value.trim().toLowerCase()
    arr = arr.filter(r =>
      r.name?.toLowerCase().includes(kw) ||
      r.display_name?.toLowerCase().includes(kw) ||
      r.expression?.toLowerCase().includes(kw),
    )
  }
  if (filterDatasource.value != null) {
    arr = arr.filter(r => r.datasource_id === filterDatasource.value)
  }
  if (filterSeverity.value) {
    arr = arr.filter(r => r.severity === filterSeverity.value)
  }
  if (filterStatus.value) {
    arr = arr.filter(r => r.status === filterStatus.value)
  }
  return arr
})

const allCount = computed(() => total.value)

// ─── Helpers ───
function severityLabel(sev: string) {
  const map: Record<string, string> = {
    critical: t('alert.critical'),
    warning: t('alert.warning'),
    info: t('alert.info'),
    p0: t('alert.p0'), p1: t('alert.p1'), p2: t('alert.p2'), p3: t('alert.p3'), p4: t('alert.p4'),
  }
  return map[sev] || sev
}

function severitySlot(sev: string): 'critical' | 'warning' | 'info' | 'success' {
  if (sev === 'critical' || sev === 'p0' || sev === 'p1') return 'critical'
  if (sev === 'warning' || sev === 'p2') return 'warning'
  if (sev === 'info' || sev === 'p4') return 'info'
  return 'info'
}

// ─── Data fetching ───
// fetchRules is now handled by usePaginatedList.fetchList
// isFirstLoad tracking
watch(loading, (isLoading) => {
  if (!isLoading && isFirstLoad.value) {
    setTimeout(() => { isFirstLoad.value = false }, 800)
  }
})

async function fetchCategories() {
  try {
    const { data } = await alertRuleApi.listCategories()
    categories.value = data.data || []
  } catch { /* ignore */ }
}

function handleCategoryChange(cat: string) {
  activeCategory.value = cat
  refresh()
}

async function fetchDatasources() {
  try {
    const { data } = await datasourceApi.list({ page: 1, page_size: 100 })
    datasources.value = data.data.list || []
  } catch { /* ignore */ }
}

// ─── Batch operations ───
async function handleBatchEnable() {
  if (selectedKeys.value.length === 0) return
  batchLoading.value = true
  try {
    await alertRuleApi.batchEnable(selectedKeys.value)
    message.success(t('alert.batchEnabled', { count: selectedKeys.value.length }))
    selectedKeys.value = []
    fetchList()
  } catch (err: unknown) { message.error((err as Error).message) } finally { batchLoading.value = false }
}

async function handleBatchDisable() {
  if (selectedKeys.value.length === 0) return
  batchLoading.value = true
  try {
    await alertRuleApi.batchDisable(selectedKeys.value)
    message.success(t('alert.batchDisabled', { count: selectedKeys.value.length }))
    selectedKeys.value = []
    fetchList()
  } catch (err: unknown) { message.error((err as Error).message) } finally { batchLoading.value = false }
}

async function handleBatchDelete() {
  if (selectedKeys.value.length === 0) return
  batchLoading.value = true
  try {
    await alertRuleApi.batchDelete(selectedKeys.value)
    message.success(t('alert.batchDeleted', { count: selectedKeys.value.length }))
    selectedKeys.value = []
    fetchList()
  } catch (err: unknown) { message.error((err as Error).message) } finally { batchLoading.value = false }
}

function toggleSelect(id: number, checked: boolean) {
  if (checked) {
    if (!selectedKeys.value.includes(id)) selectedKeys.value = [...selectedKeys.value, id]
  } else {
    selectedKeys.value = selectedKeys.value.filter(k => k !== id)
  }
}

function isSelected(id: number) {
  return selectedKeys.value.includes(id)
}

const allSelected = computed(() =>
  filteredRules.value.length > 0 && filteredRules.value.every(r => selectedKeys.value.includes(r.id)),
)

function toggleSelectAll(checked: boolean) {
  if (checked) {
    selectedKeys.value = filteredRules.value.map(r => r.id)
  } else {
    selectedKeys.value = []
  }
}

// ─── Modal handlers ───
function openCreate() {
  currentRule.value = null
  duplicateFrom.value = null
  showFormModal.value = true
}

function openEdit(rule: AlertRule) {
  currentRule.value = rule
  duplicateFrom.value = null
  showFormModal.value = true
}

function onFormSaved() {
  showFormModal.value = false
  fetchList()
}

function onImportDone() {
  showImportModal.value = false
  fetchList()
  fetchCategories()
}

// ─── Row actions ───
async function toggleEnabled(rule: AlertRule) {
  const newStatus = rule.status === 'enabled' ? 'disabled' : 'enabled'
  try {
    await alertRuleApi.toggleStatus(rule.id, newStatus)
    message.success(newStatus === 'enabled' ? t('alert.ruleEnabled') : t('alert.ruleDisabled'))
    fetchList()
  } catch (err: unknown) { message.error((err as Error).message) }
}

async function handleDelete(id: number) {
  try {
    await alertRuleApi.delete(id)
    message.success(t('alert.ruleDeleted'))
    fetchList()
  } catch (err: unknown) { message.error((err as Error).message) }
}

function rowActions(rule: AlertRule) {
  return [
    { label: t('common.edit'), key: 'edit', icon: () => h(NIcon, { component: CreateOutline }) },
    { label: rule.status === 'enabled' ? t('common.disabled') : t('common.enabled'), key: 'toggle', icon: () => h(NIcon, { component: PowerOutline }) },
    { label: t('common.duplicate') || 'Duplicate', key: 'duplicate', icon: () => h(NIcon, { component: CopyOutline }) },
    { type: 'divider' as const, key: 'd1' },
    { label: t('common.delete'), key: 'delete', icon: () => h(NIcon, { component: TrashOutline }) },
  ]
}

function onRowAction(key: string, rule: AlertRule) {
  if (key === 'edit') openEdit(rule)
  else if (key === 'toggle') toggleEnabled(rule)
  else if (key === 'duplicate') {
    currentRule.value = null
    duplicateFrom.value = rule
    showFormModal.value = true
  } else if (key === 'delete') {
    dialog.warning({
      title: t('common.confirmDelete'),
      content: t('alert.deleteRuleConfirm'),
      positiveText: t('common.confirm'),
      negativeText: t('common.cancel'),
      onPositiveClick: () => handleDelete(rule.id),
    })
  }
}

function goDetail(rule: AlertRule) {
  router.push(`/alert/rules/${rule.id}`).catch(() => { /* no-op */ })
}

onMounted(() => {
  fetchList()
  fetchDatasources()
  fetchCategories()
  loadModules()
})
</script>

<template>
  <div class="rules-page">
    <PageHeader :title="t('alert.rules')" :subtitle="t('alert.rulesSubtitle')">
      <template #actions>
        <n-button v-if="isAIModuleEnabled('rule_gen')" size="small" secondary @click="openAIGenerate">
          <template #icon><n-icon :component="SparklesOutline" /></template>
          {{ t('alert.aiGenerate') || 'AI Generate' }}
        </n-button>
        <n-button size="small" secondary @click="showImportModal = true">
          <template #icon><n-icon :component="CloudUploadOutline" /></template>
          {{ t('alert.importExport') }}
        </n-button>
        <n-button size="small" type="primary" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('alert.createRule') }}
        </n-button>
      </template>
    </PageHeader>

    <div class="rules-layout">
      <!-- Sidebar: categories -->
      <aside class="cat-aside">
        <div class="sre-label-eyebrow cat-eyebrow">{{ t('alert.category') }}</div>
        <a
          class="cat-item"
          :class="{ active: activeCategory === '' }"
          @click="handleCategoryChange('')"
        >
          <span class="cat-name">{{ t('alert.allCategories') }}</span>
          <span class="cat-count tnum">{{ allCount }}</span>
        </a>
        <a
          v-for="cat in categories"
          :key="cat"
          class="cat-item"
          :class="{ active: activeCategory === cat }"
          @click="handleCategoryChange(cat)"
        >
          <span class="cat-name">{{ cat }}</span>
          <span class="cat-count tnum">{{ categoryCounts[cat] ?? '' }}</span>
        </a>
      </aside>

      <!-- Main column -->
      <section class="rules-main">
        <!-- Toolbar -->
        <div class="toolbar">
          <n-input
            v-model:value="searchKeyword"
            size="small"
            :placeholder="t('common.search')"
            clearable
            class="toolbar-search"
          >
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <n-select
            v-model:value="filterDatasource"
            size="small"
            :options="datasourceOptions"
            :placeholder="t('alert.dataSource')"
            clearable
            class="toolbar-select"
          />
          <n-select
            v-model:value="filterSeverity"
            size="small"
            :options="severityFilterOptions"
            :placeholder="t('alert.severity')"
            clearable
            class="toolbar-select"
          />
          <n-select
            v-model:value="filterStatus"
            size="small"
            :options="statusFilterOptions"
            :placeholder="t('common.status')"
            clearable
            class="toolbar-select"
          />
          <div class="toolbar-spacer"></div>
          <label class="select-all-label">
            <input
              type="checkbox"
              :checked="allSelected"
              @change="toggleSelectAll(($event.target as HTMLInputElement).checked)"
            />
            <span>{{ t('common.selectAll') || 'Select All' }}</span>
          </label>
        </div>

        <!-- Selection bar -->
        <BatchOperations
          v-if="selectedKeys.length > 0"
          :selected-count="selectedKeys.length"
          :loading="batchLoading"
          @batch-enable="handleBatchEnable"
          @batch-disable="handleBatchDisable"
          @batch-delete="handleBatchDelete"
          @clear-selection="selectedKeys = []"
        />

        <!-- Loading skeleton -->
        <LoadingSkeleton v-if="loading && filteredRules.length === 0" :rows="6" variant="row" />

        <!-- Empty state -->
        <EmptyState
          v-else-if="!loading && filteredRules.length === 0"
          :icon="DocumentTextOutline"
          :title="t('alert.noRules') || 'No alert rules'"
          :description="t('alert.rulesSubtitle') || 'Create your first rule to start monitoring'"
          :primary-text="t('alert.createFirstRule')"
          :secondary-text="t('alert.importFile')"
          @primary="openCreate"
          @secondary="showImportModal = true"
        />

        <!-- Rule list -->
        <DynamicScroller
          v-else
          class="rule-list"
          :class="{ 'sre-stagger': isFirstLoad }"
          :items="filteredRules"
          key-field="id"
          :min-item-size="72"
        >
          <template #default="{ item: rule }">
            <DynamicScrollerItem
              :item="rule"
              :active="true"
              :size-dependencies="[rule.expression, rule.category, rule.for_duration]"
            >
              <div
                class="sre-row-card rule-row"
                :data-severity="severitySlot(rule.severity)"
                :data-dim="rule.status !== 'enabled' || undefined"
                @click="goDetail(rule)"
              >
            <input
              type="checkbox"
              class="rc-check"
              :checked="isSelected(rule.id)"
              @click.stop
              @change="toggleSelect(rule.id, ($event.target as HTMLInputElement).checked)"
            />
            <div class="rc-main">
              <div class="rc-title">
                <span class="rc-name">{{ rule.display_name || rule.name }}</span>
                <span class="rc-id tnum">#{{ rule.id }}</span>
              </div>
              <div class="rc-expr">{{ rule.expression }}</div>
              <div class="rc-meta">
                <span class="rc-meta-item">
                  <span class="sre-dot" :data-severity="severitySlot(rule.severity)"></span>
                  {{ severityLabel(rule.severity) }}
                </span>
                <span class="sre-meta-divider"></span>
                <span class="rc-meta-item">{{ rule.datasource?.name || '—' }}</span>
                <template v-if="rule.category">
                  <span class="sre-meta-divider"></span>
                  <span class="rc-meta-item">{{ rule.category }}</span>
                </template>
                <template v-if="rule.for_duration">
                  <span class="sre-meta-divider"></span>
                  <span class="rc-meta-item tnum">{{ t('alert.forPrefix') }} {{ rule.for_duration }}</span>
                </template>
              </div>
            </div>
            <div class="rc-toggle" @click.stop>
              <n-switch :value="rule.status === 'enabled'" size="small" @update:value="toggleEnabled(rule)" />
            </div>
            <div class="rc-actions" @click.stop>
              <n-dropdown :options="rowActions(rule)" trigger="click" @select="(k: string) => onRowAction(k, rule)">
                <n-button quaternary circle size="small">
                  <template #icon><n-icon :component="EllipsisHorizontalOutline" /></template>
                </n-button>
              </n-dropdown>
            </div>
              </div>
            </DynamicScrollerItem>
          </template>
        </DynamicScroller>

        <!-- Pagination -->
        <div v-if="filteredRules.length > 0" class="pagination-wrap">
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

    <!-- Create/Edit Modal -->
    <RuleFormModal
      :show="showFormModal"
      :rule="currentRule"
      :duplicate-from="duplicateFrom"
      :datasources="datasources"
      @close="showFormModal = false"
      @saved="onFormSaved"
    />

    <!-- Import/Export Drawer -->
    <ImportModal
      :show="showImportModal"
      :datasources="datasources"
      :categories="categories"
      @close="showImportModal = false"
      @imported="onImportDone"
    />

    <!-- AI Generate Modal -->
    <NModal
      v-model:show="showAIModal"
      :title="t('alert.aiGenerate') || 'AI Generate Rule'"
      preset="card"
      class="ai-gen-modal"
      :mask-closable="false"
      :bordered="false"
      style="max-width: 680px"
    >
      <div class="ai-gen-form">
        <div class="ai-gen-field">
          <label class="ai-gen-label">{{ t('alert.aiDescription') || 'Describe the rule you want' }}</label>
          <NInput
            v-model:value="aiDescription"
            type="textarea"
            :rows="3"
            :placeholder="t('alert.aiDescriptionPlaceholder') || 'e.g. Alert when CPU usage exceeds 90% for 5 minutes on production servers'"
          />
        </div>
        <div class="ai-gen-field">
          <label class="ai-gen-label">{{ t('alert.dataSource') }} ({{ t('common.optional') || 'Optional' }})</label>
          <NSelect
            v-model:value="aiDatasourceId"
            :options="datasourceOptions"
            :placeholder="t('alert.selectDatasource') || 'Select datasource'"
            clearable
          />
        </div>
        <NButton type="primary" :loading="aiGenerating" :disabled="!aiDescription.trim()" @click="handleAIGenerate">
          <template #icon><NIcon :component="SparklesOutline" /></template>
          {{ t('alert.aiGenerateBtn') || 'Generate' }}
        </NButton>
      </div>

      <!-- AI Error -->
      <NAlert v-if="aiError" type="error" style="margin-top: 16px">
        {{ aiError }}
      </NAlert>

      <!-- AI Result Preview -->
      <div v-if="aiResult" class="ai-gen-preview">
        <div class="ai-gen-preview-header">
          <span class="ai-gen-preview-title">{{ aiResult.name }}</span>
          <NTag v-if="aiResult.severity" :type="aiResult.severity === 'critical' ? 'error' : aiResult.severity === 'warning' ? 'warning' : 'info'" size="small">
            {{ aiResult.severity }}
          </NTag>
          <span class="ai-gen-confidence">{{ Math.round(aiResult.confidence * 100) }}%</span>
        </div>
        <div v-if="aiResult.expression" class="ai-gen-expr">{{ aiResult.expression }}</div>
        <div v-if="aiResult.description" class="ai-gen-desc">{{ aiResult.description }}</div>
        <div v-if="aiResult.for_duration" class="ai-gen-meta">
          <span class="ai-gen-meta-label">Duration:</span> {{ aiResult.for_duration }}
        </div>
        <div v-if="aiResult.labels && Object.keys(aiResult.labels).length > 0" class="ai-gen-meta">
          <span class="ai-gen-meta-label">Labels:</span>
          <NTag v-for="(v, k) in aiResult.labels" :key="k" size="small" style="margin-right: 4px">{{ k }}={{ v }}</NTag>
        </div>
        <div v-if="aiResult.annotations?.summary" class="ai-gen-meta">
          <span class="ai-gen-meta-label">Summary:</span> {{ aiResult.annotations.summary }}
        </div>
        <NAlert v-if="aiResult.warnings?.length" type="warning" style="margin-top: 12px">
          <div v-for="w in aiResult.warnings" :key="w">{{ w }}</div>
        </NAlert>
        <NSpace justify="end" style="margin-top: 16px">
          <NButton @click="handleAIGenerate">{{ t('alert.aiRegenerate') || 'Regenerate' }}</NButton>
          <NButton type="primary" @click="handleAIConfirmCreate">{{ t('alert.aiConfirmCreate') || 'Confirm & Create' }}</NButton>
        </NSpace>
      </div>
    </NModal>
  </div>
</template>

<style scoped>
.rules-page {
  max-width: 1400px;
  font-family: var(--sre-font-sans);
}

.rules-layout {
  display: grid;
  grid-template-columns: 220px 1fr;
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
  padding: 8px 16px;
  font-size: 13px;
  color: var(--sre-text-secondary);
  cursor: pointer;
  position: relative;
  transition: background 120ms ease, color 120ms ease;
  border-left: 2px solid transparent;
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
.rules-main {
  min-width: 0;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 0;
  margin-bottom: 4px;
}
.toolbar-search { width: 240px; }
.toolbar-select { width: 160px; }
.toolbar-spacer { flex: 1; }

.select-all-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--sre-text-tertiary);
  cursor: pointer;
  user-select: none;
}
.select-all-label input {
  accent-color: var(--sre-primary);
}

/* List */
.rule-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: calc(100vh - 320px);
  overflow-y: auto;
}

.rule-row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px 16px 14px 20px;
  cursor: pointer;
}
.rc-check {
  width: 14px;
  height: 14px;
  cursor: pointer;
  flex-shrink: 0;
  align-self: center;
  accent-color: var(--sre-primary);
}
.rc-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.rc-title {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.rc-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rc-id {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  color: var(--sre-text-tertiary);
}
.rc-expr {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  color: var(--sre-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rc-meta {
  display: flex;
  align-items: center;
  gap: 0;
  font-size: 12px;
  color: var(--sre-text-tertiary);
  flex-wrap: wrap;
  row-gap: 4px;
}
.rc-meta-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.rc-toggle, .rc-actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

/* Dimmed (disabled) rows */
.sre-row-card[data-dim] {
  opacity: 0.55;
}
.sre-row-card[data-dim] .rc-name {
  color: var(--sre-text-secondary);
}

/* Pagination */
.pagination-wrap {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}

/* AI Generate Modal */
.ai-gen-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.ai-gen-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.ai-gen-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--sre-text-secondary);
}
.ai-gen-preview {
  margin-top: 20px;
  padding: 16px;
  background: var(--sre-bg-elevated, rgba(255,255,255,0.04));
  border: var(--sre-hairline);
  border-radius: 8px;
}
.ai-gen-preview-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}
.ai-gen-preview-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--sre-text-primary);
}
.ai-gen-confidence {
  font-size: 12px;
  font-family: var(--sre-font-mono, monospace);
  color: var(--sre-text-tertiary);
  margin-left: auto;
}
.ai-gen-expr {
  font-family: var(--sre-font-mono, monospace);
  font-size: 13px;
  color: var(--sre-text-secondary);
  background: var(--sre-bg-card, rgba(0,0,0,0.15));
  padding: 10px 12px;
  border-radius: 6px;
  margin-bottom: 10px;
  word-break: break-all;
}
.ai-gen-desc {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin-bottom: 8px;
}
.ai-gen-meta {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  margin-bottom: 4px;
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.ai-gen-meta-label {
  font-weight: 600;
  color: var(--sre-text-secondary);
}
</style>
