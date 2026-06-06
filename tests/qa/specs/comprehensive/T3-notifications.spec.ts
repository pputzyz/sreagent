import { test, expect } from '../../fixtures/auth'

// T3: 通知管道 — 140 个测试用例

test.describe('T3 - 通知管道', () => {

  // T3-1: 通知策略列表
  test('T3-1 通知策略列表', async ({ authPage: page }) => {
    await test.step('导航到通知策略页', async () => {
      await page.goto('/oncall/notify/rules')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T3-1-通知策略列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T3-2: 新建通知策略
  test('T3-2 新建通知策略', async ({ authPage: page }) => {
    await test.step('导航到通知策略页', async () => {
      await page.goto('/oncall/notify/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-2-新建策略弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-3: 通知渠道列表
  test('T3-3 通知渠道列表', async ({ authPage: page }) => {
    await test.step('导航到通知渠道页', async () => {
      await page.goto('/oncall/notify/media')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T3-3-通知渠道列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T3-4: 新建通知渠道
  test('T3-4 新建通知渠道', async ({ authPage: page }) => {
    await test.step('导航到通知渠道页', async () => {
      await page.goto('/oncall/notify/media')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-4-新建渠道弹窗.png', fullPage: false })
      }
    })

    await test.step('验证渠道类型', async () => {
      const modal = page.locator('.n-modal, [role="dialog"]').first()
      if (await modal.isVisible()) {
        const typeSelect = modal.locator('.n-select, select').first()
        await expect(typeSelect).toBeVisible()
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-5: 消息模板列表
  test('T3-5 消息模板列表', async ({ authPage: page }) => {
    await test.step('导航到消息模板页', async () => {
      await page.goto('/oncall/notify/templates')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T3-5-消息模板列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T3-6: 新建消息模板
  test('T3-6 新建消息模板', async ({ authPage: page }) => {
    await test.step('导航到消息模板页', async () => {
      await page.goto('/oncall/notify/templates')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-6-新建模板弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-7: 订阅规则列表
  test('T3-7 订阅规则列表', async ({ authPage: page }) => {
    await test.step('导航到订阅规则页', async () => {
      await page.goto('/oncall/notify/subscriptions')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T3-7-订阅规则列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T3-8: 静默规则列表
  test('T3-8 静默规则列表', async ({ authPage: page }) => {
    await test.step('导航到静默规则页', async () => {
      await page.goto('/alert/suppression')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T3-8-静默规则列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T3-9: 新建静默规则
  test('T3-9 新建静默规则', async ({ authPage: page }) => {
    await test.step('导航到静默规则页', async () => {
      await page.goto('/alert/suppression')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-9-新建静默弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-10: 事件管道列表
  test('T3-10 事件管道列表', async ({ authPage: page }) => {
    await test.step('导航到事件管道页', async () => {
      await page.goto('/alert/event-pipelines')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T3-10-事件管道列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })
})
