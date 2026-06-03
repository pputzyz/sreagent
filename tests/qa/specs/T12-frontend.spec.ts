import { test, expect } from '../fixtures/auth'

// T12: 前端通用 — 冒烟测试

test.describe('T12 - 前端通用', () => {

  test('T12-1 登录页', async ({ page }) => {
    await page.goto('/login')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('input[type="password"]')).toBeVisible()
  })

  test('T12-2 侧边栏导航', async ({ authenticatedPage: page }) => {
    await expect(page.locator('nav, [class*="sidebar"], [class*="rail"]')).toBeVisible()
  })

  test('T12-3 主题切换', async ({ authenticatedPage: page }) => {
    const themeBtn = page.locator('button').filter({ hasText: /主题|Theme|🌙|☀️/ }).first()
    if (await themeBtn.isVisible()) {
      await themeBtn.click()
    }
  })

  test('T12-4 语言切换', async ({ authenticatedPage: page }) => {
    const langBtn = page.locator('button').filter({ hasText: /语言|Language|🌐/ }).first()
    if (await langBtn.isVisible()) {
      await langBtn.click()
    }
  })

  test('T12-5 404 页面', async ({ authenticatedPage: page }) => {
    await page.goto('/nonexistent-page')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('text=404, text=未找到, text=Not Found')).toBeVisible()
  })
})
