<script setup lang="ts">
import { h, ref, shallowRef, computed, onMounted } from 'vue'
import { useMessage, NButton, NIcon, NDropdown, NInput, NSelect, NPagination, NSwitch } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { alertRuleApi, datasourceApi } from '@/api'
import type { AlertRule, DataSource } from '@/types'
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
} from '@vicons/ionicons5'

const message = useMessage()
const { t } = useI18n()
const router = useRouter()

// ─── List state ───
const loading = ref(false)
const rules = shallowRef<AlertRule[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 50
const datasources = ref<DataSource[]>([])
const isFirstLoad = ref(true)

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
async function fetchRules() {
  loading.value = true
  try {
    const params: Record<string, any> = { page: page.value, page_size: pageSize }
    if (activeCategory.value) params.category = activeCategory.value
    const { data } = await alertRuleApi.list(params)
    rules.value = data.data.list || []
    total.value = data.data.total
  } catch (err: any) {
    message.error(err.message)
  } finally {
    loading.value = false
    if (isFirstLoad.value) {
      setTimeout(() => { isFirstLoad.value = false }, 800)
    }
  }
}

async function fetchCategories() {
  try {
    const { data } = await alertRuleApi.listCategories()
    categories.value = data.data || []
  } catch { /* ignore */ }
}

function handleCategoryChange(cat: string) {
  activeCategory.value = cat
  page.value = 1
  fetchRules()
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
    fetchRules()
  } catch (err: any) { message.error(err.message) } finally { batchLoading.value = false }
}

async function handleBatchDisable() {
  if (selectedKeys.value.length === 0) return
  batchLoading.value = true
  try {
    await alertRuleApi.batchDisable(selectedKeys.value)
    message.success(t('alert.batchDisabled', { count: selectedKeys.value.length }))
    selectedKeys.value = []
    fetchRules()
  } catch (err: any) { message.error(err.message) } finally { batchLoading.value = false }
}

async function handleBatchDelete() {
  if (selectedKeys.value.length === 0) return
  batchLoading.value = true
  try {
    await alertRuleApi.batchDelete(selectedKeys.value)
    message.success(t('alert.batchDeleted', { count: selectedKeys.value.length }))
    selectedKeys.value = []
    fetchRules()
  } catch (err: any) { message.error(err.message) } finally { batchLoading.value = false }
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
  fetchRules()
}

function onImportDone() {
  showImportModal.value = false
  fetchRules()
  fetchCategories()
}

// ─── Row actions ───
async function toggleEnabled(rule: AlertRule) {
  const newStatus = rule.status === 'enabled' ? 'disabled' : 'enabled'
  try {
    await alertRuleApi.toggleStatus(rule.id, newStatus)
    message.success(newStatus === 'enabled' ? t('alert.ruleEnabled') : t('alert.ruleDisabled'))
    fetchRules()
  } catch (err: any) { message.error(err.message) }
}

async function handleDelete(id: number) {
  try {
    await alertRuleApi.delete(id)
    message.success(t('alert.ruleDeleted'))
    fetchRules()
  } catch (err: any) { message.error(err.message) }
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
    if (window.confirm(t('alert.deleteRuleConfirm'))) handleDelete(rule.id)
  }
}

function goDetail(rule: AlertRule) {
  router.push(`/alerts/rules/${rule.id}`).catch(() => { /* no-op */ })
}

onMounted(() => {
  fetchRules()
  fetchDatasources()
  fetchCategories()
})
</script>

<template>
  <div class="rules-page">
    <PageHeader :title="t('alert.rules')" :subtitle="t('alert.rulesSubtitle')">
      <template #actions>
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
        <div v-else class="rule-list" :class="{ 'sre-stagger': isFirstLoad }">
          <div
            v-for="rule in filteredRules"
            :key="rule.id"
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
                  <span class="rc-meta-item tnum">for {{ rule.for_duration }}</span>
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
        </div>

        <!-- Pagination -->
        <div v-if="filteredRules.length > 0" class="pagination-wrap">
          <n-pagination
            v-model:page="page"
            :page-size="pageSize"
            :item-count="total"
            :page-slot="7"
            @update:page="fetchRules"
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
</style>
