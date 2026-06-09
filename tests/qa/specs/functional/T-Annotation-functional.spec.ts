import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create an annotation via API and return the created object */
async function createAnnotation(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const now = new Date()
  const payload = {
    title: `annotation-${tag}`,
    content: 'Functional test annotation',
    start_time: now.toISOString(),
    end_time: new Date(now.getTime() + 3600000).toISOString(),
    tags: { test: tag },
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/annotations`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete an annotation by ID, ignoring errors (for cleanup) */
async function cleanupAnnotation(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/annotations/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// AN-1: 标注 CRUD
// ---------------------------------------------------------------------------
test('AN-1 标注 CRUD', async ({ authPage: page }) => {
  let annotationId: number | null = null

  try {
    await test.step('创建标注', async () => {
      const annotation = await createAnnotation(page)
      annotationId = annotation.id
      expect(annotation.title).toContain('annotation-')
      await page.screenshot({ path: 'test-results/AN-1-01-创建成功.png', fullPage: false })
    })

    await test.step('GET 验证标注已保存', async () => {
      const res = await API.get(page, `${API_BASE}/annotations/${annotationId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(annotationId)
      expect(res.data.title).toContain('annotation-')
      await page.screenshot({ path: 'test-results/AN-1-02-GET验证.png', fullPage: false })
    })

    await test.step('更新标注', async () => {
      const res = await API.put(page, `${API_BASE}/annotations/${annotationId}`, {
        title: `updated-annotation-${uid()}`,
        content: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/AN-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/annotations/${annotationId}`)
      expect(res.code).toBe(0)
      expect(res.data.content).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/AN-1-04-更新验证.png', fullPage: false })
    })

    await test.step('删除标注', async () => {
      const res = await API.del(page, `${API_BASE}/annotations/${annotationId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/AN-1-05-删除成功.png', fullPage: false })
    })

    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/annotations/${annotationId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/AN-1-06-删除验证.png', fullPage: false })
    })

    annotationId = null
  } finally {
    if (annotationId) await cleanupAnnotation(page, annotationId)
  }
})

// ---------------------------------------------------------------------------
// AN-2: 标注 batch 批量创建
// ---------------------------------------------------------------------------
test('AN-2 标注 batch批量创建', async ({ authPage: page }) => {
  const annotationIds: number[] = []

  try {
    await test.step('批量创建3个标注', async () => {
      const now = new Date()
      for (let i = 0; i < 3; i++) {
        const tag = uid()
        const startTime = new Date(now.getTime() + i * 3600000)
        const endTime = new Date(startTime.getTime() + 3600000)
        const res = await API.post(page, `${API_BASE}/annotations`, {
          title: `batch-annotation-${tag}`,
          content: `Batch annotation ${i}`,
          start_time: startTime.toISOString(),
          end_time: endTime.toISOString(),
          tags: ['batch-test'],
        })
        expect(res.code).toBe(0)
        annotationIds.push(res.data.id)
      }
      expect(annotationIds.length).toBe(3)
      await page.screenshot({ path: 'test-results/AN-2-01-批量创建成功.png', fullPage: false })
    })

    await test.step('验证批量创建的标注均存在', async () => {
      for (const id of annotationIds) {
        const res = await API.get(page, `${API_BASE}/annotations/${id}`)
        expect(res.code).toBe(0)
        expect(res.data.id).toBe(id)
      }
      await page.screenshot({ path: 'test-results/AN-2-02-批量验证.png', fullPage: false })
    })

    await test.step('批量删除', async () => {
      for (const id of annotationIds) {
        const res = await API.del(page, `${API_BASE}/annotations/${id}`)
        expect(res.code).toBe(0)
      }
      await page.screenshot({ path: 'test-results/AN-2-03-批量删除成功.png', fullPage: false })
    })

    annotationIds.length = 0
  } finally {
    for (const id of annotationIds) await cleanupAnnotation(page, id)
  }
})

// ---------------------------------------------------------------------------
// AN-3: 标注时间范围查询
// ---------------------------------------------------------------------------
test('AN-3 标注时间范围查询', async ({ authPage: page }) => {
  const annotationIds: number[] = []

  try {
    await test.step('创建不同时间的标注', async () => {
      const now = new Date()
      for (let i = 0; i < 2; i++) {
        const tag = uid()
        const startTime = new Date(now.getTime() - (2 - i) * 3600000)
        const endTime = new Date(startTime.getTime() + 1800000)
        const res = await API.post(page, `${API_BASE}/annotations`, {
          title: `time-range-${tag}`,
          content: `Time range test ${i}`,
          start_time: startTime.toISOString(),
          end_time: endTime.toISOString(),
          tags: ['time-range-test'],
        })
        expect(res.code).toBe(0)
        annotationIds.push(res.data.id)
      }
      await page.screenshot({ path: 'test-results/AN-3-01-创建时间标注.png', fullPage: false })
    })

    await test.step('按时间范围查询', async () => {
      const now = new Date()
      const threeHoursAgo = new Date(now.getTime() - 3 * 3600000)
      const res = await API.get(page, `${API_BASE}/annotations?start=${encodeURIComponent(threeHoursAgo.toISOString())}&end=${encodeURIComponent(now.toISOString())}`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/AN-3-02-时间范围查询.png', fullPage: false })
    })

    await test.step('验证查询结果在时间范围内', async () => {
      const now = new Date()
      const threeHoursAgo = new Date(now.getTime() - 3 * 3600000)
      const res = await API.get(page, `${API_BASE}/annotations?start=${encodeURIComponent(threeHoursAgo.toISOString())}&end=${encodeURIComponent(now.toISOString())}`)
      expect(res.code).toBe(0)
      const list = res.data.list || res.data || []
      expect(Array.isArray(list)).toBe(true)
      await page.screenshot({ path: 'test-results/AN-3-03-范围验证.png', fullPage: false })
    })
  } finally {
    for (const id of annotationIds) await cleanupAnnotation(page, id)
  }
})
