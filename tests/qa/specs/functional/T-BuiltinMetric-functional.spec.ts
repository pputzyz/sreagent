import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a builtin metric via API and return the created object */
async function createBuiltinMetric(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `bm-test-${tag}`,
    expression: `sum(rate(test_metric_${tag}[5m]))`,
    description: 'Functional test builtin metric',
    unit: 'req/s',
    metric_type: 'counter',
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/builtin-metrics`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a builtin metric by ID, ignoring errors (for cleanup) */
async function cleanupBuiltinMetric(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/builtin-metrics/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// BM-1: 内置指标 CRUD
// ---------------------------------------------------------------------------
test('BM-1 内置指标 CRUD', async ({ authPage: page }) => {
  let metricId: number | null = null

  try {
    await test.step('创建内置指标', async () => {
      const metric = await createBuiltinMetric(page)
      metricId = metric.id
      expect(metric.name).toContain('bm-test-')
      await page.screenshot({ path: 'test-results/BM-1-01-创建成功.png', fullPage: false })
    })

    await test.step('GET 验证内置指标已保存', async () => {
      const res = await API.get(page, `${API_BASE}/builtin-metrics/${metricId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(metricId)
      expect(res.data.unit).toBe('req/s')
      await page.screenshot({ path: 'test-results/BM-1-02-GET验证.png', fullPage: false })
    })

    await test.step('更新内置指标', async () => {
      const res = await API.put(page, `${API_BASE}/builtin-metrics/${metricId}`, {
        name: `bm-test-updated-${uid()}`,
        expression: 'sum(rate(test_updated[5m]))',
        description: 'Updated by functional test',
        unit: 'ops/s',
        metric_type: 'counter',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/BM-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/builtin-metrics/${metricId}`)
      expect(res.code).toBe(0)
      expect(res.data.description).toBe('Updated by functional test')
      expect(res.data.unit).toBe('ops/s')
      await page.screenshot({ path: 'test-results/BM-1-04-更新验证.png', fullPage: false })
    })

    await test.step('删除内置指标', async () => {
      const res = await API.del(page, `${API_BASE}/builtin-metrics/${metricId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/BM-1-05-删除成功.png', fullPage: false })
    })

    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/builtin-metrics/${metricId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/BM-1-06-删除验证.png', fullPage: false })
    })

    metricId = null
  } finally {
    if (metricId) await cleanupBuiltinMetric(page, metricId)
  }
})

// ---------------------------------------------------------------------------
// BM-2: 内置指标 types
// ---------------------------------------------------------------------------
test('BM-2 内置指标 types', async ({ authPage: page }) => {
  await test.step('获取指标类型列表', async () => {
    const res = await API.get(page, `${API_BASE}/builtin-metrics/types`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    await page.screenshot({ path: 'test-results/BM-2-01-类型列表.png', fullPage: false })
  })

  await test.step('验证类型列表结构', async () => {
    const res = await API.get(page, `${API_BASE}/builtin-metrics/types`)
    expect(res.code).toBe(0)
    const types = Array.isArray(res.data) ? res.data : res.data.list || []
    expect(Array.isArray(types)).toBe(true)
    await page.screenshot({ path: 'test-results/BM-2-02-类型结构.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// BM-3: 内置指标 collectors
// ---------------------------------------------------------------------------
test('BM-3 内置指标 collectors', async ({ authPage: page }) => {
  await test.step('获取采集器列表', async () => {
    const res = await API.get(page, `${API_BASE}/builtin-metrics/collectors`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    await page.screenshot({ path: 'test-results/BM-3-01-采集器列表.png', fullPage: false })
  })

  await test.step('验证采集器列表结构', async () => {
    const res = await API.get(page, `${API_BASE}/builtin-metrics/collectors`)
    expect(res.code).toBe(0)
    const collectors = Array.isArray(res.data) ? res.data : res.data.list || []
    expect(Array.isArray(collectors)).toBe(true)
    await page.screenshot({ path: 'test-results/BM-3-02-采集器结构.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// BM-4: 内置指标批量创建
// ---------------------------------------------------------------------------
test('BM-4 内置指标批量创建', async ({ authPage: page }) => {
  const metricIds: number[] = []

  try {
    await test.step('批量创建3个内置指标', async () => {
      for (let i = 0; i < 3; i++) {
        const metric = await createBuiltinMetric(page, {
          metric_type: i === 0 ? 'counter' : i === 1 ? 'gauge' : 'histogram',
        })
        metricIds.push(metric.id)
      }
      expect(metricIds.length).toBe(3)
      await page.screenshot({ path: 'test-results/BM-4-01-批量创建成功.png', fullPage: false })
    })

    await test.step('验证批量创建的指标均存在', async () => {
      for (const id of metricIds) {
        const res = await API.get(page, `${API_BASE}/builtin-metrics/${id}`)
        expect(res.code).toBe(0)
        expect(res.data.id).toBe(id)
      }
      await page.screenshot({ path: 'test-results/BM-4-02-批量验证.png', fullPage: false })
    })
  } finally {
    for (const id of metricIds) await cleanupBuiltinMetric(page, id)
  }
})
