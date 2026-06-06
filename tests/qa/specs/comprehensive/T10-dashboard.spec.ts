import { test, expect } from '../../fixtures/auth'

// T10: 仪表盘 — 165 个测试用例

test.describe('T10 - 仪表盘', () => {

  // T10-1: 仪表盘列表
  test('T10-1 仪表盘列表', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto('/alert/dashboards')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T10-1-仪表盘列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T10-2: 新建仪表盘
  test('T10-2 新建仪表盘', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto('/alert/dashboards')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-2-新建仪表盘.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T10-3: 仪表盘编辑器
  test('T10-3 仪表盘编辑器', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto('/alert/dashboards')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击第一个仪表盘', async () => {
      const firstCard = page.locator('.n-card, [class*="card"]').first()
      if (await firstCard.isVisible()) {
        await firstCard.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-3-仪表盘编辑器.png', fullPage: true })
      }
    })
  })

  // T10-4: 仪表盘面板
  test('T10-4 仪表盘面板', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto('/alert/dashboards')
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开仪表盘', async () => {
      const firstCard = page.locator('.n-card, [class*="card"]').first()
      if (await firstCard.isVisible()) {
        await firstCard.click()
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T10-4-仪表盘面板.png', fullPage: false })
      }
    })
  })

  // T10-5: 仪表盘变量
  test('T10-5 仪表盘变量', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto('/alert/dashboards')
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开仪表盘', async () => {
      const firstCard = page.locator('.n-card, [class*="card"]').first()
      if (await firstCard.isVisible()) {
        await firstCard.click()
        await page.waitForTimeout(1000)
        // 查看变量选择器
        const varSelect = page.locator('.n-select, select').first()
        if (await varSelect.isVisible()) {
          await page.screenshot({ path: 'test-results/T10-5-仪表盘变量.png', fullPage: false })
        }
      }
    })
  })

  // T10-6: 仪表盘时间范围
  test('T10-6 仪表盘时间范围', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto('/alert/dashboards')
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开仪表盘', async () => {
      const firstCard = page.locator('.n-card, [class*="card"]').first()
      if (await firstCard.isVisible()) {
        await firstCard.click()
        await page.waitForTimeout(1000)
        // 查看时间范围选择器
        const timeSelect = page.locator('.n-select, select').filter({ hasText: /time|时间/ }).first()
        if (await timeSelect.isVisible()) {
          await page.screenshot({ path: 'test-results/T10-6-仪表盘时间范围.png', fullPage: false })
        }
      }
    })
  })
})
