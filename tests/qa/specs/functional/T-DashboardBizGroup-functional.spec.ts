import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a dashboard biz-group and return the created object */
async function createBizGroup(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `biz-group-${tag}`,
    description: `Functional test biz group ${tag}`,
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/biz-groups`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a biz-group by ID, ignoring errors (for cleanup) */
async function cleanupBizGroup(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/biz-groups/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// DBG-1 仪表盘业务分组 CRUD
// ---------------------------------------------------------------------------
test('DBG-1 仪表盘业务分组CRUD', async ({ authPage: page }) => {
  let groupId: number | null = null

  try {
    // ---- 1. 创建业务分组 ----
    await test.step('创建业务分组', async () => {
      const group = await createBizGroup(page, {
        description: 'CRUD 测试分组',
      })
      groupId = group.id
      expect(group.name).toContain('biz-group-')
      expect(group.description).toBe('CRUD 测试分组')
      await page.screenshot({ path: 'test-results/DBG-1-01-创建成功.png', fullPage: false })
    })

    // ---- 2. GET 验证 ----
    await test.step('GET 验证分组已保存', async () => {
      const res = await API.get(page, `${API_BASE}/biz-groups/${groupId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(groupId)
      expect(res.data.name).toContain('biz-group-')
      expect(res.data.description).toBe('CRUD 测试分组')
      await page.screenshot({ path: 'test-results/DBG-1-02-GET验证.png', fullPage: false })
    })

    // ---- 3. 更新分组 ----
    await test.step('更新分组名称和描述', async () => {
      const res = await API.put(page, `${API_BASE}/biz-groups/${groupId}`, {
        name: `updated-group-${uid()}`,
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DBG-1-03-更新成功.png', fullPage: false })
    })

    // ---- 4. 验证更新 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/biz-groups/${groupId}`)
      expect(res.code).toBe(0)
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/DBG-1-04-更新验证.png', fullPage: false })
    })

    // ---- 5. 删除分组 ----
    await test.step('删除分组', async () => {
      const res = await API.del(page, `${API_BASE}/biz-groups/${groupId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DBG-1-05-删除成功.png', fullPage: false })
    })

    // ---- 6. 验证删除 ----
    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/biz-groups/${groupId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/DBG-1-06-删除验证.png', fullPage: false })
    })

    groupId = null
  } finally {
    if (groupId) await cleanupBizGroup(page, groupId)
  }
})

// ---------------------------------------------------------------------------
// DBG-2 仪表盘绑定/解绑分组
// ---------------------------------------------------------------------------
test('DBG-2 仪表盘绑定/解绑分组', async ({ authPage: page }) => {
  let groupId: number | null = null
  let dashboardId: number | null = null
  const tag = uid()

  try {
    // ---- 1. 创建业务分组 ----
    await test.step('创建业务分组', async () => {
      const group = await createBizGroup(page, { name: `bind-test-${tag}` })
      groupId = group.id
      await page.screenshot({ path: 'test-results/DBG-2-01-创建分组.png', fullPage: false })
    })

    // ---- 2. 创建仪表盘 ----
    await test.step('创建仪表盘', async () => {
      const res = await API.post(page, `${API_BASE}/dashboards`, {
        name: `dashboard-for-bind-${tag}`,
        description: 'Dashboard for bind test',
        is_public: true,
      })
      expect(res.code).toBe(0)
      dashboardId = res.data.id || res.data.ID
      expect(dashboardId).toBeGreaterThan(0)
      await page.screenshot({ path: 'test-results/DBG-2-02-创建仪表盘.png', fullPage: false })
    })

    // ---- 3. 绑定仪表盘到分组 ----
    await test.step('绑定仪表盘到分组', async () => {
      const res = await API.post(page, `${API_BASE}/dashboards/${dashboardId}/biz-groups`, {
        biz_group_id: groupId,
        perm_flag: 'ro',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DBG-2-03-绑定成功.png', fullPage: false })
    })

    // ---- 4. 验证绑定 ----
    await test.step('验证绑定关系', async () => {
      const res = await API.get(page, `${API_BASE}/dashboards/${dashboardId}/biz-groups`)
      expect(res.code).toBe(0)
      const bindings = res.data?.list || res.data || []
      const found = Array.isArray(bindings) && bindings.some((b: any) => (b.biz_group_id || b.biz_group_ID) === groupId)
      expect(found).toBe(true)
      await page.screenshot({ path: 'test-results/DBG-2-04-绑定验证.png', fullPage: false })
    })

    // ---- 5. 解绑仪表盘 ----
    await test.step('解绑仪表盘', async () => {
      const res = await API.del(page, `${API_BASE}/dashboards/${dashboardId}/biz-groups/${groupId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DBG-2-05-解绑成功.png', fullPage: false })
    })

    // ---- 6. 验证解绑 ----
    await test.step('验证解绑生效', async () => {
      const res = await API.get(page, `${API_BASE}/dashboards/${dashboardId}/biz-groups`)
      expect(res.code).toBe(0)
      const bindings = res.data?.list || res.data || []
      const found = Array.isArray(bindings) && bindings.some((b: any) => (b.biz_group_id || b.biz_group_ID) === groupId)
      expect(found).toBe(false)
      await page.screenshot({ path: 'test-results/DBG-2-06-解绑验证.png', fullPage: false })
    })
  } finally {
    if (dashboardId) {
      try { await API.del(page, `${API_BASE}/dashboards/${dashboardId}`) } catch { /* ignore */ }
    }
    if (groupId) await cleanupBizGroup(page, groupId)
  }
})

