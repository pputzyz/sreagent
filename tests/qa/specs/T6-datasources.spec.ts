import { test, expect } from '../fixtures/auth'

// T6: 数据源 — 冒烟测试

test.describe('T6 - 数据源', () => {

  test.beforeEach(async ({ authPage: page }) => {
    await page.goto('/alert/datasources')
    await page.waitForLoadState('networkidle')
  })

  // T6-1: 数据源列表
  test('T6-1 数据源列表', async ({ authPage: page }) => {
    await expect(page.locator('body')).toBeVisible()
  })

  // T6-2: 新建数据源
  test('T6-2 新建数据源弹窗', async ({ authPage: page }) => {
    const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
    if (await createBtn.isVisible()) {
      await createBtn.click()
      await expect(page.locator('.n-modal, [role="dialog"]')).toBeVisible()
      await page.keyboard.press('Escape')
    }
  })
})

test.describe('T6 - 数据查询', () => {

  test.beforeEach(async ({ authPage: page }) => {
    await page.goto('/alert/explore')
    await page.waitForLoadState('networkidle')
  })

  // T6-10: 查询页面加载
  test('T6-10 查询页面加载', async ({ authPage: page }) => {
    await expect(page.locator('body')).toBeVisible()
  })
})
