import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

// ---------------------------------------------------------------------------
// NC-1 通知中心列表
// ---------------------------------------------------------------------------
test('NC-1 通知中心列表', async ({ authPage: page }) => {
  // ---- 1. 获取通知列表 ----
  await test.step('获取通知列表', async () => {
    const res = await API.get(page, `${API_BASE}/notifications?page=1&page_size=10`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    expect(res.data.list).toBeDefined()
    expect(Array.isArray(res.data.list)).toBe(true)
    await page.screenshot({ path: 'test-results/NC-1-01-通知列表.png', fullPage: false })
  })

  // ---- 2. 验证通知条目结构 ----
  await test.step('验证通知条目结构', async () => {
    const res = await API.get(page, `${API_BASE}/notifications?page=1&page_size=5`)
    expect(res.code).toBe(0)
    const list = res.data.list || []
    if (list.length > 0) {
      const notification = list[0]
      expect(notification).toHaveProperty('id')
      expect(typeof notification.id).toBe('number')
    }
    await page.screenshot({ path: 'test-results/NC-1-02-条目结构.png', fullPage: false })
  })

  // ---- 3. 验证分页 ----
  await test.step('验证分页', async () => {
    const res = await API.get(page, `${API_BASE}/notifications?page=1&page_size=2`)
    expect(res.code).toBe(0)
    expect(res.data.list.length).toBeLessThanOrEqual(2)
    expect(res.data).toHaveProperty('total')
    await page.screenshot({ path: 'test-results/NC-1-03-分页验证.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// NC-2 通知中心未读数
// ---------------------------------------------------------------------------
test('NC-2 通知中心未读数', async ({ authPage: page }) => {
  // ---- 1. 获取未读通知数 ----
  await test.step('获取未读通知数', async () => {
    const res = await API.get(page, `${API_BASE}/notifications/unread-count`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    expect(typeof res.data.count).toBe('number')
    expect(res.data.count).toBeGreaterThanOrEqual(0)
    await page.screenshot({ path: 'test-results/NC-2-01-未读数.png', fullPage: false })
  })

  // ---- 2. 验证未读数与列表一致 ----
  await test.step('验证未读数与列表一致', async () => {
    const countRes = await API.get(page, `${API_BASE}/notifications/unread-count`)
    expect(countRes.code).toBe(0)
    const unreadCount = countRes.data.count

    const listRes = await API.get(page, `${API_BASE}/notifications?page=1&page_size=100&status=unread`)
    expect(listRes.code).toBe(0)
    const listTotal = listRes.data.total || 0

    // 未读数应与未读列表总数一致
    expect(unreadCount).toBe(listTotal)
    await page.screenshot({ path: 'test-results/NC-2-02-一致性验证.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// NC-3 通知中心标记已读
// ---------------------------------------------------------------------------
test('NC-3 通知中心标记已读', async ({ authPage: page }) => {
  let notificationId: number | null = null

  try {
    // ---- 1. 获取一条未读通知 ----
    await test.step('获取一条未读通知', async () => {
      const res = await API.get(page, `${API_BASE}/notifications?page=1&page_size=10&status=unread`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      if (list.length > 0) {
        notificationId = list[0].id
        expect(typeof notificationId).toBe('number')
      }
      await page.screenshot({ path: 'test-results/NC-3-01-获取未读通知.png', fullPage: false })
    })

    // ---- 2. 标记单条已读 ----
    if (notificationId) {
      await test.step('标记单条已读', async () => {
        const res = await API.put(page, `${API_BASE}/notifications/${notificationId}/read`)
        expect(res.code).toBe(0)
        await page.screenshot({ path: 'test-results/NC-3-02-标记已读.png', fullPage: false })
      })

      // ---- 3. 验证已标记已读 ----
      await test.step('验证已标记已读', async () => {
        const res = await API.get(page, `${API_BASE}/notifications?page=1&page_size=100&status=read`)
        expect(res.code).toBe(0)
        const readList = res.data.list || []
        const found = readList.find((n: any) => n.id === notificationId)
        expect(found).toBeTruthy()
        await page.screenshot({ path: 'test-results/NC-3-03-验证已读.png', fullPage: false })
      })
    }
  } finally {
    // cleanup is not needed — marking as read is not destructive
    notificationId = null
  }
})

// ---------------------------------------------------------------------------
// NC-4 通知中心全部已读
// ---------------------------------------------------------------------------
test('NC-4 通知中心全部已读', async ({ authPage: page }) => {
  // ---- 1. 记录当前未读数 ----
  let beforeUnread = 0
  await test.step('记录当前未读数', async () => {
    const res = await API.get(page, `${API_BASE}/notifications/unread-count`)
    expect(res.code).toBe(0)
    beforeUnread = res.data.count
    await page.screenshot({ path: 'test-results/NC-4-01-当前未读数.png', fullPage: false })
  })

  // ---- 2. 全部标记已读 ----
  await test.step('全部标记已读', async () => {
    const res = await API.post(page, `${API_BASE}/notifications/read-all`)
    expect(res.code).toBe(0)
    await page.screenshot({ path: 'test-results/NC-4-02-全部已读.png', fullPage: false })
  })

  // ---- 3. 验证未读数为 0 ----
  await test.step('验证未读数为 0', async () => {
    const res = await API.get(page, `${API_BASE}/notifications/unread-count`)
    expect(res.code).toBe(0)
    expect(res.data.count).toBe(0)
    await page.screenshot({ path: 'test-results/NC-4-03-未读数为0.png', fullPage: false })
  })
})
