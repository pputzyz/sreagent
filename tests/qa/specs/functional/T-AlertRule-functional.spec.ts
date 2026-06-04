import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

test.describe('告警规则功能测试', () => {

  test('AR-1 告警规则列表页面', async ({ authPage: page }) => {
    await test.step('导航到告警规则页', async () => {
      await page.goto('/alert/rules')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/AR-1-规则列表.png', fullPage: true })
    })

    await test.step('验证页面元素', async () => {
      await expect(page.locator('h1, h2').filter({ hasText: /告警规则|Alert Rules/ }).first()).toBeVisible()
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      await expect(createBtn).toBeVisible()
    })

    await test.step('验证规则卡片存在', async () => {
      const cards = page.locator('.rule-card, [class*="rule-card"], [class*="card"]')
      const count = await cards.count()
      if (count > 0) {
        await expect(cards.first()).toBeVisible()
        await page.screenshot({ path: 'test-results/AR-1-规则卡片.png', fullPage: false })
      }
    })
  })

  test('AR-2 创建告警规则流程', async ({ authPage: page }) => {
    await test.step('导航到告警规则页', async () => {
      await page.goto('/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击创建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      await createBtn.click()
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/AR-2-创建弹窗.png', fullPage: false })
    })

    await test.step('验证弹窗内容', async () => {
      const modal = page.locator('.n-modal, [role="dialog"]').first()
      await expect(modal).toBeVisible()
      // 验证弹窗有必要的表单字段
      const nameInput = modal.locator('input').first()
      await expect(nameInput).toBeVisible()
    })

    await test.step('填写表单', async () => {
      const modal = page.locator('.n-modal, [role="dialog"]').first()
      const nameInput = modal.locator('input').first()
      await nameInput.fill('测试规则-' + Date.now())
      await page.screenshot({ path: 'test-results/AR-2-填写表单.png', fullPage: false })
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  test('AR-3 告警规则分类筛选', async ({ authPage: page }) => {
    await test.step('导航到告警规则页', async () => {
      await page.goto('/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证分类侧栏', async () => {
      const categories = page.locator('button').filter({ hasText: /全部|All|CPU|Memory|Disk/ })
      const count = await categories.count()
      await page.screenshot({ path: 'test-results/AR-3-分类侧栏.png', fullPage: false })
      expect(count).toBeGreaterThan(0)
    })

    await test.step('点击分类筛选', async () => {
      const categories = page.locator('button').filter({ hasText: /全部|All|CPU|Memory|Disk/ })
      if (await categories.count() > 1) {
        await categories.nth(1).click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/AR-3-筛选结果.png', fullPage: false })
      }
    })
  })

  test('AR-4 告警规则搜索', async ({ authPage: page }) => {
    await test.step('导航到告警规则页', async () => {
      await page.goto('/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible()) {
        await searchInput.fill('CPU')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/AR-4-搜索结果.png', fullPage: false })
      }
    })
  })

  test('AR-5 告警规则详情页', async ({ authPage: page }) => {
    await test.step('导航到告警规则页', async () => {
      await page.goto('/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击第一条规则', async () => {
      const firstCard = page.locator('.rule-card, [class*="rule-card"], [class*="card"]').first()
      if (await firstCard.isVisible()) {
        await firstCard.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/AR-5-规则详情.png', fullPage: false })
      }
    })
  })
})
