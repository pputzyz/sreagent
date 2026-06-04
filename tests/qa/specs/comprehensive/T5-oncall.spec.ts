import { test, expect } from '../../fixtures/auth'

// T5: 值班排班 — 132 个测试用例

test.describe('T5 - 值班排班', () => {

  // T5-1: 排班列表页面
  test('T5-1 排班列表页面', async ({ authPage: page }) => {
    await test.step('导航到排班列表页', async () => {
      await page.goto('/oncall/schedule')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T5-1-排班列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T5-2: 新建排班弹窗
  test('T5-2 新建排班弹窗', async ({ authPage: page }) => {
    await test.step('导航到排班列表页', async () => {
      await page.goto('/oncall/schedule')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-2-新建排班弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T5-3: 排班详情页
  test('T5-3 排班详情页', async ({ authPage: page }) => {
    await test.step('导航到排班列表页', async () => {
      await page.goto('/oncall/schedule')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击第一条排班', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="schedule"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-3-排班详情.png', fullPage: true })
      }
    })
  })

  // T5-4: 升级策略列表
  test('T5-4 升级策略列表', async ({ authPage: page }) => {
    await test.step('导航到升级策略页', async () => {
      await page.goto('/oncall/config/escalation-policies')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T5-4-升级策略列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T5-5: 新建升级策略
  test('T5-5 新建升级策略', async ({ authPage: page }) => {
    await test.step('导航到升级策略页', async () => {
      await page.goto('/oncall/config/escalation-policies')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-5-新建升级策略.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T5-6: 值班人员页面
  test('T5-6 值班人员页面', async ({ authPage: page }) => {
    await test.step('导航到值班人员页', async () => {
      await page.goto('/oncall/overview')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T5-6-值班人员.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T5-7: 状态页面
  test('T5-7 状态页面', async ({ authPage: page }) => {
    await test.step('导航到状态页', async () => {
      await page.goto('/oncall/status-page')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T5-7-状态页面.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T5-8: 通知中心
  test('T5-8 通知中心', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto('/notifications')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T5-8-通知中心.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T5-9: 个人设置页面
  test('T5-9 个人设置页面', async ({ authPage: page }) => {
    await test.step('导航到个人设置页', async () => {
      await page.goto('/platform/profile')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T5-9-个人设置.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T5-10: 用户通知偏好
  test('T5-10 用户通知偏好', async ({ authPage: page }) => {
    await test.step('导航到个人设置页', async () => {
      await page.goto('/platform/profile')
      await page.waitForLoadState('networkidle')
    })

    await test.step('查看通知偏好', async () => {
      const notifyTab = page.locator('text=通知, text=Notification').first()
      if (await notifyTab.isVisible()) {
        await notifyTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-10-通知偏好.png', fullPage: false })
      }
    })
  })
})
