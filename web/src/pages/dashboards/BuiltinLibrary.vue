<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NInput, NSpace, NTag, NEmpty, NSpin, NGrid, NGi, NDrawer, NDrawerContent, NScrollbar, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { builtinDashboardApi } from '@/api'
import type { BuiltinDashboard } from '@/api/builtin-dashboard'
import PageHeader from '@/components/common/PageHeader.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import { LibraryOutline, SearchOutline, GridOutline, DownloadOutline, ArrowBackOutline, CheckmarkCircleOutline } from '@vicons/ionicons5'

const router = useRouter()
const message = useMessage()
const { t } = useI18n()

// --- State ---
const loading = ref(false)
const dashboards = ref<BuiltinDashboard[]>([])
const categories = ref<string[]>([])
const components = ref<string[]>([])
const search = ref('')
const selectedCategory = ref<string | null>(null)
const selectedComponent = ref<string | null>(null)
const importingIdents = ref<Set<string>>(new Set())
const importedIdents = ref<Set<string>>(new Set())

// Preview drawer
const showPreview = ref(false)
const previewDashboard = ref<BuiltinDashboard | null>(null)
const previewLoading = ref(false)

// --- Filtered list ---
const filteredDashboards = computed(() => {
  let list = dashboards.value
  if (selectedCategory.value) {
    list = list.filter(d => d.category === selectedCategory.value)
  }
  if (selectedComponent.value) {
    list = list.filter(d => d.component === selectedComponent.value)
  }
  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter(d =>
      d.name.toLowerCase().includes(q) ||
      d.ident.toLowerCase().includes(q) ||
      d.tags?.toLowerCase().includes(q)
    )
  }
  return list
})

// --- Category counts ---
const categoryCounts = computed(() => {
  const counts: Record<string, number> = {}
  for (const d of dashboards.value) {
    counts[d.category] = (counts[d.category] || 0) + 1
  }
  return counts
})

const componentCounts = computed(() => {
  const counts: Record<string, number> = {}
  for (const d of dashboards.value) {
    counts[d.component] = (counts[d.component] || 0) + 1
  }
  return counts
})