// ---------------------------------------------------------------------------
// DBG-3 仪表盘按分组筛选
// ---------------------------------------------------------------------------
test('DBG-3 仪表盘按分组筛选', async ({ authPage: page }) => {
  const groupIds: number[] = []
  let dashboardId: number | null = null
  const uniqueName = `filter-test-${uid()}`

  try {
    // ---- 1. 创建多个分组 ----
    await test.step('创建多个分组', async () => {
      for (let i = 0; i < 2; i++) {
        const group = await createBizGroup(page, { name: `${uniqueName}-${i}` })
        groupIds.push(group.id)
      }
      expect(groupIds.length).toBe(2)
      await page.screenshot({ path: 'test-results/DBG-3-01-创建分组.png', fullPage: false })
    })

    // ---- 2. 创建仪表盘并绑定到分组 ----
    await test.step('创建仪表盘并绑定到分组', async () => {
      const res = await API.post(page, `${API_BASE}/dashboards`, {
        name: `dashboard-for-filter-${uniqueName}`,
        description: 'Dashboard for filter test',
        is_public: true,
      })
      expect(res.code).toBe(0)
      dashboardId = res.data.id || res.data.ID

      // 绑定到第一个分组
      const bindRes = await API.post(page, `${API_BASE}/dashboards/${dashboardId}/biz-groups`, {
        biz_group_id: groupIds[0],
        perm_flag: 'ro',
      })
      expect(bindRes.code).toBe(0)
      await page.screenshot({ path: 'test-results/DBG-3-02-绑定分组.png', fullPage: false })
    })

    // ---- 3. 获取分组列表 ----
    await test.step('获取分组列表', async () => {
      const res = await API.get(page, `${API_BASE}/biz-groups?page=1&page_size=100`)
      expect(res.code).toBe(0)
      const list = res.data?.list || res.data || []
      expect(Array.isArray(list)).toBe(true)
      // 验证我们创建的分组在列表中
      const foundNames = list.filter((g: any) => g.name?.startsWith(uniqueName))
      expect(foundNames.length).toBeGreaterThanOrEqual(2)
      await page.screenshot({ path: 'test-results/DBG-3-03-分组列表.png', fullPage: false })
    })

    // ---- 4. 获取仪表盘关联的分组 ----
    await test.step('获取仪表盘关联的分组', async () => {
      const res = await API.get(page, `${API_BASE}/dashboards/${dashboardId}/biz-groups`)
      expect(res.code).toBe(0)
      const bindings = res.data?.list || res.data || []
      expect(Array.isArray(bindings)).toBe(true)
      expect(bindings.length).toBeGreaterThanOrEqual(1)
      const found = bindings.some((b: any) => (b.biz_group_id || b.biz_group_ID) === groupIds[0])
      expect(found).toBe(true)
      await page.screenshot({ path: 'test-results/DBG-3-04-关联分组.png', fullPage: false })
    })

    // ---- 5. 获取分组树形结构 ----
    await test.step('获取分组树形结构', async () => {
      const res = await API.get(page, `${API_BASE}/biz-groups/tree`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/DBG-3-05-树形结构.png', fullPage: false })
    })
  } finally {
    if (dashboardId) {
      try { await API.del(page, `${API_BASE}/dashboards/${dashboardId}`) } catch { /* ignore */ }
    }
    for (const id of groupIds) await cleanupBizGroup(page, id)
  }
})
