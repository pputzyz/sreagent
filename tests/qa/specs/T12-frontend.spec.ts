import { test, expect } from '../fixtures/auth'

test.describe('T12 - 前端通用', () => {

  test('T12-1 已登录状态', async ({ authPage: page }) => {
    await test.step('验证已登录（侧边栏可见）', async () => {
      const nav = page.locator('nav, [class*="sidebar"], [class*="rail"], [class*="app-shell"]').first()
      await expect(nav).toBeVisible({ timeout: 15000 })
    })
  })

  test('T12-2 导航到不同页面', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
      await expect(page.locator('body')).toBeVisible()
    })
  })

  test('T12-3 通知中心', async ({ authPage: page }) => {
    await test.step('点击通知铃铛', async () => {
      const bell = page.locator('button').filter({ hasText: /通知|Notification/ }).first()
      if (await bell.isVisible()) {
        await bell.click()
      }
    })
  })
})
