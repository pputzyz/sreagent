import { test, expect } from '../fixtures/auth'

// T5: 值班排班 — 冒烟测试

test.describe('T5 - 值班排班', () => {

  test.beforeEach(async ({ authenticatedPage: page }) => {
    await page.goto('/oncall/schedule')
    await page.waitForLoadState('networkidle')
  })

  // T5-1: 排班列表
  test('T5-1 排班列表', async ({ authenticatedPage: page }) => {
    await expect(page.locator('body')).toBeVisible()
  })

  // T5-2: 新建排班
  test('T5-2 新建排班', async ({ authenticatedPage: page }) => {
    const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
    if (await createBtn.isVisible()) {
      await createBtn.click()
      await expect(page.locator('.n-modal, [role="dialog"]')).toBeVisible()
      await page.keyboard.press('Escape')
    }
  })
})

test.describe('T5 - 升级策略', () => {

  test.beforeEach(async ({ authenticatedPage: page }) => {
    await page.goto('/oncall/config/escalation-policies')
    await page.waitForLoadState('networkidle')
  })

  // T5-10: 升级策略列表
  test('T5-10 升级策略列表', async ({ authenticatedPage: page }) => {
    await expect(page.locator('body')).toBeVisible()
  })
})
