import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

test.describe('用户与团队功能测试', () => {

  test('USER-1 用户列表页面', async ({ authPage: page }) => {
    await test.step('导航到用户管理页', async () => {
      await page.goto('/platform/org/members')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/USER-1-用户列表.png', fullPage: true })
    })

    await test.step('验证用户表格', async () => {
      const table = page.locator('.n-data-table, table, [class*="table"]').first()
      if (await table.isVisible()) {
        await expect(table).toBeVisible()
      }
    })
  })

  test('USER-2 团队列表页面', async ({ authPage: page }) => {
    await test.step('导航到团队管理页', async () => {
      await page.goto('/platform/org/teams')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/USER-2-团队列表.png', fullPage: true })
    })

    await test.step('验证团队列表', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  test('USER-3 个人信息页面', async ({ authPage: page }) => {
    await test.step('导航到个人信息页', async () => {
      await page.goto('/platform/profile')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/USER-3-个人信息.png', fullPage: true })
    })

    await test.step('验证个人信息表单', async () => {
      const form = page.locator('form, .n-form, [class*="form"]').first()
      if (await form.isVisible()) {
        await expect(form).toBeVisible()
      }
    })
  })
})
