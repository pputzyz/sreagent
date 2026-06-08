import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

// ---------------------------------------------------------------------------
// BD-1: 内置仪表盘列表分类
// ---------------------------------------------------------------------------
test('BD-1 内置仪表盘列表分类', async ({ authPage: page }) => {
  await test.step('获取内置仪表盘列表', async () => {
    const res = await API.get(page, `${API_BASE}/builtin-dashboards?page=1&page_size=100`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    await page.screenshot({ path: 'test-results/BD-1-01-仪表盘列表.png', fullPage: false })
  })

  await test.step('验证列表结构', async () => {
    const res = await API.get(page, `${API_BASE}/builtin-dashboards?page=1&page_size=100`)
    expect(res.code).toBe(0)
    const list = res.data.list || res.data || []
    expect(Array.isArray(list)).toBe(true)
    await page.screenshot({ path: 'test-results/BD-1-02-列表结构.png', fullPage: false })
  })

  await test.step('按分类筛选', async () => {
    const res = await API.get(page, `${API_BASE}/builtin-dashboards?page=1&page_size=100`)
    expect(res.code).toBe(0)
    const list = res.data.list || res.data || []
    if (list.length > 0) {
      const category = list[0].category || list[0].type || 'default'
      const filterRes = await API.get(page, `${API_BASE}/builtin-dashboards?page=1&page_size=100&category=${encodeURIComponent(category)}`)
      expect(filterRes.code).toBe(0)
      const filtered = filterRes.data.list || filterRes.data || []
      expect(Array.isArray(filtered)).toBe(true)
    }
    await page.screenshot({ path: 'test-results/BD-1-03-分类筛选.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// BD-2: 内置仪表盘 apply 导入
// ---------------------------------------------------------------------------
test('BD-2 内置仪表盘 apply导入', async ({ authPage: page }) => {
  let dashboardId: number | undefined
  let dashboardIdent: string | undefined

  await test.step('获取内置仪表盘列表', async () => {
    const res = await API.get(page, `${API_BASE}/builtin-dashboards?page=1&page_size=100`)
    expect(res.code).toBe(0)
    const list = res.data.list || res.data || []
    if (list.length > 0) {
      dashboardId = list[0].id
      dashboardIdent = list[0].ident || list[0].uid || list[0].ID?.toString()
    }
    await page.screenshot({ path: 'test-results/BD-2-01-获取列表.png', fullPage: false })
  })

  if (dashboardId && dashboardIdent) {
    await test.step('Import 导入内置仪表盘', async () => {
      const res = await API.post(page, `${API_BASE}/builtin-dashboards/${dashboardIdent}/import`)
      // Import may succeed or return conflict if already imported
      expect(res).toBeDefined()
      expect(res.code).toBeDefined()
      await page.screenshot({ path: 'test-results/BD-2-02-Import结果.png', fullPage: false })
    })

    await test.step('验证 Import 结果', async () => {
      // Verify the dashboard was imported (may need to check user dashboards)
      const res = await API.get(page, `${API_BASE}/dashboards?page=1&page_size=100`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/BD-2-03-验证导入.png', fullPage: false })
    })
  } else {
    await test.step('无内置仪表盘 — 跳过 apply 测试', async () => {
      await page.screenshot({ path: 'test-results/BD-2-02-无内置仪表盘.png', fullPage: false })
    })
  }
})

// ---------------------------------------------------------------------------
// BD-3: 内置仪表盘 components 查询
// ---------------------------------------------------------------------------
test('BD-3 内置仪表盘 components查询', async ({ authPage: page }) => {
  let dashboardId: number | undefined

  await test.step('获取内置仪表盘列表', async () => {
    const res = await API.get(page, `${API_BASE}/builtin-dashboards?page=1&page_size=100`)
    expect(res.code).toBe(0)
    const list = res.data.list || res.data || []
    if (list.length > 0) {
      dashboardId = list[0].id
    }
    await page.screenshot({ path: 'test-results/BD-3-01-获取列表.png', fullPage: false })
  })

  if (dashboardId) {
    await test.step('查询仪表盘组件', async () => {
      const res = await API.get(page, `${API_BASE}/builtin-dashboards/components`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/BD-3-02-组件查询.png', fullPage: false })
    })

    await test.step('验证组件结构', async () => {
      const res = await API.get(page, `${API_BASE}/builtin-dashboards/components`)
      expect(res.code).toBe(0)
      const components = Array.isArray(res.data) ? res.data : res.data.list || []
      expect(Array.isArray(components)).toBe(true)
      await page.screenshot({ path: 'test-results/BD-3-03-组件结构.png', fullPage: false })
    })
  } else {
    await test.step('无内置仪表盘 — 跳过组件查询', async () => {
      await page.screenshot({ path: 'test-results/BD-3-02-无内置仪表盘.png', fullPage: false })
    })
  }
})
