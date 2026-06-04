import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

test.describe('故障管理功能测试', () => {

  test('INC-1 故障列表页面', async ({ authPage: page }) => {
    await test.step('导航到故障列表页', async () => {
      await page.goto('/oncall/incidents')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/INC-1-故障列表.png', fullPage: true })
    })

    await test.step('验证页面元素', async () => {
      await expect(page.locator('h1, h2').filter({ hasText: /故障|Incident/ }).first()).toBeVisible()
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      await expect(createBtn).toBeVisible()
    })
  })

  test('INC-2 创建故障弹窗', async ({ authPage: page }) => {
    await test.step('导航到故障列表页', async () => {
      await page.goto('/oncall/incidents')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击创建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      await createBtn.click()
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/INC-2-创建弹窗.png', fullPage: false })
    })

    await test.step('验证弹窗内容', async () => {
      const modal = page.locator('.n-modal, [role="dialog"]').first()
      await expect(modal).toBeVisible()
      // 验证有标题输入框
      const titleInput = modal.locator('input').first()
      await expect(titleInput).toBeVisible()
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  test('INC-3 故障详情页', async ({ authPage: page }) => {
    await test.step('导航到故障列表页', async () => {
      await page.goto('/oncall/incidents')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击第一条故障', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="incident"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/INC-3-故障详情.png', fullPage: false })
      }
    })
  })
})
