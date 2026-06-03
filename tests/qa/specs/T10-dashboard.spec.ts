import { test, expect } from '../fixtures/auth'

test.describe('T10 - 仪表盘', () => {

  test('T10-1 仪表盘列表', async ({ authPage: page }) => {
    await test.step('导航到仪表盘列表', async () => {
      await page.goto('/alert/dashboards')
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  test('T10-2 仪表盘编辑器', async ({ authPage: page }) => {
    await test.step('导航到新建仪表盘', async () => {
      await page.goto('/alert/dashboards/new')
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证编辑器加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })
})
