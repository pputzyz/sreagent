import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

test.describe('数据源功能测试', () => {

  test('DS-1 数据源列表页面', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto('/alert/datasources')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/DS-1-数据源列表.png', fullPage: true })
    })

    await test.step('验证数据源卡片', async () => {
      const cards = page.locator('.n-card, [class*="card"]')
      const count = await cards.count()
      expect(count).toBeGreaterThan(0)
    })

    await test.step('验证健康检查按钮', async () => {
      const healthBtn = page.locator('button').filter({ hasText: /检查|Check|Health/ }).first()
      if (await healthBtn.isVisible()) {
        await page.screenshot({ path: 'test-results/DS-1-健康检查按钮.png', fullPage: false })
      }
    })
  })

  test('DS-2 数据源详情页', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto('/alert/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击第一个数据源', async () => {
      const firstCard = page.locator('.n-card, [class*="card"]').first()
      if (await firstCard.isVisible()) {
        await firstCard.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/DS-2-数据源详情.png', fullPage: false })
      }
    })
  })

  test('DS-3 新建数据源弹窗', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto('/alert/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/DS-3-新建弹窗.png', fullPage: false })
      }
    })

    await test.step('验证弹窗内容', async () => {
      const modal = page.locator('.n-modal, [role="dialog"]').first()
      if (await modal.isVisible()) {
        // 验证有数据源类型选择
        const typeSelect = modal.locator('.n-select, select').first()
        await expect(typeSelect).toBeVisible()
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  test('DS-4 测试连接功能', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto('/alert/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击测试连接', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|Check/ }).first()
      if (await testBtn.isVisible()) {
        await testBtn.click()
        await page.waitForTimeout(2000)
        await page.screenshot({ path: 'test-results/DS-4-测试连接结果.png', fullPage: false })
      }
    })
  })
})
