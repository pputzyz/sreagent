<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NInput, NSpace, NPopconfirm, NPagination } from 'naive-ui'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { dashboardV2Api } from '@/api'
import type { DashboardV2 } from '@/types/dashboard'
import PageHeader from '@/components/common/PageHeader.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { AddOutline, BarChartOutline, ChevronForwardOutline } from '@vicons/ionicons5'
import { relTime } from '@/utils/format'

const router = useRouter()
const message = useMessage()
const { t } = useI18n()
const loading = ref(false)
const firstLoaded = ref(false)
const search = ref('')
const list = ref<DashboardV2[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const isEmpty = computed(() => firstLoaded.value && !loading.value && list.value.length === 0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

async function fetchList() {
  loading.value = true
  try {
    const res = await dashboardV2Api.list({ page: page.value, page_size: pageSize.value, search: search.value || undefined })
    list.value = res.data.data.list || []
    total.value = res.data.data.total || 0
    firstLoaded.value = true
  } catch (err: any) {
    message.error(err.message || t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

function onSearch() {
  page.value = 1
  fetchList()
}

function onPageChange(p: number) {
  page.value = p
  fetchList()
}

async function handleDelete(id: number) {
  try {
    await dashboardV2Api.delete(id)
    message.success(t('dashboardV2.deleted'))
    fetchList()
  } catch (err: any) {
    message.error(err.message || t('common.deleteFailed'))
  }
}

function handleEdit(id: number) {
  router.push({ name: 'DashboardV2View', params: { id } })
}

onMounted(fetchList)
</script>

<template>
  <div class="dash-list-page">
    <PageHeader :title="t('dashboardV2.title')" :subtitle="t('dashboardV2.subtitle')">
      <template #actions>
        <NInput
          v-model:value="search"
          :placeholder="t('common.search')"
          clearable
          class="search-input"
          @update:value="onSearch"
        />
        <NButton type="primary" @click="router.push({ name: 'DashboardV2View', params: { id: 'new' } })">
          <template #icon>
            <AddOutline />
          </template>
          {{ t('dashboardV2.newDashboard') }}
        </NButton>
      </template>
    </PageHeader>

    <!-- Loading skeleton -->
    <LoadingSkeleton v-if="loading && list.length === 0" :rows="5" variant="card-grid" />

    <!-- Empty state -->
    <EmptyState
      v-else-if="isEmpty"
      :icon="BarChartOutline"
      :title="t('dashboardV2.emptyHint') || 'No dashboards yet'"
      :description="t('dashboardV2.subtitle')"
      :primary-text="t('dashboardV2.newDashboard')"
      @primary="router.push({ name: 'DashboardV2View', params: { id: 'new' } })"
    />

    <!-- List -->
    <n-spin v-else :show="loading && list.length > 0">
      <div class="dash-list sre-stagger">
        <div
          v-for="dash in list"
          :key="dash.id"
          class="sre-row-card dash-card"
          @click="handleEdit(dash.id)"
        >
          <div class="dash-content">
            <div class="dash-headline">
              <span class="dash-name">{{ dash.name }}</span>
            </div>
            <div v-if="dash.description" class="dash-desc">{{ dash.description }}</div>
            <div class="dash-meta">
              <span class="tnum">{{ relTime(dash.created_at) }}</span>
              <template v-if="dash.is_public">
                <span class="sre-meta-divider" />
                <span>{{ t('dashboardV2.public') || 'Public' }}</span>
              </template>
            </div>
          </div>
          <div class="dash-actions" @click.stop>
            <NSpace :size="4">
              <NButton quaternary size="tiny" @click="handleEdit(dash.id)">
                {{ t('common.edit') }}
              </NButton>
              <NPopconfirm @positive-click="handleDelete(dash.id)">
                <template #trigger>
                  <NButton quaternary size="tiny" type="error">
                    {{ t('common.delete') }}
                  </NButton>
                </template>
                {{ t('common.confirmDelete') }}
              </NPopconfirm>
            </NSpace>
          </div>
          <div class="dash-arrow">
            <ChevronForwardOutline />
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <div v-if="total > pageSize" class="pagination-row">
        <NPagination
          v-model:page="page"
          :page-count="totalPages"
          @update:page="onPageChange"
        />
      </div>
    </n-spin>
  </div>
</template>

<style scoped>
.dash-list-page {
  max-width: 1400px;
}

.search-input {
  width: 200px;
}

.dash-list {
  display: flex;
  flex-direction: column;
  gap: var(--sre-row-gap, 4px);
}

.dash-card {
  cursor: pointer;
}

.dash-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.dash-headline {
  display: flex;
  align-items: center;
  gap: 8px;
}

.dash-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dash-desc {
  font-size: 13px;
  color: var(--sre-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dash-meta {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  display: flex;
  align-items: center;
  flex-wrap: wrap;
}

.dash-actions {
  flex-shrink: 0;
  align-self: center;
}

.dash-arrow {
  color: var(--sre-text-tertiary);
  flex-shrink: 0;
  align-self: center;
  opacity: 0.5;
  transition: opacity var(--sre-duration-fast) ease;
  width: 16px;
  height: 16px;
}

.sre-row-card:hover .dash-arrow {
  opacity: 1;
}

.pagination-row {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
