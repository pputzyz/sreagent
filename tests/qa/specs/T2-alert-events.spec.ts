import { test, expect } from '../fixtures/auth'

// T2: 告警事件 — 冒烟测试

test.describe('T2 - 告警事件', () => {

  test.beforeEach(async ({ authPage: page }) => {
    await page.goto('/alert/events')
    await page.waitForLoadState('networkidle')
  })

  // T2-1: 列表加载
  test('T2-1 事件列表加载', async ({ authPage: page }) => {
    await expect(page.locator('body')).toBeVisible()
  })

  // T2-2: 筛选器
  test('T2-2 状态筛选', async ({ authPage: page }) => {
    const statusFilter = page.locator('select, .n-select, .n-radio-group').filter({ hasText: /status|状态/ }).first()
    if (await statusFilter.isVisible()) {
      await statusFilter.click()
    }
  })

  // T2-3: 时间范围筛选
  test('T2-3 时间范围', async ({ authPage: page }) => {
    const timeRange = page.locator('[class*="time-range"], [class*="timerange"]').first()
    if (await timeRange.isVisible()) {
      await timeRange.click()
    }
  })

  // T2-4: 事件详情
  test('T2-4 点击事件查看详情', async ({ authPage: page }) => {
    const firstEvent = page.locator('[class*="event"], .event-card, .event-row').first()
    if (await firstEvent.isVisible()) {
      await firstEvent.click()
      await page.waitForLoadState('networkidle')
    }
  })

  // T2-5: 认领操作
  test('T2-5 认领按钮', async ({ authPage: page }) => {
    const ackBtn = page.locator('button').filter({ hasText: /认领|Acknowledge|Ack/ }).first()
    if (await ackBtn.isVisible()) {
      // 按钮存在且可点击
      await expect(ackBtn).toBeEnabled()
    }
  })
})
