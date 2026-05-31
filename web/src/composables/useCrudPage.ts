import { ref, type Ref } from 'vue'
import { useMessage, useDialog } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { getErrorMessage } from '@/utils/format'

/**
 * Standard API module shape that useCrudPage expects.
 * Each method should return a standard axios promise.
 */
export interface CrudApiModule<T> {
  list: (params?: Record<string, unknown>) => Promise<{ data: { data: PageData<T> } }>
  create: (data: Partial<T>) => Promise<unknown>
  update: (id: number, data: Partial<T>) => Promise<unknown>
  delete: (id: number) => Promise<unknown>
}

/**
 * Flexible page data shape — handles both standard and alternate API responses.
 * Standard: `{ list: [...], total: N }`
 * Alternate: `{ items: [...], count: N }`
 */
export interface PageData<T> {
  list?: T[]
  items?: T[]
  total?: number
  count?: number
  [key: string]: unknown
}

/**
 * Normalizes paginated API response data to a consistent `{ list, total }` shape.
 * Handles both standard (`list`/`total`) and alternate (`items`/`count`) formats.
 */
function normalizePageData<T>(data: PageData<T>): { list: T[]; total: number } {
  return {
    list: data.list ?? data.items ?? [],
    total: data.total ?? data.count ?? 0,
  }
}

export interface UseCrudPageOptions<T> {
  /** The API module providing list/create/update/delete methods */
  api: CrudApiModule<T>
  /** Factory function returning a fresh default form object */
  defaultForm: () => Partial<T>
  /** i18n keys for messages */
  i18nKeys: {
    created?: string
    updated?: string
    deleted?: string
    deleteConfirm?: string
    nameRequired?: string
    createTitle?: string
    editTitle?: string
  }
  /** Optional function to transform a row into form fields for editing */
  rowToForm?: (row: T) => Partial<T>
  /** Optional function to transform form data before sending to API */
  formToPayload?: (form: Partial<T>) => Partial<T>
  /** Optional custom validation; return error message string or null if valid */
  validate?: (form: Partial<T>) => string | null
  /** Items per page (default 100) */
  pageSize?: number
  /** Called after successful create/update/delete to refresh additional data */
  onAfterSave?: () => void
}

export interface UseCrudPageReturn<T> {
  // List state
  loading: Ref<boolean>
  items: Ref<T[]>
  total: Ref<number>
  page: Ref<number>
  pageSize: Ref<number>
  // Search
  search: Ref<string>
  // Modal state
  showModal: Ref<boolean>
  modalTitle: Ref<string>
  editingId: Ref<number | null>
  saving: Ref<boolean>
  form: Ref<Partial<T>>
  // Actions
  fetchList: () => Promise<void>
  refresh: () => Promise<void>
  openCreate: () => void
  openEdit: (row: T) => void
  handleSave: () => Promise<void>
  handleDelete: (id: number) => Promise<void>
  resetForm: () => void
  confirmDelete: (id: number) => void
  /** FE3-6: Batch delete with confirmation dialog showing item count */
  confirmBatchDelete: (ids: number[]) => void
}

/**
 * Generalized CRUD page composable.
 *
 * Encapsulates list loading, pagination, search, modal state, form management,
 * and standard CRUD operations. Works with any API module that follows the
 * standard list/create/update/delete pattern.
 *
 * @example
 * ```ts
 * const crud = useCrudPage({
 *   api: notifyMediaApi,
 *   defaultForm: () => ({ name: '', type: 'lark_webhook', is_enabled: true }),
 *   i18nKeys: { created: 'media.created', updated: 'media.updated', deleted: 'media.deleted' },
 *   rowToForm: (row) => ({ name: row.name, type: row.type, ... }),
 * })
 * ```
 */
