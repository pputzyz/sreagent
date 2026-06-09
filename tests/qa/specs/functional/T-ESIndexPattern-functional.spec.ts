import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: get the first available datasource ID */
async function getDatasourceId(page: any): Promise<number> {
  const res = await API.get(page, '/api/v1/datasources?page=1&page_size=100')
  expect(res.code).toBe(0)
  const ds = res.data.list?.[0]
  expect(ds).toBeDefined()
  return ds.id
}

/** Helper: create an ES index pattern via API and return the created object */
async function createESIndexPattern(page: any, datasourceId: number, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    datasource_id: datasourceId,
    name: `es-test-${tag}`,
    time_field: '@timestamp',
    note: 'Functional test ES index pattern',
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
  let datasourceId: number

  try {
    await test.step('获取数据源 ID', async () => {
      datasourceId = await getDatasourceId(page)
    })

    await test.step('创建 ES 索引模式', async () => {
      const pattern = await createESIndexPattern(page, datasourceId)
      patternId = pattern.id
      expect(pattern.name).toContain('es-test-')
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
        datasource_id: datasourceId,
        name: `updated-es-${uid()}`,
        note: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ES-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/es-index-patterns/${patternId}`)
      expect(res.code).toBe(0)
      expect(res.data.note).toBe('Updated by functional test')
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
// ES-2: ES 索引模式列表
// ---------------------------------------------------------------------------
test('ES-2 ES索引模式 列表', async ({ authPage: page }) => {
  let patternId: number | null = null
  let datasourceId: number

  try {
    await test.step('获取数据源 ID', async () => {
      datasourceId = await getDatasourceId(page)
    })

    await test.step('创建 ES 索引模式', async () => {
      const pattern = await createESIndexPattern(page, datasourceId)
      patternId = pattern.id
      await page.screenshot({ path: 'test-results/ES-2-01-创建索引模式.png', fullPage: false })
    })

    await test.step('查询列表', async () => {
      const res = await API.get(page, `${API_BASE}/es-index-patterns`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      const list = res.data?.list || res.data || []
      expect(Array.isArray(list)).toBe(true)
      const found = list.find((p: any) => p.id === patternId)
      expect(found).toBeTruthy()
      await page.screenshot({ path: 'test-results/ES-2-02-列表查询.png', fullPage: false })
    })

    await test.step('验证列表结构', async () => {
      const res = await API.get(page, `${API_BASE}/es-index-patterns`)
      expect(res.code).toBe(0)
      const list = res.data?.list || res.data || []
      for (const p of list) {
        expect(p.id).toBeDefined()
        expect(p.name).toBeDefined()
      }
      await page.screenshot({ path: 'test-results/ES-2-03-列表结构.png', fullPage: false })
    })
  } finally {
    if (patternId) await cleanupESIndexPattern(page, patternId)
  }
})

// ---------------------------------------------------------------------------
// ES-3: ES 索引模式详情
// ---------------------------------------------------------------------------
test('ES-3 ES索引模式 详情', async ({ authPage: page }) => {
  let patternId: number | null = null
  let datasourceId: number

  try {
    await test.step('获取数据源 ID', async () => {
      datasourceId = await getDatasourceId(page)
    })

    await test.step('创建 ES 索引模式', async () => {
      const pattern = await createESIndexPattern(page, datasourceId)
      patternId = pattern.id
      await page.screenshot({ path: 'test-results/ES-3-01-创建索引模式.png', fullPage: false })
    })

    await test.step('获取详情', async () => {
      const res = await API.get(page, `${API_BASE}/es-index-patterns/${patternId}`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      expect(res.data.id).toBe(patternId)
      expect(res.data.name).toContain('es-test-')
      await page.screenshot({ path: 'test-results/ES-3-02-详情查看.png', fullPage: false })
    })

    await test.step('验证详情结构', async () => {
      const res = await API.get(page, `${API_BASE}/es-index-patterns/${patternId}`)
      expect(res.code).toBe(0)
      expect(res.data.datasource_id).toBe(datasourceId)
      expect(res.data.time_field).toBe('@timestamp')
      await page.screenshot({ path: 'test-results/ES-3-03-详情结构.png', fullPage: false })
    })
  } finally {
    if (patternId) await cleanupESIndexPattern(page, patternId)
  }
})
