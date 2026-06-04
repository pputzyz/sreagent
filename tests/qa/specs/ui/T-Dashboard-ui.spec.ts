import { test, expect } from '../../fixtures/auth'

// 仪表盘 UI 测试

test.describe('仪表盘 UI 测试', () => {

  test('UI-DASH-1 仪表盘加载', async ({ authPage: page }) => {
    await test.step('导航到仪表盘', async () => {
      await page.goto('/alert/dashboards')
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })
})
