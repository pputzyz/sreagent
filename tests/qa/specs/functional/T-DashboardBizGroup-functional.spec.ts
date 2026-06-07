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
  const res = await API.post(page, `${API_BASE}/dashboard-biz-groups`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a biz-group by ID, ignoring errors (for cleanup) */
async function cleanupBizGroup(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/dashboard-biz-groups/${id}`)
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
      const res = await API.get(page, `${API_BASE}/dashboard-biz-groups/${groupId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(groupId)
      expect(res.data.name).toContain('biz-group-')
      expect(res.data.description).toBe('CRUD 测试分组')
      await page.screenshot({ path: 'test-results/DBG-1-02-GET验证.png', fullPage: false })
    })

    // ---- 3. 更新分组 ----
    await test.step('更新分组名称和描述', async () => {
      const res = await API.put(page, `${API_BASE}/dashboard-biz-groups/${groupId}`, {
        name: `updated-group-${uid()}`,
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DBG-1-03-更新成功.png', fullPage: false })
    })

    // ---- 4. 验证更新 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/dashboard-biz-groups/${groupId}`)
      expect(res.code).toBe(0)
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/DBG-1-04-更新验证.png', fullPage: false })
    })

    // ---- 5. 删除分组 ----
    await test.step('删除分组', async () => {
      const res = await API.del(page, `${API_BASE}/dashboard-biz-groups/${groupId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DBG-1-05-删除成功.png', fullPage: false })
    })

    // ---- 6. 验证删除 ----
    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/dashboard-biz-groups/${groupId}`)
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
  const tag = uid()

  try {
    // ---- 1. 创建业务分组 ----
    await test.step('创建业务分组', async () => {
      const group = await createBizGroup(page, { name: `bind-test-${tag}` })
      groupId = group.id
      await page.screenshot({ path: 'test-results/DBG-2-01-创建分组.png', fullPage: false })
    })

    // ---- 2. 绑定仪表盘到分组 ----
    await test.step('绑定仪表盘到分组', async () => {
      // 使用一个虚拟 dashboard id 进行绑定测试
      const dashboardId = 1 // 假设默认仪表盘 ID 为 1
      const res = await API.post(page, `${API_BASE}/dashboard-biz-groups/${groupId}/bind`, {
        dashboard_ids: [dashboardId],
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DBG-2-02-绑定成功.png', fullPage: false })
    })

    // ---- 3. 验证绑定 ----
    await test.step('验证绑定关系', async () => {
      const res = await API.get(page, `${API_BASE}/dashboard-biz-groups/${groupId}`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/DBG-2-03-绑定验证.png', fullPage: false })
    })

    // ---- 4. 解绑仪表盘 ----
    await test.step('解绑仪表盘', async () => {
      const dashboardId = 1
      const res = await API.post(page, `${API_BASE}/dashboard-biz-groups/${groupId}/unbind`, {
        dashboard_ids: [dashboardId],
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DBG-2-04-解绑成功.png', fullPage: false })
    })

    // ---- 5. 验证解绑 ----
    await test.step('验证解绑生效', async () => {
      const res = await API.get(page, `${API_BASE}/dashboard-biz-groups/${groupId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DBG-2-05-解绑验证.png', fullPage: false })
    })
  } finally {
    if (groupId) await cleanupBizGroup(page, groupId)
  }
})

// ---------------------------------------------------------------------------
// DBG-3 仪表盘按分组筛选
// ---------------------------------------------------------------------------
test('DBG-3 仪表盘按分组筛选', async ({ authPage: page }) => {
  const groupIds: number[] = []
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

    // ---- 2. 按分组名称筛选 ----
    await test.step('按分组名称筛选', async () => {
      const res = await API.get(page, `${API_BASE}/dashboard-biz-groups?keyword=${uniqueName}&page_size=100`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      expect(list.length).toBeGreaterThanOrEqual(2)
      for (const g of list) {
        expect(g.name).toContain(uniqueName)
      }
      await page.screenshot({ path: 'test-results/DBG-3-02-筛选结果.png', fullPage: false })
    })

    // ---- 3. 获取分组列表 ----
    await test.step('获取分组列表', async () => {
      const res = await API.get(page, `${API_BASE}/dashboard-biz-groups?page=1&page_size=100`)
      expect(res.code).toBe(0)
      expect(res.data.list).toBeDefined()
      expect(Array.isArray(res.data.list)).toBe(true)
      await page.screenshot({ path: 'test-results/DBG-3-03-分组列表.png', fullPage: false })
    })
  } finally {
    for (const id of groupIds) await cleanupBizGroup(page, id)
  }
})
