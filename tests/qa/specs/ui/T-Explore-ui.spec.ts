import { test, expect } from '../../fixtures/auth'

// 数据查询 UI 测试

test.describe('数据查询 UI 测试', () => {

  test('UI-EXP-1 查询页面加载', async ({ authPage: page }) => {
    await test.step('导航到查询页', async () => {
      await page.goto('/alert/explore')
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证页面元素', async () => {
      await expect(page.locator('body')).toBeVisible()
      // 检查是否有数据源选择器
      const dsSelector = page.locator('.n-select, select').first()
      await expect(dsSelector).toBeVisible({ timeout: 10000 })
    })
  })

  test('UI-EXP-2 执行查询', async ({ authPage: page }) => {
    await test.step('导航到查询页', async () => {
      await page.goto('/alert/explore')
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入查询表达式', async () => {
      const queryInput = page.locator('textarea, .cm-content, input[placeholder*="query"]').first()
      if (await queryInput.isVisible()) {
        await queryInput.fill('up')
      }
    })

    await test.step('执行查询', async () => {
      const executeBtn = page.locator('button').filter({ hasText: /查询|Query|Execute|Run/ }).first()
      if (await executeBtn.isVisible()) {
        await executeBtn.click()
        await page.waitForLoadState('networkidle')
      }
    })
  })
})
