import { test, expect } from '../fixtures/auth'

// T10: 仪表盘 — 冒烟测试

test.describe('T10 - 仪表盘', () => {

  test('T10-1 仪表盘列表', async ({ authenticatedPage: page }) => {
    await page.goto('/alert/dashboards')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })

  test('T10-2 新建仪表盘', async ({ authenticatedPage: page }) => {
    await page.goto('/alert/dashboards')
    await page.waitForLoadState('networkidle')
    const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
    if (await createBtn.isVisible()) {
      await createBtn.click()
      await expect(page.locator('.n-modal, [role="dialog"]')).toBeVisible()
      await page.keyboard.press('Escape')
    }
  })
})
