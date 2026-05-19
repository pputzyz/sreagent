<script setup lang="ts">
/**
 * CrudListPage — Generic CRUD list page shell.
 *
 * Provides the common page structure: header, toolbar (search + filters + create button),
 * empty state, loading state, content area, and an add/edit modal.
 *
 * Slots:
 *   - header-actions: extra buttons in the header area (e.g., refresh)
 *   - filters: extra filter controls after the search input
 *   - list: the main list/table content (receives filtered items)
 *   - empty-icon: icon component for the empty state
 *   - modal-form: the form fields inside the modal
 *   - modal-footer: custom modal footer (replaces default save/cancel buttons)
 *
 * @example
 * ```vue
 * <CrudListPage
 *   :title="t('media.title')"
 *   :subtitle="t('media.subtitle')"
 *   :create-label="t('media.create')"
 *   :loading="crud.loading.value"
 *   :items="filteredItems"
 *   :show-modal="crud.showModal.value"
 *   :modal-title="crud.modalTitle.value"
 *   :saving="crud.saving.value"
 *   :search-value="crud.search.value"
 *   @create="crud.openCreate()"
 *   @update:search-value="(v) => crud.search.value = v"
 *   @save="crud.handleSave()"
 *   @close-modal="crud.showModal.value = false"
 * >
 *   <template #list="{ items }">
 *     <!-- render your list here -->
 *   </template>
 *   <template #modal-form>
 *     <!-- form fields -->
 *   </template>
 * </CrudListPage>
 * ```
 */
import { type Component } from 'vue'
import { AddOutline, SearchOutline } from '@vicons/ionicons5'
import PageHeader from './PageHeader.vue'
import EmptyState from './EmptyState.vue'
import LoadingSkeleton from './LoadingSkeleton.vue'

defineProps<{
  /** Page title */
  title: string
  /** Page subtitle / description */
  subtitle?: string
  /** Label for the create button */
  createLabel?: string
  /** Whether the list is loading */
  loading?: boolean
  /** The list items (after filtering) */
  items?: unknown[]
  /** Total items before filtering (for the counter) */
  totalCount?: number
  /** Whether the modal is visible */
  showModal?: boolean
  /** Modal title */
  modalTitle?: string
  /** Whether the save action is in progress */
  saving?: boolean
  /** Search input value */
  searchValue?: string
  /** Search input placeholder */
  searchPlaceholder?: string
  /** Icon for empty state */
  emptyIcon?: Component
  /** Title for empty state */
  emptyTitle?: string
  /** Description for empty state */
  emptyDescription?: string
  /** Label for empty state primary button */
  emptyActionLabel?: string
  /** Modal width (default 600px) */
  modalWidth?: string | number
  /** Hide the search toolbar */
  hideSearch?: boolean
  /** Hide the create button */
  hideCreate?: boolean
}>()

const emit = defineEmits<{
  create: []
  save: []
  'close-modal': []
  'update:search-value': [value: string]
}>()
</script>

<template>
  <div class="crud-page">
    <!-- Header -->
    <PageHeader :title="title" :subtitle="subtitle">
      <template #actions>
        <slot name="header-actions" />
        <n-button
          v-if="!hideCreate"
          type="primary"
          size="small"
          @click="emit('create')"
        >
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ createLabel || $t('common.create') }}
        </n-button>
      </template>
    </PageHeader>

    <!-- Toolbar -->
    <div v-if="!hideSearch" class="crud-toolbar">
      <n-input
        :value="searchValue"
        size="small"
        :placeholder="searchPlaceholder || $t('common.search')"
        clearable
        style="width: 240px"
        @update:value="(v: string) => emit('update:search-value', v)"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <slot name="filters" />
      <span v-if="totalCount !== undefined" class="crud-count tnum">
        {{ items?.length ?? 0 }} / {{ totalCount }}
      </span>
    </div>

    <!-- Content -->
    <LoadingSkeleton v-if="loading && (!items || items.length === 0)" :rows="6" variant="row" />

    <EmptyState
      v-else-if="!loading && items && items.length === 0"
      :icon="emptyIcon || AddOutline"
      :title="emptyTitle || $t('common.noData')"
      :description="emptyDescription"
      :primary-text="!hideCreate ? (emptyActionLabel || createLabel || $t('common.create')) : undefined"
      @primary="emit('create')"
    />

    <n-spin v-else :show="loading">
      <slot name="list" :items="items || []" />
    </n-spin>

    <!-- Modal -->
    <n-modal
      :show="showModal"
      preset="card"
      :title="modalTitle"
      :bordered="false"
      :style="{ width: typeof modalWidth === 'number' ? modalWidth + 'px' : (modalWidth || '600px') }"
      @update:show="(v: boolean) => { if (!v) emit('close-modal') }"
    >
      <n-form label-placement="top">
        <slot name="modal-form" />
      </n-form>
      <template v-if="!$slots['modal-footer']" #action>
        <n-space justify="end">
          <n-button @click="emit('close-modal')">{{ $t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="emit('save')">
            {{ $t('common.save') }}
          </n-button>
        </n-space>
      </template>
      <template v-else #action>
        <slot name="modal-footer" />
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.crud-page {
  font-family: var(--sre-font-sans);
  max-width: 1400px;
}

.crud-toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
}

.crud-count {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin-left: auto;
  font-variant-numeric: tabular-nums;
}
</style>
