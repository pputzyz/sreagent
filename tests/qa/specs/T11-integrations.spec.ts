import { test, expect } from '../fixtures/auth'

// T11: 集成中心 — 冒烟测试

test.describe('T11 - 集成中心', () => {

  test('T11-1 集成列表', async ({ authPage: page }) => {
    await page.goto('/alert/integrations')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })

  test('T11-2 路由规则', async ({ authPage: page }) => {
    await page.goto('/alert/routing-rules')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })
})
