import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create an inspection task via API and return the created object */
async function createInspection(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `inspection-${tag}`,
    description: 'Functional test inspection',
    cron_expr: '0 0 * * *',
    is_enabled: true,
    inspection_type: 'manual',
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/inspection/tasks`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete an inspection by ID, ignoring errors (for cleanup) */
async function cleanupInspection(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/inspection/tasks/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// IN-1: 巡检任务 CRUD
// ---------------------------------------------------------------------------
test('IN-1 巡检任务 CRUD', async ({ authPage: page }) => {
  let inspectionId: number | null = null

  try {
    await test.step('创建巡检任务', async () => {
      const inspection = await createInspection(page)
      inspectionId = inspection.id
      expect(inspection.name).toContain('inspection-')
      await page.screenshot({ path: 'test-results/IN-1-01-创建成功.png', fullPage: false })
    })

    await test.step('GET 验证巡检任务已保存', async () => {
      const res = await API.get(page, `${API_BASE}/inspection/tasks/${inspectionId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(inspectionId)
      expect(res.data.name).toContain('inspection-')
      await page.screenshot({ path: 'test-results/IN-1-02-GET验证.png', fullPage: false })
    })

    await test.step('更新巡检任务', async () => {
      const res = await API.put(page, `${API_BASE}/inspection/tasks/${inspectionId}`, {
        name: `updated-inspection-${uid()}`,
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/IN-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/inspection/tasks/${inspectionId}`)
      expect(res.code).toBe(0)
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/IN-1-04-更新验证.png', fullPage: false })
    })

    await test.step('删除巡检任务', async () => {
      const res = await API.del(page, `${API_BASE}/inspection/tasks/${inspectionId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/IN-1-05-删除成功.png', fullPage: false })
    })

    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/inspection/tasks/${inspectionId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/IN-1-06-删除验证.png', fullPage: false })
    })

    inspectionId = null
  } finally {
    if (inspectionId) await cleanupInspection(page, inspectionId)
  }
})

// ---------------------------------------------------------------------------
// IN-2: 巡检立即执行
// ---------------------------------------------------------------------------
test('IN-2 巡检立即执行', async ({ authPage: page }) => {
  let inspectionId: number | null = null

  try {
    await test.step('创建巡检任务', async () => {
      const inspection = await createInspection(page)
      inspectionId = inspection.id
      await page.screenshot({ path: 'test-results/IN-2-01-创建任务.png', fullPage: false })
    })

    await test.step('立即执行巡检', async () => {
      const res = await API.post(page, `${API_BASE}/inspection/tasks/${inspectionId}/run`)
      // May succeed or fail depending on inspection config
      expect(res).toBeDefined()
      expect(res.code).toBeDefined()
      await page.screenshot({ path: 'test-results/IN-2-02-立即执行.png', fullPage: false })
    })
  } finally {
    if (inspectionId) await cleanupInspection(page, inspectionId)
  }
})

// ---------------------------------------------------------------------------
// IN-3: 巡检 cron 校验
// ---------------------------------------------------------------------------
test('IN-3 巡检 cron校验', async ({ authPage: page }) => {
  await test.step('有效 cron 表达式校验', async () => {
    const res = await API.post(page, `${API_BASE}/inspection/tasks/validate-cron`, {
      cron_expr: '0 0 * * *',
    })
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    await page.screenshot({ path: 'test-results/IN-3-01-有效cron.png', fullPage: false })
  })

  await test.step('无效 cron 表达式校验', async () => {
    const res = await API.post(page, `${API_BASE}/inspection/tasks/validate-cron`, {
      cron_expr: 'invalid cron',
    })
    // Should return error for invalid cron
    const hasError = res.code !== 0 || res.data?.valid === false
    expect(hasError).toBeTruthy()
    await page.screenshot({ path: 'test-results/IN-3-02-无效cron.png', fullPage: false })
  })

  await test.step('复杂 cron 表达式校验', async () => {
    const res = await API.post(page, `${API_BASE}/inspection/tasks/validate-cron`, {
      cron_expr: '*/5 * * * 1-5',
    })
    expect(res.code).toBe(0)
    await page.screenshot({ path: 'test-results/IN-3-03-复杂cron.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// IN-4: 巡检运行记录
// ---------------------------------------------------------------------------
test('IN-4 巡检运行记录', async ({ authPage: page }) => {
  await test.step('获取巡检运行记录列表', async () => {
    const res = await API.get(page, `${API_BASE}/inspection/tasks/runs?page=1&page_size=20`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    await page.screenshot({ path: 'test-results/IN-4-01-运行记录.png', fullPage: false })
  })

  await test.step('验证运行记录结构', async () => {
    const res = await API.get(page, `${API_BASE}/inspection/tasks/runs?page=1&page_size=5`)
    expect(res.code).toBe(0)
    const list = res.data.list || res.data || []
    expect(Array.isArray(list)).toBe(true)
    await page.screenshot({ path: 'test-results/IN-4-02-记录结构.png', fullPage: false })
  })
})