// --- Data fetching ---
async function fetchDashboards() {
  loading.value = true
  try {
    const res = await builtinDashboardApi.list()
    dashboards.value = res.data.data || []
  } catch (err: unknown) {
    message.error((err as Error)?.message || t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function fetchFilters() {
  try {
    const [catRes, compRes] = await Promise.all([
      builtinDashboardApi.categories(),
      builtinDashboardApi.components(),
    ])
    categories.value = catRes.data.data || []
    components.value = compRes.data.data || []
  } catch {
    /* non-critical */
  }
}

// --- Actions ---
async function handleImport(dash: BuiltinDashboard) {
  if (importingIdents.value.has(dash.ident)) return
  importingIdents.value.add(dash.ident)
  try {
    const res = await builtinDashboardApi.importDash(dash.ident)
    importedIdents.value.add(dash.ident)
    message.success(t('builtinDash.importSuccess', { name: res.data.data.name }))
  } catch (err: unknown) {
    message.error((err as Error)?.message || t('common.failed'))
  } finally {
    importingIdents.value.delete(dash.ident)
  }
}

function handleViewDashboard(ident: string) {
  const imported = importedIdents.value.has(ident)
  if (imported) {
    // Navigate to the user's dashboard list to find the imported one
    router.push('/alert/dashboards')
  }
}

async function openPreview(dash: BuiltinDashboard) {
  previewDashboard.value = dash
  showPreview.value = true
  previewLoading.value = true
  try {
    const res = await builtinDashboardApi.getByIdent(dash.ident)
    previewDashboard.value = res.data.data
  } catch (err: unknown) {
    message.error((err as Error)?.message || t('common.loadFailed'))
  } finally {
    previewLoading.value = false
  }
}

function closePreview() {
  showPreview.value = false
  previewDashboard.value = null
}

function clearFilters() {
  selectedCategory.value = null
  selectedComponent.value = null
  search.value = ''
}

// --- Parse tags helper ---
function parseTags(tagsStr: string): string[] {
  if (!tagsStr) return []
  try {
    const parsed = JSON.parse(tagsStr)
    if (Array.isArray(parsed)) return parsed
    if (typeof parsed === 'object') return Object.values(parsed).map(String)
    return [String(parsed)]
  } catch {
    return tagsStr.split(',').map(s => s.trim()).filter(Boolean)
  }
}

// --- Placeholder thumbnail colors ---
const thumbColors = [
  'linear-gradient(135deg, #0D9488 0%, #14B8A6 100%)',
  'linear-gradient(135deg, #6366F1 0%, #8B5CF6 100%)',
  'linear-gradient(135deg, #F59E0B 0%, #EF4444 100%)',
  'linear-gradient(135deg, #3B82F6 0%, #06B6D4 100%)',
  'linear-gradient(135deg, #EC4899 0%, #F43F5E 100%)',
  'linear-gradient(135deg, #10B981 0%, #34D399 100%)',
]

function getThumbColor(ident: string): string {
  let hash = 0
  for (let i = 0; i < ident.length; i++) {
    hash = ident.charCodeAt(i) + ((hash << 5) - hash)
  }
  return thumbColors[Math.abs(hash) % thumbColors.length]
}

onMounted(() => {
  fetchDashboards()
  fetchFilters()
})
</script>

<template>
  <div class="builtin-dash-page">
    <PageHeader :title="t('builtinDash.title')" :subtitle="t('builtinDash.subtitle')">
      <template #actions>
        <NButton quaternary @click="router.push('/alert/dashboards')">
          <template #icon><ArrowBackOutline /></template>
          {{ t('dashboardV2.back') }}
        </NButton>
      </template>
    </PageHeader>

    <div class="builtin-layout">
      <!-- Sidebar filters -->
      <aside class="filter-sidebar">
        <div class="filter-section">
          <div class="filter-title">{{ t('builtinDash.search') }}</div>
          <NInput
            v-model:value="search"
            :placeholder="t('builtinDash.searchPlaceholder')"
            clearable
            size="small"
          >
            <template #prefix><SearchOutline style="width: 14px; height: 14px; opacity: 0.5" /></template>
          </NInput>
        </div>

        <div v-if="categories.length > 0" class="filter-section">
          <div class="filter-title">{{ t('builtinDash.category') }}</div>
          <div class="filter-chips">
            <NTag
              v-for="cat in categories"
              :key="cat"
              :bordered="false"
              size="small"
              :type="selectedCategory === cat ? 'success' : 'default'"
              class="filter-chip"
              @click="selectedCategory = selectedCategory === cat ? null : cat"
            >
              {{ cat }}
              <span class="chip-count">{{ categoryCounts[cat] || 0 }}</span>
            </NTag>
          </div>
        </div>

        <div v-if="components.length > 0" class="filter-section">
          <div class="filter-title">{{ t('builtinDash.component') }}</div>
          <div class="filter-chips">
            <NTag
              v-for="comp in components"
              :key="comp"
              :bordered="false"
              size="small"
              :type="selectedComponent === comp ? 'info' : 'default'"
              class="filter-chip"
              @click="selectedComponent = selectedComponent === comp ? null : comp"
            >
              {{ comp }}
              <span class="chip-count">{{ componentCounts[comp] || 0 }}</span>
            </NTag>
          </div>
        </div>

        <NButton
          v-if="selectedCategory || selectedComponent || search"
          quaternary
          size="small"
          @click="clearFilters"
        >
          {{ t('builtinDash.clearFilters') }}
        </NButton>
      </aside>

      <!-- Main content -->
      <main class="builtin-main">
        <LoadingSkeleton v-if="loading" :rows="6" variant="card-grid" />

        <NEmpty
          v-else-if="filteredDashboards.length === 0"
          :description="t('builtinDash.empty')"
          style="padding: 80px 0"
        >
          <template #icon>
            <LibraryOutline style="width: 48px; height: 48px; opacity: 0.3" />
          </template>
        </NEmpty>

        <NGrid v-else :x-gap="16" :y-gap="16" cols="1 s:2 m:3 l:4" responsive="screen">
          <NGi v-for="dash in filteredDashboards" :key="dash.id">
            <div class="dash-card" @click="openPreview(dash)">
              <!-- Thumbnail placeholder -->
              <div class="dash-thumb" :style="{ background: getThumbColor(dash.ident) }">
                <GridOutline class="thumb-icon" />
                <span class="thumb-version">v{{ dash.version }}</span>
              </div>

              <!-- Card body -->
              <div class="dash-card-body">
                <div class="dash-card-name">{{ dash.name }}</div>
                <div class="dash-card-ident">{{ dash.ident }}</div>

                <div class="dash-card-tags">
                  <NTag size="tiny" :bordered="false" type="success">{{ dash.category }}</NTag>
                  <NTag size="tiny" :bordered="false" type="info">{{ dash.component }}</NTag>
                </div>

                <div v-if="dash.tags" class="dash-card-extra-tags">
                  <NTag
                    v-for="tag in parseTags(dash.tags).slice(0, 3)"
                    :key="tag"
                    size="tiny"
                    :bordered="false"
                    round
                  >
                    {{ tag }}
                  </NTag>
                </div>

                <div class="dash-card-actions" @click.stop>
                  <NButton
                    v-if="importedIdents.has(dash.ident)"
                    size="small"
                    type="success"
                    quaternary
                    @click="handleViewDashboard(dash.ident)"
                  >
                    <template #icon><CheckmarkCircleOutline /></template>
                    {{ t('builtinDash.imported') }}
                  </NButton>
                  <NButton
                    v-else
                    size="small"
                    type="primary"
                    :loading="importingIdents.has(dash.ident)"
                    @click="handleImport(dash)"
                  >
                    <template #icon><DownloadOutline /></template>
                    {{ t('builtinDash.import') }}
                  </NButton>
                </div>
              </div>
            </div>
          </NGi>
        </NGrid>
      </main>
    </div>

    <!-- Preview Drawer -->
    <NDrawer v-model:show="showPreview" :width="480" placement="right">
      <NDrawerContent :title="previewDashboard?.name || t('builtinDash.preview')">
        <NSpin :show="previewLoading">
          <div v-if="previewDashboard" class="preview-content">
            <div class="preview-field">
              <span class="preview-label">{{ t('builtinDash.ident') }}</span>
              <span class="preview-value mono">{{ previewDashboard.ident }}</span>
            </div>
            <div class="preview-field">
              <span class="preview-label">{{ t('builtinDash.category') }}</span>
              <NTag size="small" :bordered="false" type="success">{{ previewDashboard.category }}</NTag>
            </div>
            <div class="preview-field">
              <span class="preview-label">{{ t('builtinDash.component') }}</span>
              <NTag size="small" :bordered="false" type="info">{{ previewDashboard.component }}</NTag>
            </div>
            <div class="preview-field">
              <span class="preview-label">{{ t('builtinDash.version') }}</span>
              <span class="preview-value">v{{ previewDashboard.version }}</span>
            </div>
            <div v-if="previewDashboard.tags" class="preview-field">
              <span class="preview-label">{{ t('builtinDash.tags') }}</span>
              <NSpace :size="4">
                <NTag
                  v-for="tag in parseTags(previewDashboard.tags)"
                  :key="tag"
                  size="tiny"
                  :bordered="false"
                  round
                >
                  {{ tag }}
                </NTag>
              </NSpace>
            </div>
            <div v-if="previewDashboard.config" class="preview-field">
              <span class="preview-label">{{ t('builtinDash.panels') }}</span>
              <div class="preview-config">
                <pre>{{ (() => { try { const c = JSON.parse(previewDashboard.config); return `${(c.panels || []).length} panels`; } catch { return '—'; } })() }}</pre>
              </div>
            </div>
          </div>
        </NSpin>

        <template #footer>
          <NSpace>
            <NButton @click="closePreview">{{ t('common.close') }}</NButton>
            <NButton
              v-if="previewDashboard && !importedIdents.has(previewDashboard.ident)"
              type="primary"
              :loading="importingIdents.has(previewDashboard?.ident || '')"
              @click="previewDashboard && handleImport(previewDashboard)"
            >
              <template #icon><DownloadOutline /></template>
              {{ t('builtinDash.import') }}
            </NButton>
          </NSpace>
        </template>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style scoped>
.builtin-dash-page {
  max-width: 1400px;
}

.builtin-layout {
  display: flex;
  gap: 24px;
  margin-top: 8px;
}

/* --- Sidebar --- */
.filter-sidebar {
  width: 220px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
  position: sticky;
  top: 80px;
  align-self: flex-start;
}

.filter-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.filter-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.filter-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.filter-chip {
  cursor: pointer;
  transition: all 0.15s ease;
}
.filter-chip:hover {
  opacity: 0.8;
}

.chip-count {
  margin-left: 4px;
  font-size: 10px;
  opacity: 0.6;
}

/* --- Main grid --- */
.builtin-main {
  flex: 1;
  min-width: 0;
}

/* --- Card --- */
.dash-card {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-md);
  overflow: hidden;
  cursor: pointer;
  transition: all 0.2s var(--sre-ease-out);
  height: 100%;
  display: flex;
  flex-direction: column;
}
.dash-card:hover {
  border-color: var(--sre-primary);
  box-shadow: var(--sre-shadow-md);
  transform: translateY(-2px);
}

.dash-thumb {
  height: 100px;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.thumb-icon {
  width: 32px;
  height: 32px;
  color: rgba(255, 255, 255, 0.5);
}

.thumb-version {
  position: absolute;
  top: 8px;
  right: 8px;
  font-size: 10px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.8);
  background: rgba(0, 0, 0, 0.25);
  padding: 2px 6px;
  border-radius: 4px;
}

.dash-card-body {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}

.dash-card-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dash-card-ident {
  font-size: 11px;
  font-family: var(--sre-font-mono);
  color: var(--sre-text-tertiary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dash-card-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
  margin-top: 2px;
}

.dash-card-extra-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.dash-card-actions {
  margin-top: auto;
  padding-top: 8px;
}

/* --- Preview drawer --- */
.preview-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.preview-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.preview-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.preview-value {
  font-size: 14px;
  color: var(--sre-text-primary);
}
.preview-value.mono {
  font-family: var(--sre-font-mono);
  font-size: 13px;
}

.preview-config {
  background: var(--sre-bg-sunken);
  border-radius: var(--sre-radius-sm);
  padding: 8px 12px;
}
.preview-config pre {
  margin: 0;
  font-size: 12px;
  font-family: var(--sre-font-mono);
  color: var(--sre-text-secondary);
}

/* --- Responsive --- */
@media (max-width: 768px) {
  .builtin-layout {
    flex-direction: column;
  }
  .filter-sidebar {
    width: 100%;
    position: static;
    flex-direction: row;
    flex-wrap: wrap;
    gap: 12px;
  }
  .filter-section {
    flex: 1;
    min-width: 140px;
  }
}
</style>
