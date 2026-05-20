<script setup lang="ts">
import { h, ref, computed, onMounted, watch } from 'vue'
import { useMessage, useDialog, NButton, NIcon, NDropdown, NInput, NSelect, NPagination, NSwitch, NModal, NForm, NFormItem, NSpace, NSpin, NAlert, NTag } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { alertRuleApi, datasourceApi } from '@/api'
import type { AlertRule, DataSource } from '@/types'
import type { RuleGenerateResult, MuteRuleGenerateResult } from '@/types/ai-module'
import { usePaginatedList, useAIModule, useFilterMemory, usePermissions } from '@/composables'
import { getErrorMessage } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import RuleFormModal from '@/components/alert/RuleFormModal.vue'
import ImportModal from '@/components/alert/ImportModal.vue'
import BatchOperations from '@/components/alert/BatchOperations.vue'
import AIGenerateModal from '@/components/alert-rule/AIGenerateModal.vue'
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
const { hasPerm } = usePermissions()

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
    message.error(getErrorMessage(err))
  },
})

// ─── Filters ───
const searchKeyword = ref('')
const filterDatasource = ref<number | null>(null)
const filterSeverity = ref<string | null>(null)
const filterStatus = ref<string | null>(null)

// Persist filter state to localStorage
const filterMemory = useFilterMemory('alert-rules')
filterMemory.bindRefs({ searchKeyword, filterDatasource, filterSeverity, filterStatus })

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

function openAIGenerate() {
  showAIModal.value = true
}

