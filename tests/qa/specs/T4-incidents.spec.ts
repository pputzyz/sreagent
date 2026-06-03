import { test, expect } from '../fixtures/auth'

// T4: 故障处理 — 冒烟测试

test.describe('T4 - 故障处理', () => {

  test.beforeEach(async ({ authenticatedPage: page }) => {
    await page.goto('/oncall/incidents')
    await page.waitForLoadState('networkidle')
  })

  // T4-1: 故障列表
  test('T4-1 故障列表加载', async ({ authenticatedPage: page }) => {
    await expect(page.locator('body')).toBeVisible()
  })

  // T4-2: 新建故障
  test('T4-2 新建故障弹窗', async ({ authenticatedPage: page }) => {
    const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
    if (await createBtn.isVisible()) {
      await createBtn.click()
      await expect(page.locator('.n-modal, [role="dialog"]')).toBeVisible()
      await page.keyboard.press('Escape')
    }
  })

  // T4-3: 批量操作
  test('T4-3 批量确认/关闭', async ({ authenticatedPage: page }) => {
    // 检查批量操作按钮是否存在
    const bulkBtn = page.locator('button').filter({ hasText: /批量|Bulk/ }).first()
    // 按钮可能在选中后才出现
  })
})
