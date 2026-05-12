<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { postMortemApi } from '@/api'
import { formatTime } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import {
  RefreshCw,
  FileText,
  ChevronRight,
  Clock,
  User,
} from 'lucide-vue-next'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

const loading = ref(false)
const postMortems = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const statusFilter = ref('')

async function loadPostMortems() {
  loading.value = true
  try {
    const params: any = {
      page: page.value,
      page_size: pageSize.value,
    }
    if (statusFilter.value) {
      params.status = statusFilter.value
    }
    const res = await postMortemApi.list(params)
    postMortems.value = res.data.data?.list ?? []
    total.value = res.data.data?.total ?? 0
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

function gotoIncident(incidentId: number) {
  router.push(`/incidents/${incidentId}`)
}

function authorName(pm: any): string {
  return pm.author?.display_name || pm.author?.username || '—'
}

const isEmpty = computed(() => !loading.value && postMortems.value.length === 0)
const hasFilters = computed(() => statusFilter.value !== '')
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

onMounted(loadPostMortems)
</script>

<template>
  <div class="postmortems-page">
    <PageHeader :title="t('postMortem.title')">
      <template #actions>
        <n-button circle quaternary @click="loadPostMortems" :aria-label="t('common.refresh')">
          <template #icon><RefreshCw :size="18" /></template>
        </n-button>
      </template>
    </PageHeader>

    <!-- Status filter tabs -->
    <div class="filter-bar">
      <div class="filter-group">
        <span class="filter-label">{{ t('common.status') }}</span>
        <n-radio-group v-model:value="statusFilter" size="small" @update:value="page = 1; loadPostMortems()">
          <n-radio-button value="">{{ t('common.all') }}</n-radio-button>
          <n-radio-button value="draft">{{ t('postMortem.draft') }}</n-radio-button>
          <n-radio-button value="published">{{ t('postMortem.published') }}</n-radio-button>
        </n-radio-group>
      </div>
      <div class="filter-spacer" />
      <span class="result-count" v-if="total > 0">{{ total }}</span>
    </div>

    <!-- Content -->
    <n-spin :show="loading && postMortems.length > 0">
      <LoadingSkeleton v-if="loading && postMortems.length === 0" :rows="6" variant="row" />

      <EmptyState
        v-else-if="isEmpty"
        :icon="FileText"
        :title="t('postMortem.noPostMortem')"
        :description="hasFilters ? t('common.noData') : t('postMortem.noPostMortem')"
        :primary-text="hasFilters ? t('common.all') : undefined"
        @primary="statusFilter = ''; loadPostMortems()"
      />

      <div v-else class="pm-list">
        <div
          v-for="pm in postMortems"
          :key="pm.id"
          class="pm-row"
          @click="gotoIncident(pm.incident_id)"
        >
          <span class="status-bar" :data-status="pm.status" />

          <div class="row-body">
            <div class="row-line-1">
              <FileText :size="16" class="pm-icon" />
              <span class="pm-title">{{ pm.title || pm.incident?.title || '—' }}</span>
              <span
                class="status-pill"
                :data-status="pm.status"
              >
                {{ pm.status === 'published' ? t('postMortem.published') : t('postMortem.draft') }}
              </span>
            </div>

            <div class="row-line-2">
              <span v-if="pm.incident?.title" class="meta-item incident-link">
                {{ pm.incident.title }}
              </span>
              <span class="meta-item">
                <User :size="12" />
                {{ authorName(pm) }}
              </span>
              <span v-if="pm.published_at" class="meta-item">
                <Clock :size="12" />
                {{ formatTime(pm.published_at) }}
              </span>
              <span v-else-if="pm.updated_at" class="meta-item">
                <Clock :size="12" />
                {{ t('postMortem.lastUpdated') }} {{ formatTime(pm.updated_at) }}
              </span>
            </div>
          </div>

          <ChevronRight :size="18" class="chevron" />
        </div>
      </div>

      <div v-if="total > pageSize" class="pagination">
        <n-pagination
          v-model:page="page"
          :page-count="totalPages"
          @update:page="loadPostMortems"
        />
      </div>
    </n-spin>
  </div>
</template>

<style scoped>
.postmortems-page { max-width: 1400px; font-family: var(--sre-font-sans); }

.filter-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  padding: 10px 14px;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: 8px;
  margin-bottom: 16px;
}
.filter-group {
  display: flex;
  align-items: center;
  gap: 8px;
}
.filter-label {
  font-size: 12px;
  color: var(--sre-text-tertiary);
}
.filter-spacer { flex: 1; }
.result-count {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  font-variant-numeric: tabular-nums;
}

/* Post-mortem list */
.pm-list {
  display: flex;
  flex-direction: column;
  gap: var(--sre-row-gap, 6px);
}
.pm-row {
  position: relative;
  display: flex;
  align-items: center;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: 8px;
  padding: 10px 14px 10px 20px;
  cursor: pointer;
  transition: background-color 0.15s ease, border-color 0.15s ease, transform 0.15s ease;
  overflow: hidden;
}
.pm-row:hover {
  background: var(--sre-bg-hover);
  border-color: var(--sre-primary);
}
.pm-row:hover .chevron {
  opacity: 1;
}

.status-bar {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 4px;
}
.status-bar[data-status="draft"] { background: var(--sre-warning); }
.status-bar[data-status="published"] { background: var(--sre-success); }

.row-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.row-line-1 {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  min-width: 0;
}
.pm-icon {
  flex-shrink: 0;
  color: var(--sre-text-tertiary);
}
.pm-title {
  font-weight: 600;
  color: var(--sre-text-primary);
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}

.row-line-2 {
  display: flex;
  align-items: center;
  gap: 14px;
  font-size: var(--sre-fs-xs, 11px);
  color: var(--sre-text-tertiary);
}
.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  white-space: nowrap;
}
.incident-link {
  color: var(--sre-text-secondary);
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
}
.status-pill[data-status="draft"] {
  color: var(--sre-warning);
  background: var(--sre-warning-soft);
}
.status-pill[data-status="published"] {
  color: var(--sre-success);
  background: var(--sre-success-soft);
}

.chevron {
  flex-shrink: 0;
  margin-left: 12px;
  color: var(--sre-text-tertiary);
  opacity: 0;
  transition: opacity 0.15s ease;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
