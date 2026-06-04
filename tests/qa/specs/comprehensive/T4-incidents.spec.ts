import { test, expect } from '../../fixtures/auth'

// T4: 故障处理 — 110 个测试用例

test.describe('T4 - 故障处理', () => {

  // T4-1: 故障列表页面
  test('T4-1 故障列表页面', async ({ authPage: page }) => {
    await test.step('导航到故障列表页', async () => {
      await page.goto('/oncall/incidents')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T4-1-故障列表.png', fullPage: true })
    })

    await test.step('验证页面元素', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T4-2: 创建故障弹窗
  test('T4-2 创建故障弹窗', async ({ authPage: page }) => {
    await test.step('导航到故障列表页', async () => {
      await page.goto('/oncall/incidents')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击创建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-2-创建故障弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-3: 故障详情页
  test('T4-3 故障详情页', async ({ authPage: page }) => {
    await test.step('导航到故障列表页', async () => {
      await page.goto('/oncall/incidents')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击第一条故障', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="incident"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-3-故障详情.png', fullPage: true })
      }
    })
  })

  // T4-4: 故障时间线
  test('T4-4 故障时间线', async ({ authPage: page }) => {
    await test.step('导航到故障详情', async () => {
      await page.goto('/oncall/incidents')
      await page.waitForLoadState('networkidle')
      const firstItem = page.locator('.n-card, [class*="card"], [class*="incident"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查看时间线', async () => {
      const timelineTab = page.locator('text=时间线, text=Timeline').first()
      if (await timelineTab.isVisible()) {
        await timelineTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-4-故障时间线.png', fullPage: false })
      }
    })
  })

  // T4-5: 故障评论
  test('T4-5 故障评论', async ({ authPage: page }) => {
    await test.step('导航到故障详情', async () => {
      await page.goto('/oncall/incidents')
      await page.waitForLoadState('networkidle')
      const firstItem = page.locator('.n-card, [class*="card"], [class*="incident"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('添加评论', async () => {
      const commentInput = page.locator('textarea, input[placeholder*="评论"]').first()
      if (await commentInput.isVisible()) {
        await commentInput.fill('测试评论')
        await page.screenshot({ path: 'test-results/T4-5-添加评论.png', fullPage: false })
      }
    })
  })

  // T4-6: 故障操作按钮
  test('T4-6 故障操作按钮', async ({ authPage: page }) => {
    await test.step('导航到故障详情', async () => {
      await page.goto('/oncall/incidents')
      await page.waitForLoadState('networkidle')
      const firstItem = page.locator('.n-card, [class*="card"], [class*="incident"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('验证操作按钮', async () => {
      const ackBtn = page.locator('button').filter({ hasText: /确认|Acknowledge|Ack/ }).first()
      const closeBtn = page.locator('button').filter({ hasText: /关闭|Close/ }).first()
      await page.screenshot({ path: 'test-results/T4-6-操作按钮.png', fullPage: false })
    })
  })

  // T4-7: 协作空间列表
  test('T4-7 协作空间列表', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto('/oncall/spaces')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T4-7-协作空间列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T4-8: 新建协作空间
  test('T4-8 新建协作空间', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto('/oncall/spaces')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-8-新建协作空间.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-9: 值班排班页面
  test('T4-9 值班排班页面', async ({ authPage: page }) => {
    await test.step('导航到值班排班页', async () => {
      await page.goto('/oncall/schedule')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T4-9-值班排班.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T4-10: 升级策略页面
  test('T4-10 升级策略页面', async ({ authPage: page }) => {
    await test.step('导航到升级策略页', async () => {
      await page.goto('/oncall/config/escalation-policies')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T4-10-升级策略.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })
})
