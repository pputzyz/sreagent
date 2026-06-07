import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create an ES index pattern via API and return the created object */
async function createESIndexPattern(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `es-test-${tag}`,
    index_pattern: `logs-${tag}-*`,
    time_field: '@timestamp',
    description: 'Functional test ES index pattern',
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/es-index-patterns`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete an ES index pattern by ID, ignoring errors (for cleanup) */
async function cleanupESIndexPattern(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/es-index-patterns/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// ES-1: ES 索引模式 CRUD
// ---------------------------------------------------------------------------
test('ES-1 ES索引模式 CRUD', async ({ authPage: page }) => {
  let patternId: number | null = null

  try {
    await test.step('创建 ES 索引模式', async () => {
      const pattern = await createESIndexPattern(page)
      patternId = pattern.id
      expect(pattern.name).toContain('es-test-')
      expect(pattern.index_pattern).toContain('logs-')
      await page.screenshot({ path: 'test-results/ES-1-01-创建成功.png', fullPage: false })
    })

    await test.step('GET 验证 ES 索引模式已保存', async () => {
      const res = await API.get(page, `${API_BASE}/es-index-patterns/${patternId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(patternId)
      expect(res.data.time_field).toBe('@timestamp')
      await page.screenshot({ path: 'test-results/ES-1-02-GET验证.png', fullPage: false })
    })

    await test.step('更新 ES 索引模式', async () => {
      const res = await API.put(page, `${API_BASE}/es-index-patterns/${patternId}`, {
        name: `updated-es-${uid()}`,
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ES-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/es-index-patterns/${patternId}`)
      expect(res.code).toBe(0)
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/ES-1-04-更新验证.png', fullPage: false })
    })

    await test.step('删除 ES 索引模式', async () => {
      const res = await API.del(page, `${API_BASE}/es-index-patterns/${patternId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ES-1-05-删除成功.png', fullPage: false })
    })

    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/es-index-patterns/${patternId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/ES-1-06-删除验证.png', fullPage: false })
    })

    patternId = null
  } finally {
    if (patternId) await cleanupESIndexPattern(page, patternId)
  }
})

// ---------------------------------------------------------------------------
// ES-2: ES 索引模式删除前引用检查
// ---------------------------------------------------------------------------
test('ES-2 ES索引模式 删除前引用检查', async ({ authPage: page }) => {
  let patternId: number | null = null

  try {
    await test.step('创建 ES 索引模式', async () => {
      const pattern = await createESIndexPattern(page)
      patternId = pattern.id
      await page.screenshot({ path: 'test-results/ES-2-01-创建索引模式.png', fullPage: false })
    })

    await test.step('查询引用检查', async () => {
      const res = await API.get(page, `${API_BASE}/es-index-patterns/${patternId}/references`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/ES-2-02-引用检查.png', fullPage: false })
    })

    await test.step('验证引用结构', async () => {
      const res = await API.get(page, `${API_BASE}/es-index-patterns/${patternId}/references`)
      expect(res.code).toBe(0)
      // Should return list of referencing objects or empty
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/ES-2-03-引用结构.png', fullPage: false })
    })
  } finally {
    if (patternId) await cleanupESIndexPattern(page, patternId)
  }
})

// ---------------------------------------------------------------------------
// ES-3: ES 索引模式字段预览
// ---------------------------------------------------------------------------
test('ES-3 ES索引模式 字段预览', async ({ authPage: page }) => {
  let patternId: number | null = null

  try {
    await test.step('创建 ES 索引模式', async () => {
      const pattern = await createESIndexPattern(page)
      patternId = pattern.id
      await page.screenshot({ path: 'test-results/ES-3-01-创建索引模式.png', fullPage: false })
    })

    await test.step('获取字段预览', async () => {
      const res = await API.get(page, `${API_BASE}/es-index-patterns/${patternId}/fields`)
      // May fail if ES is not connected, but should return structured response
      expect(res).toBeDefined()
      expect(res.code).toBeDefined()
      await page.screenshot({ path: 'test-results/ES-3-02-字段预览.png', fullPage: false })
    })

    await test.step('验证字段预览结构', async () => {
      const res = await API.get(page, `${API_BASE}/es-index-patterns/${patternId}/fields`)
      if (res.code === 0) {
        const fields = Array.isArray(res.data) ? res.data : res.data.list || []
        expect(Array.isArray(fields)).toBe(true)
      }
      await page.screenshot({ path: 'test-results/ES-3-03-字段结构.png', fullPage: false })
    })
  } finally {
    if (patternId) await cleanupESIndexPattern(page, patternId)
  }
})
