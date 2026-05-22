import { describe, it, expect, vi } from 'vitest'
import { nextTick } from 'vue'
import { usePaginatedList } from './usePaginatedList'

function makeApiFn(list: unknown[], total = list.length) {
  return vi.fn(async () => ({
    data: { data: { list, total } },
  }))
}

describe('usePaginatedList', () => {
  it('fetches list and populates items/total', async () => {
    const apiFn = makeApiFn([{ id: 1 }, { id: 2 }], 10)
    const { items, total, loading, fetchList } = usePaginatedList({ apiFn })

    await fetchList()
    expect(items.value).toEqual([{ id: 1 }, { id: 2 }])
    expect(total.value).toBe(10)
    expect(loading.value).toBe(false)
  })

  it('calls apiFn with correct page/page_size', async () => {
    const apiFn = makeApiFn([])
    const { fetchList, page, pageSize } = usePaginatedList({ apiFn, pageSize: 50 })

    page.value = 3
    await fetchList()

    expect(apiFn).toHaveBeenCalledWith(expect.objectContaining({
      page: 3,
      page_size: 50,
    }))
  })

  it('merges extraParams into request', async () => {
    const apiFn = makeApiFn([])
    const { fetchList } = usePaginatedList({
      apiFn,
      extraParams: () => ({ status: 'firing' }),
    })

    await fetchList()

    expect(apiFn).toHaveBeenCalledWith(expect.objectContaining({
      status: 'firing',
      page: 1,
      page_size: 20,
    }))
  })

  it('refresh resets page to 1', async () => {
    const apiFn = makeApiFn([])
    const { page, refresh } = usePaginatedList({ apiFn })

    page.value = 5
    await refresh()
    expect(page.value).toBe(1)
  })

  it('handles total being undefined (fallback to 0)', async () => {
    const apiFn = vi.fn(async () => ({
      data: { data: { list: [{ id: 1 }], total: undefined as unknown as number } },
    }))
    const { total, fetchList } = usePaginatedList({ apiFn })

    await fetchList()
    expect(total.value).toBe(0)
  })

  it('handles empty list response', async () => {
    const apiFn = makeApiFn([])
    const { items, total, fetchList } = usePaginatedList({ apiFn })

    await fetchList()
    expect(items.value).toEqual([])
    expect(total.value).toBe(0)
  })

  it('calls onError on failure', async () => {
    const error = new Error('network fail')
    const apiFn = vi.fn(async () => { throw error })
    const onError = vi.fn()
    const { fetchList } = usePaginatedList({ apiFn, onError })

    await fetchList()
    expect(onError).toHaveBeenCalledWith(error)
  })

  it('falls back to console.error when no onError', async () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const apiFn = vi.fn(async () => { throw new Error('fail') })
    const { fetchList } = usePaginatedList({ apiFn })

    await fetchList()
    expect(spy).toHaveBeenCalled()
    spy.mockRestore()
  })

  it('discards stale responses on rapid page change (requestId)', async () => {
    let resolveFirst!: (v: unknown) => void
    let resolveSecond!: (v: unknown) => void

    const firstCall = new Promise(r => { resolveFirst = r })
    const secondCall = new Promise(r => { resolveSecond = r })

    let callCount = 0
    const apiFn = vi.fn(async () => {
      callCount++
      if (callCount === 1) {
        await firstCall
        return { data: { data: { list: ['stale'], total: 1 } } }
      }
      await secondCall
      return { data: { data: { list: ['fresh'], total: 1 } } }
    })

    const { items, page, fetchList } = usePaginatedList({ apiFn })

    // Fire first fetch (page 1)
    const p1 = fetchList()
    await nextTick()

    // Change page and fire second fetch before first resolves
    page.value = 2
    const p2 = fetchList()
    await nextTick()

    // Now resolve the first (stale) response
    resolveFirst(null)
    await p1
    await nextTick()

    // Items should NOT be stale — they should still be empty (or from a previous state)
    // because the first response was discarded
    expect(items.value).not.toEqual(['stale'])

    // Now resolve the second (fresh) response
    resolveSecond(null)
    await p2
    await nextTick()

    expect(items.value).toEqual(['fresh'])
  })

  it('sets loading to true during fetch', async () => {
    let resolveFetch!: (v: unknown) => void
    const apiFn = vi.fn(async () => {
      await new Promise(r => { resolveFetch = r })
      return { data: { data: { list: [], total: 0 } } }
    })

    const { loading, fetchList } = usePaginatedList({ apiFn })

    expect(loading.value).toBe(false)
    const promise = fetchList()
    expect(loading.value).toBe(true)

    resolveFetch(null)
    await promise
    expect(loading.value).toBe(false)
  })

  it('sets loading to false even when request is stale', async () => {
    let resolveFirst!: (v: unknown) => void
    let resolveSecond!: (v: unknown) => void
    let callCount = 0
    const apiFn = vi.fn(async () => {
      callCount++
      if (callCount === 1) {
        await new Promise(r => { resolveFirst = r })
      } else {
        await new Promise(r => { resolveSecond = r })
      }
      return { data: { data: { list: [], total: 0 } } }
    })

    const { loading, fetchList, page } = usePaginatedList({ apiFn })

    // Start first fetch
    const p1 = fetchList()
    expect(loading.value).toBe(true)

    // Start second fetch (makes first stale)
    page.value = 2
    const p2 = fetchList()

    // Resolve first (stale) — loading stays true because second is still pending
    resolveFirst(null)
    await p1
    await nextTick()
    expect(loading.value).toBe(true)

    // Resolve second — loading goes false
    resolveSecond(null)
    await p2
    await nextTick()
    expect(loading.value).toBe(false)
  })

  it('defaults pageSize to 20', async () => {
    const apiFn = makeApiFn([])
    const { pageSize } = usePaginatedList({ apiFn })
    expect(pageSize.value).toBe(20)
  })

  it('accepts custom pageSize', async () => {
    const apiFn = makeApiFn([])
    const { pageSize } = usePaginatedList({ apiFn, pageSize: 50 })
    expect(pageSize.value).toBe(50)
  })
})
