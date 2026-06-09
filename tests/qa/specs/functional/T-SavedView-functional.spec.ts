import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a saved view via API and return the created object */
async function createSavedView(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `sv-test-${tag}`,
    description: 'Functional test saved view',
    tab: 'metrics',
    expression: 'up',
    query_config: JSON.stringify({ filters: { severity: 'critical' }, sort: 'time_desc' }),
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/saved-views`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a saved view by ID, ignoring errors (for cleanup) */
async function cleanupSavedView(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/saved-views/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// SV-1: 快捷视图 CRUD
// ---------------------------------------------------------------------------
test('SV-1 快捷视图 CRUD', async ({ authPage: page }) => {
  let viewId: number | null = null

  try {
    await test.step('创建快捷视图', async () => {
      const view = await createSavedView(page)
      viewId = view.id
      expect(view.name).toContain('sv-test-')
      await page.screenshot({ path: 'test-results/SV-1-01-创建成功.png', fullPage: false })
    })

    await test.step('GET 验证快捷视图已保存', async () => {
      const res = await API.get(page, `${API_BASE}/saved-views/${viewId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(viewId)
      expect(res.data.name).toContain('sv-test-')
      await page.screenshot({ path: 'test-results/SV-1-02-GET验证.png', fullPage: false })
    })

    await test.step('更新快捷视图', async () => {
      const res = await API.put(page, `${API_BASE}/saved-views/${viewId}`, {
        name: `updated-sv-${uid()}`,
        description: 'Updated by functional test',
        tab: 'metrics',
        expression: 'up',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/SV-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/saved-views/${viewId}`)
      expect(res.code).toBe(0)
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/SV-1-04-更新验证.png', fullPage: false })
    })

    await test.step('删除快捷视图', async () => {
      const res = await API.del(page, `${API_BASE}/saved-views/${viewId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/SV-1-05-删除成功.png', fullPage: false })
    })

    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/saved-views/${viewId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/SV-1-06-删除验证.png', fullPage: false })
    })

    viewId = null
  } finally {
    if (viewId) await cleanupSavedView(page, viewId)
  }
})

// ---------------------------------------------------------------------------
// SV-2: 快捷视图 copy 复制
// ---------------------------------------------------------------------------
test('SV-2 快捷视图 copy复制', async ({ authPage: page }) => {
  let viewId: number | null = null
  let copiedId: number | null = null

  try {
    await test.step('创建原始快捷视图', async () => {
      const view = await createSavedView(page, {
        query_config: JSON.stringify({ filters: { severity: 'warning', env: 'staging' } }),
      })
      viewId = view.id
      await page.screenshot({ path: 'test-results/SV-2-01-创建原始视图.png', fullPage: false })
    })

    await test.step('复制快捷视图', async () => {
      const res = await API.post(page, `${API_BASE}/saved-views/${viewId}/copy`, {
        name: `copied-sv-${uid()}`,
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      copiedId = res.data.id
      await page.screenshot({ path: 'test-results/SV-2-02-复制成功.png', fullPage: false })
    })

    await test.step('验证复制的视图', async () => {
      const res = await API.get(page, `${API_BASE}/saved-views/${copiedId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(copiedId)
      await page.screenshot({ path: 'test-results/SV-2-03-复制验证.png', fullPage: false })
    })
  } finally {
    if (viewId) await cleanupSavedView(page, viewId)
    if (copiedId) await cleanupSavedView(page, copiedId)
  }
})

// ---------------------------------------------------------------------------
// SV-3: 快捷视图列表
// ---------------------------------------------------------------------------
test('SV-3 快捷视图 列表', async ({ authPage: page }) => {
  let viewId: number | null = null

  try {
    await test.step('创建快捷视图', async () => {
      const view = await createSavedView(page)
      viewId = view.id
      await page.screenshot({ path: 'test-results/SV-3-01-创建视图.png', fullPage: false })
    })

    await test.step('验证视图在列表中', async () => {
      const res = await API.get(page, `${API_BASE}/saved-views`)
      expect(res.code).toBe(0)
      const list = res.data?.list || res.data || []
      const found = list.find((v: any) => v.id === viewId)
      expect(found).toBeTruthy()
      await page.screenshot({ path: 'test-results/SV-3-02-列表验证.png', fullPage: false })
    })
  } finally {
    if (viewId) await cleanupSavedView(page, viewId)
  }
})
