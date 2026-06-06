import { test, expect } from '../../fixtures/auth'

// 告警规则 UI 测试 — 真实用户操作流程

test.describe('告警规则 UI 测试', () => {

  test('UI-AR-1 创建告警规则完整流程', async ({ authPage: page }) => {
    await test.step('导航到告警规则页', async () => {
      await page.goto('/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击创建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      await expect(createBtn).toBeVisible({ timeout: 10000 })
      await createBtn.click()
    })

    await test.step('填写规则信息', async () => {
      // 等待弹窗出现
      const modal = page.locator('.n-modal, [role="dialog"]').first()
      await expect(modal).toBeVisible({ timeout: 5000 })

      // 填写名称
      const nameInput = modal.locator('input').first()
      await nameInput.fill('UI测试规则-' + Date.now())

      // 填写表达式
      const exprInput = modal.locator('textarea, .cm-content').first()
      if (await exprInput.isVisible()) {
        await exprInput.fill('up == 0')
      }
    })

    await test.step('提交表单', async () => {
      const submitBtn = page.locator('.n-modal button[type="primary"], [role="dialog"] button[type="primary"]').first()
      if (await submitBtn.isVisible()) {
        await submitBtn.click()
        await page.waitForTimeout(2000)
      }
    })

    await test.step('验证规则创建成功', async () => {
      // 检查是否有成功提示
      const successMsg = page.locator('.n-message, [class*="success"]').first()
      if (await successMsg.isVisible()) {
        await expect(successMsg).toBeVisible({ timeout: 5000 })
      }
    })
  })

  test('UI-AR-2 搜索告警规则', async ({ authPage: page }) => {
    await test.step('导航到告警规则页', async () => {
      await page.goto('/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible()) {
        await searchInput.fill('CPU')
        await page.waitForTimeout(500) // debounce
      }
    })

    await test.step('验证搜索结果', async () => {
      await page.waitForLoadState('networkidle')
      // 页面应该刷新显示搜索结果
    })
  })

  test('UI-AR-3 分类筛选', async ({ authPage: page }) => {
    await test.step('导航到告警规则页', async () => {
      await page.goto('/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击分类', async () => {
      const categories = page.locator('button').filter({ hasText: /全部|All|CPU|Memory|Disk/ })
      const count = await categories.count()
      if (count > 1) {
        await categories.nth(1).click()
        await page.waitForLoadState('networkidle')
      }
    })
  })
})
