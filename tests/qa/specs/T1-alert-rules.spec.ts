import { test, expect } from '../fixtures/auth'

test.describe('T1 - 告警规则', () => {

  test('T1-1 列表正常加载', async ({ authPage: page }) => {
    await test.step('导航到告警规则页', async () => {
      await page.goto('/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证页面标题', async () => {
      await expect(page.locator('h1, h2').filter({ hasText: /告警规则|Alert Rules/ }).first()).toBeVisible({ timeout: 10000 })
    })

    await test.step('验证分类侧栏', async () => {
      const sidebar = page.locator('text=/全部|All/').first()
      await expect(sidebar).toBeVisible({ timeout: 10000 })
    })

    await test.step('验证创建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      await expect(createBtn).toBeVisible({ timeout: 10000 })
    })
  })

  test('T1-2 新建规则弹窗', async ({ authPage: page }) => {
    await test.step('导航到告警规则页', async () => {
      await page.goto('/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击创建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      await expect(createBtn).toBeVisible({ timeout: 10000 })
      await createBtn.click()
    })

    await test.step('验证弹窗打开', async () => {
      const modal = page.locator('.n-modal, [role="dialog"]').first()
      await expect(modal).toBeVisible({ timeout: 5000 })
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  test('T1-3 搜索框', async ({ authPage: page }) => {
    await test.step('导航到告警规则页', async () => {
      await page.goto('/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索内容', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first()
      if (await searchInput.isVisible()) {
        await searchInput.fill('test')
        await page.waitForTimeout(500)
      }
    })
  })
})
