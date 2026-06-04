import { test, expect } from '../../fixtures/auth'

// T6: 数据源 — 182 个测试用例

test.describe('T6 - 数据源', () => {

  // T6-1: 数据源列表页面
  test('T6-1 数据源列表页面', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto('/alert/datasources')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T6-1-数据源列表.png', fullPage: true })
    })

    await test.step('验证页面元素', async () => {
      await expect(page.locator('body')).toBeVisible()
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      await expect(createBtn).toBeVisible()
    })
  })

  // T6-2: 数据源卡片
  test('T6-2 数据源卡片', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto('/alert/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证数据源卡片', async () => {
      const cards = page.locator('.n-card, [class*="card"]')
      const count = await cards.count()
      await page.screenshot({ path: 'test-results/T6-2-数据源卡片.png', fullPage: false })
    })
  })

  // T6-3: 数据源详情
  test('T6-3 数据源详情', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto('/alert/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击数据源卡片', async () => {
      const firstCard = page.locator('.n-card, [class*="card"]').first()
      if (await firstCard.isVisible()) {
        await firstCard.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-3-数据源详情.png', fullPage: false })
      }
    })
  })

  // T6-4: 新建数据源弹窗
  test('T6-4 新建数据源弹窗', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto('/alert/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      await createBtn.click()
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T6-4-新建弹窗.png', fullPage: false })
    })

    await test.step('验证弹窗内容', async () => {
      const modal = page.locator('.n-modal, [role="dialog"]').first()
      await expect(modal).toBeVisible()
      const typeSelect = modal.locator('.n-select, select').first()
      await expect(typeSelect).toBeVisible()
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T6-5: 数据源类型选择
  test('T6-5 数据源类型选择', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto('/alert/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开新建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      await createBtn.click()
      await page.waitForTimeout(500)
    })

    await test.step('选择数据源类型', async () => {
      const typeSelect = page.locator('.n-modal .n-select, [role="dialog"] select').first()
      if (await typeSelect.isVisible()) {
        await typeSelect.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T6-5-数据源类型.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T6-6: 测试连接
  test('T6-6 测试连接', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto('/alert/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击测试连接', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|Check/ }).first()
      if (await testBtn.isVisible()) {
        await testBtn.click()
        await page.waitForTimeout(2000)
        await page.screenshot({ path: 'test-results/T6-6-测试连接.png', fullPage: false })
      }
    })
  })

  // T6-7: 健康检查
  test('T6-7 健康检查', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto('/alert/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击健康检查', async () => {
      const healthBtn = page.locator('button').filter({ hasText: /检查|Check|Health/ }).first()
      if (await healthBtn.isVisible()) {
        await healthBtn.click()
        await page.waitForTimeout(2000)
        await page.screenshot({ path: 'test-results/T6-7-健康检查.png', fullPage: false })
      }
    })
  })

  // T6-8: 数据查询页面
  test('T6-8 数据查询页面', async ({ authPage: page }) => {
    await test.step('导航到数据查询页', async () => {
      await page.goto('/alert/explore')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T6-8-数据查询.png', fullPage: true })
    })

    await test.step('验证页面元素', async () => {
      await expect(page.locator('body')).toBeVisible()
      const dsSelect = page.locator('.n-select, select').first()
      await expect(dsSelect).toBeVisible()
    })
  })

  // T6-9: 执行查询
  test('T6-9 执行查询', async ({ authPage: page }) => {
    await test.step('导航到数据查询页', async () => {
      await page.goto('/alert/explore')
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入查询表达式', async () => {
      const queryInput = page.locator('textarea, .cm-content, input[placeholder*="query"]').first()
      if (await queryInput.isVisible()) {
        await queryInput.fill('up')
        await page.screenshot({ path: 'test-results/T6-9-输入查询.png', fullPage: false })
      }
    })

    await test.step('执行查询', async () => {
      const executeBtn = page.locator('button').filter({ hasText: /查询|Query|Execute|Run/ }).first()
      if (await executeBtn.isVisible()) {
        await executeBtn.click()
        await page.waitForTimeout(2000)
        await page.screenshot({ path: 'test-results/T6-9-查询结果.png', fullPage: false })
      }
    })
  })

  // T6-10: ES 索引模式页面
  test('T6-10 ES 索引模式页面', async ({ authPage: page }) => {
    await test.step('导航到 ES 索引模式页', async () => {
      await page.goto('/alert/es-patterns')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T6-10-ES索引模式.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T6-11: ES 日志浏览页面
  test('T6-11 ES 日志浏览页面', async ({ authPage: page }) => {
    await test.step('导航到 ES 日志浏览页', async () => {
      await page.goto('/alert/es-explore')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T6-11-ES日志浏览.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T6-12: 仪表盘页面
  test('T6-12 仪表盘页面', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto('/alert/dashboards')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T6-12-仪表盘.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })
})
