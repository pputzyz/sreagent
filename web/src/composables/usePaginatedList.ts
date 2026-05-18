import { ref, type Ref } from 'vue'

interface PaginatedListOptions<T, P = Record<string, unknown>> {
  /** API call function: receives merged params, returns { data: { data: { list, total } } } */
  apiFn: (params: P & { page: number; page_size: number }) => Promise<{ data: { data: { list: T[]; total: number } } }>
  /** Items per page (default 20) */
  pageSize?: number
  /** Additional params to merge into every request (reactive getter) */
  extraParams?: () => Record<string, unknown>
  /** Error handler (defaults to console.error) */
  onError?: (err: unknown) => void
}

interface PaginatedListReturn<T> {
  loading: Ref<boolean>
  items: Ref<T[]>
  total: Ref<number>
  page: Ref<number>
  pageSize: Ref<number>
  fetchList: () => Promise<void>
  /** Reset page to 1 and fetch */
  refresh: () => Promise<void>
}

/**
 * Composable for paginated list data fetching.
 * Manages loading, items, total, page state and provides fetch/refresh methods.
 */
export function usePaginatedList<T, P = Record<string, unknown>>(
  options: PaginatedListOptions<T, P>,
): PaginatedListReturn<T> {
  const loading = ref(false)
  const items = ref<T[]>([]) as Ref<T[]>
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(options.pageSize ?? 20)

  async function fetchList() {
    loading.value = true
    try {
      const extra = options.extraParams ? options.extraParams() : {}
      const params = {
        ...extra,
        page: page.value,
        page_size: pageSize.value,
      } as P & { page: number; page_size: number }
      const { data } = await options.apiFn(params)
      items.value = data.data.list || []
      total.value = data.data.total
    } catch (err: unknown) {
      if (options.onError) {
        options.onError(err)
      } else {
        console.error('usePaginatedList fetch error:', err)
      }
    } finally {
      loading.value = false
    }
  }

  async function refresh() {
    page.value = 1
    await fetchList()
  }

  return {
    loading,
    items,
    total,
    page,
    pageSize,
    fetchList,
    refresh,
  }
}
