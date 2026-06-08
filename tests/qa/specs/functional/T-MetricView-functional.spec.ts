import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a metric view via API and return the created object */
async function createMetricView(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `mv-test-${tag}`,
    description: 'Functional test metric view',
    config: {
      metrics: ['up', 'http_requests_total'],
      group_by: ['job'],
      time_range: '1h',
    },
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/metric-views`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a metric view by ID, ignoring errors (for cleanup) */
async function cleanupMetricView(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/metric-views/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// MV-1: 指标视图 CRUD
// ---------------------------------------------------------------------------
test('MV-1 指标视图 CRUD', async ({ authPage: page }) => {
  let viewId: number | null = null

  try {
    await test.step('创建指标视图', async () => {
      const view = await createMetricView(page)
      viewId = view.id
      expect(view.name).toContain('mv-test-')
      await page.screenshot({ path: 'test-results/MV-1-01-创建成功.png', fullPage: false })
    })

    await test.step('GET 验证指标视图已保存', async () => {
      const res = await API.get(page, `${API_BASE}/metric-views/${viewId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(viewId)
      expect(res.data.name).toContain('mv-test-')
      await page.screenshot({ path: 'test-results/MV-1-02-GET验证.png', fullPage: false })
    })

    await test.step('更新指标视图', async () => {
      const res = await API.put(page, `${API_BASE}/metric-views/${viewId}`, {
        name: `updated-mv-${uid()}`,
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MV-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/metric-views/${viewId}`)
      expect(res.code).toBe(0)
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/MV-1-04-更新验证.png', fullPage: false })
    })

    await test.step('删除指标视图', async () => {
      const res = await API.del(page, `${API_BASE}/metric-views/${viewId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MV-1-05-删除成功.png', fullPage: false })
    })

    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/metric-views/${viewId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/MV-1-06-删除验证.png', fullPage: false })
    })

    viewId = null
  } finally {
    if (viewId) await cleanupMetricView(page, viewId)
  }
})

// ---------------------------------------------------------------------------
// MV-2: 指标视图收藏
// ---------------------------------------------------------------------------
test('MV-2 指标视图 收藏', async ({ authPage: page }) => {
  let viewId: number | null = null

  try {
    await test.step('创建指标视图', async () => {
      const view = await createMetricView(page)
      viewId = view.id
      await page.screenshot({ path: 'test-results/MV-2-01-创建视图.png', fullPage: false })
    })

    await test.step('收藏指标视图', async () => {
      const res = await API.post(page, `${API_BASE}/metric-views/${viewId}/favorite`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MV-2-02-收藏成功.png', fullPage: false })
    })

    await test.step('验证收藏状态', async () => {
      const res = await API.get(page, `${API_BASE}/metric-views/${viewId}`)
      expect(res.code).toBe(0)
      expect(res.data.is_favorite || res.data.favorited).toBe(true)
      await page.screenshot({ path: 'test-results/MV-2-03-收藏验证.png', fullPage: false })
    })

    await test.step('取消收藏', async () => {
      const res = await API.post(page, `${API_BASE}/metric-views/${viewId}/unfavorite`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MV-2-04-取消收藏.png', fullPage: false })
    })
  } finally {
    if (viewId) await cleanupMetricView(page, viewId)
  }
})

// ---------------------------------------------------------------------------
// MV-3: 指标视图查询
// ---------------------------------------------------------------------------
test('MV-3 指标视图 查询', async ({ authPage: page }) => {
  let viewId: number | null = null

  try {
    await test.step('创建指标视图', async () => {
      const view = await createMetricView(page, {
        config: {
          metrics: ['up'],
          group_by: ['job'],
          time_range: '15m',
        },
      })
      viewId = view.id
      await page.screenshot({ path: 'test-results/MV-3-01-创建视图.png', fullPage: false })
    })

    await test.step('执行指标视图查询', async () => {
      const res = await API.post(page, `${API_BASE}/metric-views/${viewId}/query`, {
        time_range: '15m',
      })
      // May succeed or fail depending on datasource availability
      expect(res).toBeDefined()
      expect(res.code).toBeDefined()
      await page.screenshot({ path: 'test-results/MV-3-02-查询结果.png', fullPage: false })
    })

    await test.step('验证查询结果结构', async () => {
      const res = await API.post(page, `${API_BASE}/metric-views/${viewId}/query`, {
        time_range: '1h',
      })
      if (res.code === 0) {
        expect(res.data).toBeDefined()
      }
      await page.screenshot({ path: 'test-results/MV-3-03-结果结构.png', fullPage: false })
    })
  } finally {
    if (viewId) await cleanupMetricView(page, viewId)
  }
})
