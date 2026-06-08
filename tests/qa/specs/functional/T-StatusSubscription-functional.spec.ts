import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

// ---------------------------------------------------------------------------
// SS-1 状态页邮件订阅
// ---------------------------------------------------------------------------
test('SS-1 状态页邮件订阅', async ({ authPage: page }) => {
  const tag = uid()
  let subscriptionId: number | null = null

  try {
    // ---- 1. 创建邮件订阅 ----
    await test.step('创建邮件订阅', async () => {
      const res = await API.post(page, `${API_BASE}/status-subscriptions`, {
        email: `test-${tag}@example.com`,
        status_page_id: 1,
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      expect(res.data.id).toBeGreaterThan(0)
      subscriptionId = res.data.id
      await page.screenshot({ path: 'test-results/SS-1-01-创建订阅.png', fullPage: false })
    })

    // ---- 2. 验证订阅已创建 ----
    await test.step('验证订阅已创建', async () => {
      const res = await API.get(page, `${API_BASE}/status-subscriptions?page=1&page_size=100`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      const found = list.find((s: any) => s.id === subscriptionId)
      expect(found).toBeTruthy()
      expect(found.email).toContain(`test-${tag}@example.com`)
      await page.screenshot({ path: 'test-results/SS-1-02-验证订阅.png', fullPage: false })
    })

    // ---- 3. 验证订阅状态 ----
    await test.step('验证订阅状态', async () => {
      const res = await API.get(page, `${API_BASE}/status-subscriptions?page=1&page_size=100`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      const found = list.find((s: any) => s.id === subscriptionId)
      if (found) {
        expect(found).toHaveProperty('email')
        expect(found).toHaveProperty('id')
      }
      await page.screenshot({ path: 'test-results/SS-1-03-订阅状态.png', fullPage: false })
    })
  } finally {
    // cleanup
    if (subscriptionId) {
      try {
        await API.del(page, `${API_BASE}/status-subscriptions/${subscriptionId}`)
      } catch { /* ignore */ }
    }
  }
})

// ---------------------------------------------------------------------------
// SS-2 状态页取消订阅
// ---------------------------------------------------------------------------
test('SS-2 状态页取消订阅', async ({ authPage: page }) => {
  const tag = uid()
  let subscriptionId: number | null = null

  try {
    // ---- 1. 先创建订阅 ----
    await test.step('创建订阅', async () => {
      const res = await API.post(page, `${API_BASE}/status-subscriptions`, {
        email: `unsub-test-${tag}@example.com`,
        status_page_id: 1,
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      subscriptionId = res.data.id
      await page.screenshot({ path: 'test-results/SS-2-01-创建订阅.png', fullPage: false })
    })

    // ---- 2. 取消订阅 ----
    await test.step('取消订阅', async () => {
      const res = await API.del(page, `${API_BASE}/status-subscriptions/${subscriptionId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/SS-2-02-取消订阅.png', fullPage: false })
    })

    // ---- 3. 验证订阅已删除 ----
    await test.step('验证订阅已删除', async () => {
      const res = await API.get(page, `${API_BASE}/status-subscriptions?page=1&page_size=100`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      const found = list.find((s: any) => s.id === subscriptionId)
      expect(found).toBeFalsy()
      await page.screenshot({ path: 'test-results/SS-2-03-验证删除.png', fullPage: false })
    })

    subscriptionId = null
  } finally {
    if (subscriptionId) {
      try {
        await API.del(page, `${API_BASE}/status-subscriptions/${subscriptionId}`)
      } catch { /* ignore */ }
    }
  }
})