async function handleAIGenerated(result: RuleGenerateResult | MuteRuleGenerateResult) {
  try {
    await alertRuleApi.create({
      name: result.name,
      expression: (result as RuleGenerateResult).expression || '',
      for_duration: (result as RuleGenerateResult).for_duration || '0s',
      severity: ((result as RuleGenerateResult).severity as AlertRule['severity']) || 'warning',
      labels: (result as RuleGenerateResult).labels || {},
      annotations: (result as RuleGenerateResult).annotations || {},
    })
    message.success(t('common.createSuccess'))
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

// ─── Computed ───
const severityFilterOptions = computed(() => [
  { label: t('alert.critical'), value: 'critical' },
  { label: t('alert.warning'), value: 'warning' },
  { label: t('alert.info'), value: 'info' },
])

const statusFilterOptions = computed(() => [
  { label: t('alert.statusDraft'), value: 'draft' },
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
  const ids = [...selectedKeys.value]
  batchLoading.value = true
  try {
    await alertRuleApi.batchEnable(ids)
    message.success(t('alert.batchEnabled', { count: ids.length }))
    selectedKeys.value = []
    fetchList()
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { batchLoading.value = false }
}

async function handleBatchDisable() {
  if (selectedKeys.value.length === 0) return
  const ids = [...selectedKeys.value]
  batchLoading.value = true
  try {
    await alertRuleApi.batchDisable(ids)
    message.success(t('alert.batchDisabled', { count: ids.length }))
    selectedKeys.value = []
    fetchList()
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { batchLoading.value = false }
}

async function doBatchDelete() {
  if (selectedKeys.value.length === 0) return
  const ids = [...selectedKeys.value]
  batchLoading.value = true
  try {
    await alertRuleApi.batchDelete(ids)
    message.success(t('alert.batchDeleted', { count: ids.length }))
    selectedKeys.value = []
    fetchList()
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { batchLoading.value = false }
}

function handleBatchDelete() {
  if (selectedKeys.value.length === 0) return
  dialog.warning({
    title: t('common.confirmDelete'),
    content: t('alert.batchDeleteConfirm', { count: selectedKeys.value.length }),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: () => doBatchDelete(),
  })
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
  if (rule.status === 'draft') return
  const newStatus = rule.status === 'active' ? 'disabled' : 'active'
  try {
    await alertRuleApi.toggleStatus(rule.id, newStatus)
    message.success(newStatus === 'active' ? t('alert.ruleEnabled') : t('alert.ruleDisabled'))
    fetchList()
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
}

function statusTagType(status: string): 'default' | 'success' | 'warning' {
  if (status === 'active') return 'success'
  if (status === 'disabled') return 'warning'
  return 'default'
}

function statusLabel(status: string): string {
  if (status === 'draft') return t('alert.statusDraft')
  if (status === 'active') return t('common.enabled')
  if (status === 'disabled') return t('common.disabled')
  return status
}

async function handleDelete(id: number) {
  try {
    await alertRuleApi.delete(id)
    message.success(t('alert.ruleDeleted'))
    fetchList()
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
}

function rowActions(rule: AlertRule) {
  const actions: Array<{ label?: string; key: string; icon?: () => ReturnType<typeof h>; type?: 'divider' }> = [
    { label: t('common.edit'), key: 'edit', icon: () => h(NIcon, { component: CreateOutline }) },
  ]
  if (rule.status !== 'draft') {
    actions.push({ label: rule.status === 'active' ? t('common.disabled') : t('common.enabled'), key: 'toggle', icon: () => h(NIcon, { component: PowerOutline }) })
  }
  actions.push(
    { label: t('common.duplicate'), key: 'duplicate', icon: () => h(NIcon, { component: CopyOutline }) },
    { type: 'divider', key: 'd1' },
    { label: t('common.delete'), key: 'delete', icon: () => h(NIcon, { component: TrashOutline }) },
  )
  return actions
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
        <n-button v-if="isAIModuleEnabled('rule_gen') && hasPerm('rules.create')" size="small" secondary @click="openAIGenerate">
          <template #icon><n-icon :component="SparklesOutline" /></template>
          {{ t('alert.aiGenerate') }}
        </n-button>
        <n-button v-if="hasPerm('rules.manage')" size="small" secondary @click="showImportModal = true">
          <template #icon><n-icon :component="CloudUploadOutline" /></template>
          {{ t('alert.importExport') }}
        </n-button>
        <n-button v-if="hasPerm('rules.create')" size="small" type="primary" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('alert.createRule') }}
        </n-button>
      </template>
    </PageHeader>

    <div class="rules-layout">
      <!-- Sidebar: categories -->
      <aside class="cat-aside">
        <div class="sre-label-eyebrow cat-eyebrow">{{ t('alert.category') }}</div>
        <button
          type="button"
          class="cat-item"
          :class="{ active: activeCategory === '' }"
          @click="handleCategoryChange('')"
        >
          <span class="cat-name">{{ t('alert.allCategories') }}</span>
          <span class="cat-count tnum">{{ allCount }}</span>
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
          <span class="cat-count tnum">{{ categoryCounts[cat] ?? '' }}</span>
        </button>
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
            <span>{{ t('common.selectAll') }}</span>
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
          :title="t('alert.noRules')"
          :description="t('alert.rulesSubtitle')"
          :primary-text="t('alert.createFirstRule')"
          :secondary-text="t('alert.importFile')"
          @primary="openCreate"
          @secondary="showImportModal = true"
        />

        <!-- Rule list -->
        <div
          v-else
          class="rule-list"
          :class="{ 'sre-stagger': isFirstLoad }"
        >
          <div
            v-for="rule in filteredRules"
            :key="rule.id"
            class="sre-row-card rule-row"
            :data-severity="severitySlot(rule.severity)"
            :data-dim="rule.status !== 'active' || undefined"
            :data-status="rule.status"
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
                <NTag :type="statusTagType(rule.status)" size="small" :bordered="false" round>
                  {{ statusLabel(rule.status) }}
                </NTag>
                <span class="sre-meta-divider"></span>
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
              <n-switch :value="rule.status === 'active'" size="small" :disabled="rule.status === 'draft'" :aria-label="rule.name" @update:value="toggleEnabled(rule)" />
            </div>
            <div class="rc-actions" @click.stop>
              <n-dropdown :options="rowActions(rule)" trigger="click" @select="(k: string) => onRowAction(k, rule)">
                <n-button quaternary circle size="small">
                  <template #icon><n-icon :component="EllipsisHorizontalOutline" /></template>
                </n-button>
              </n-dropdown>
            </div>
          </div>
        </div>

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
    <AIGenerateModal
      v-model:visible="showAIModal"
      rule-type="rule"
      :datasource-options="datasourceOptions"
      @generated="handleAIGenerated"
    />
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