export function useCrudPage<T extends { id: number }>(
  options: UseCrudPageOptions<T>,
): UseCrudPageReturn<T> {
  const message = useMessage()
  const dialog = useDialog()
  const { t } = useI18n()

  // List state
  const loading = ref(false)
  const items = ref<T[]>([]) as Ref<T[]>
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(options.pageSize ?? 100)
  const search = ref('')

  // Incrementing request ID to discard stale responses on rapid page/filter changes
  let requestId = 0

  // Modal state
  const showModal = ref(false)
  const modalTitle = ref('')
  const editingId = ref<number | null>(null)
  const saving = ref(false)
  const form = ref<Partial<T>>(options.defaultForm()) as Ref<Partial<T>>

  // ---- List ----

  async function fetchList() {
    const id = ++requestId
    loading.value = true
    try {
      const { data } = await options.api.list({ page: page.value, page_size: pageSize.value })
      if (id !== requestId) return // stale response, discard
      const pageData = normalizePageData<T>(data.data)
      items.value = pageData.list
      total.value = pageData.total
    } catch (err: unknown) {
      if (id !== requestId) return // stale response, discard
      message.error(getErrorMessage(err))
    } finally {
      if (id === requestId) loading.value = false
    }
  }

  async function refresh() {
    page.value = 1
    await fetchList()
  }

  // ---- Modal ----

  function resetForm() {
    form.value = options.defaultForm()
  }

  function openCreate() {
    editingId.value = null
    modalTitle.value = options.i18nKeys.createTitle ? t(options.i18nKeys.createTitle) : t('common.create')
    resetForm()
    showModal.value = true
  }

  function openEdit(row: T) {
    editingId.value = row.id
    modalTitle.value = options.i18nKeys.editTitle ? t(options.i18nKeys.editTitle) : t('common.edit')
    form.value = options.rowToForm ? options.rowToForm(row) : { ...row }
    showModal.value = true
  }

  // ---- Save ----

  async function handleSave() {
    // Validate
    if (options.validate) {
      const err = options.validate(form.value)
      if (err) {
        message.warning(err)
        return
      }
    }

    saving.value = true
    try {
      const payload = options.formToPayload ? options.formToPayload(form.value) : form.value
      if (editingId.value) {
        await options.api.update(editingId.value, payload as Partial<T>)
        if (options.i18nKeys.updated) message.success(t(options.i18nKeys.updated))
      } else {
        await options.api.create(payload as Partial<T>)
        if (options.i18nKeys.created) message.success(t(options.i18nKeys.created))
      }
      showModal.value = false
      await fetchList()
      options.onAfterSave?.()
    } catch (err: unknown) {
      message.error(getErrorMessage(err))
    } finally {
      saving.value = false
    }
  }

  // ---- Delete ----

  async function handleDelete(id: number) {
    try {
      await options.api.delete(id)
      if (options.i18nKeys.deleted) message.success(t(options.i18nKeys.deleted))
      await fetchList()
    } catch (err: unknown) {
      message.error(getErrorMessage(err))
    }
  }

  function confirmDelete(id: number) {
    const confirmMsg = options.i18nKeys.deleteConfirm ? t(options.i18nKeys.deleteConfirm) : t('common.confirmDelete')
    dialog.warning({
      title: t('common.confirmDelete'),
      content: confirmMsg,
      positiveText: t('common.confirm'),
      negativeText: t('common.cancel'),
      onPositiveClick: () => handleDelete(id),
    })
  }

  // FE3-6: Batch delete with parallel execution via Promise.allSettled
  async function handleBatchDelete(ids: number[]) {
    const results = await Promise.allSettled(ids.map((id) => options.api.delete(id)))
    const successCount = results.filter((r) => r.status === 'fulfilled').length
    const failCount = results.length - successCount
    if (failCount > 0) {
      message.error(t('common.deleteFailed') || `${failCount} deletions failed`)
    }
    if (successCount > 0) {
      const msg = options.i18nKeys.deleted ? t(options.i18nKeys.deleted) : t('common.deleteSuccess')
      message.success(`${successCount} ${msg}`)
      await fetchList()
    }
  }

  function confirmBatchDelete(ids: number[]) {
    if (!ids.length) return
    const confirmMsg = t('common.confirmBatchDelete', { count: ids.length }) || `Delete ${ids.length} items?`
    dialog.warning({
      title: t('common.confirmDelete'),
      content: confirmMsg,
      positiveText: t('common.confirm'),
      negativeText: t('common.cancel'),
      onPositiveClick: () => handleBatchDelete(ids),
    })
  }

  return {
    loading,
    items,
    total,
    page,
    pageSize,
    search,
    showModal,
    modalTitle,
    editingId,
    saving,
    form,
    fetchList,
    refresh,
    openCreate,
    openEdit,
    handleSave,
    handleDelete,
    resetForm,
    confirmDelete,
    confirmBatchDelete,
  }
}
