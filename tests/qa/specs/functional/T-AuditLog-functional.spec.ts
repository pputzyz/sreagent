import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

// ---------------------------------------------------------------------------
// AL-1 审计日志列表查询
// ---------------------------------------------------------------------------
test('AL-1 审计日志列表查询', async ({ authPage: page }) => {
  // ---- 1. 获取审计日志列表 ----
  await test.step('获取审计日志列表', async () => {
    const res = await API.get(page, `${API_BASE}/audit-logs?page=1&page_size=10`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    expect(res.data.list).toBeDefined()
    expect(Array.isArray(res.data.list)).toBe(true)
    await page.screenshot({ path: 'test-results/AL-1-01-列表查询.png', fullPage: false })
  })

  // ---- 2. 验证日志条目结构 ----
  await test.step('验证日志条目结构', async () => {
    const res = await API.get(page, `${API_BASE}/audit-logs?page=1&page_size=5`)
    expect(res.code).toBe(0)
    const list = res.data.list || []
    if (list.length > 0) {
      const log = list[0]
      // 审计日志应包含基本字段
      expect(log).toHaveProperty('id')
      expect(log).toHaveProperty('created_at')
      expect(typeof log.id).toBe('number')
    }
    await page.screenshot({ path: 'test-results/AL-1-02-条目结构.png', fullPage: false })
  })

  // ---- 3. 验证分页参数 ----
  await test.step('验证分页参数', async () => {
    const res = await API.get(page, `${API_BASE}/audit-logs?page=1&page_size=2`)
    expect(res.code).toBe(0)
    expect(res.data.list.length).toBeLessThanOrEqual(2)
    expect(res.data).toHaveProperty('total')
    expect(typeof res.data.total).toBe('number')
    await page.screenshot({ path: 'test-results/AL-1-03-分页验证.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// AL-2 审计日志筛选
// ---------------------------------------------------------------------------
test('AL-2 审计日志筛选', async ({ authPage: page }) => {
  // ---- 1. 按时间范围筛选 ----
  await test.step('按时间范围筛选', async () => {
    const now = new Date()
    const oneDayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000)
    const start = oneDayAgo.toISOString()
    const end = now.toISOString()
    const res = await API.get(page, `${API_BASE}/audit-logs?page=1&page_size=10&start_time=${start}&end_time=${end}`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    expect(res.data.list).toBeDefined()
    await page.screenshot({ path: 'test-results/AL-2-01-时间筛选.png', fullPage: false })
  })

  // ---- 2. 按关键词筛选 ----
  await test.step('按关键词筛选', async () => {
    const keyword = uid()
    const res = await API.get(page, `${API_BASE}/audit-logs?page=1&page_size=10&keyword=${keyword}`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    expect(res.data.list).toBeDefined()
    // 关键词筛选可能返回空结果，这是正常的
    await page.screenshot({ path: 'test-results/AL-2-02-关键词筛选.png', fullPage: false })
  })

  // ---- 3. 按操作类型筛选 ----
  await test.step('按操作类型筛选', async () => {
    const res = await API.get(page, `${API_BASE}/audit-logs?page=1&page_size=10&action=create`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    expect(res.data.list).toBeDefined()
    await page.screenshot({ path: 'test-results/AL-2-03-操作类型筛选.png', fullPage: false })
  })

  // ---- 4. 组合筛选 ----
  await test.step('组合筛选', async () => {
    const res = await API.get(page, `${API_BASE}/audit-logs?page=1&page_size=5&action=update&keyword=test`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    await page.screenshot({ path: 'test-results/AL-2-04-组合筛选.png', fullPage: false })
  })
})
