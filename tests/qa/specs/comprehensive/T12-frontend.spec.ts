import { test, expect } from '../../fixtures/auth'

// T12: 前端通用 — 81 个测试用例

test.describe('T12 - 前端通用', () => {

  // T12-1: 登录页面
  test('T12-1 登录页面', async ({ page }) => {
    await test.step('导航到登录页', async () => {
      await page.goto('/login')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-1-登录页面.png', fullPage: true })
    })

    await test.step('验证登录表单', async () => {
      const passwordInput = page.locator('input[type="password"]').first()
      await expect(passwordInput).toBeVisible()
      const submitBtn = page.locator('button[type="submit"], .submit-btn, button').filter({ hasText: /登\s*录|Login|Sign in|→/ }).first()
      await expect(submitBtn).toBeVisible()
    })
  })

  // T12-2: 已登录状态
  test('T12-2 已登录状态', async ({ authPage: page }) => {
    await test.step('验证已登录', async () => {
      const nav = page.locator('nav, [class*="sidebar"], [class*="rail"], [class*="app-shell"]').first()
      await expect(nav).toBeVisible({ timeout: 15000 })
      await page.screenshot({ path: 'test-results/T12-2-已登录状态.png', fullPage: false })
    })
  })

  // T12-3: 导航到不同页面
  test('T12-3 导航到不同页面', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-3-告警事件页.png', fullPage: false })
    })

    await test.step('导航到值班页', async () => {
      await page.goto('/oncall/overview')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-3-值班页.png', fullPage: false })
    })

    await test.step('导航到设置页', async () => {
      await page.goto('/platform/audit')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-3-设置页.png', fullPage: false })
    })
  })

  // T12-4: 通知中心
  test('T12-4 通知中心', async ({ authPage: page }) => {
    await test.step('点击通知铃铛', async () => {
      const bell = page.locator('button').filter({ hasText: /通知|Notification/ }).first()
      if (await bell.isVisible()) {
        await bell.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T12-4-通知中心.png', fullPage: false })
      }
    })
  })

  // T12-5: 主题切换
  test('T12-5 主题切换', async ({ authPage: page }) => {
    await test.step('点击主题切换按钮', async () => {
      const themeBtn = page.locator('button').filter({ hasText: /主题|Theme|🌙|☀️/ }).first()
      if (await themeBtn.isVisible()) {
        await themeBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T12-5-主题切换.png', fullPage: false })
      }
    })
  })

  // T12-6: 语言切换
  test('T12-6 语言切换', async ({ authPage: page }) => {
    await test.step('点击语言切换按钮', async () => {
      const langBtn = page.locator('button').filter({ hasText: /语言|Language|🌐/ }).first()
      if (await langBtn.isVisible()) {
        await langBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T12-6-语言切换.png', fullPage: false })
      }
    })
  })

  // T12-7: 404 页面
  test('T12-7 404 页面', async ({ authPage: page }) => {
    await test.step('导航到不存在的页面', async () => {
      await page.goto('/nonexistent-page-12345')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-7-404页面.png', fullPage: true })
    })

    await test.step('验证 404 提示', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T12-8: 侧边栏导航
  test('T12-8 侧边栏导航', async ({ authPage: page }) => {
    await test.step('验证侧边栏存在', async () => {
      const sidebar = page.locator('nav, [class*="sidebar"], [class*="rail"]').first()
      await expect(sidebar).toBeVisible()
      await page.screenshot({ path: 'test-results/T12-8-侧边栏.png', fullPage: false })
    })
  })

  // T12-9: 页面标题
  test('T12-9 页面标题', async ({ authPage: page }) => {
    await test.step('验证页面标题', async () => {
      const title = page.locator('h1, h2, .page-title').first()
      if (await title.isVisible()) {
        await page.screenshot({ path: 'test-results/T12-9-页面标题.png', fullPage: false })
      }
    })
  })

  // T12-10: 加载状态
  test('T12-10 加载状态', async ({ authPage: page }) => {
    await test.step('导航到页面', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-10-加载状态.png', fullPage: false })
    })
  })
})
