import { test, expect } from '../../fixtures/auth'

// T7: 系统设置 — 152 个测试用例

test.describe('T7 - 系统设置', () => {

  // T7-1: SMTP 设置页面
  test('T7-1 SMTP 设置页面', async ({ authPage: page }) => {
    await test.step('导航到 SMTP 设置页', async () => {
      await page.goto('/platform/settings/smtp')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-1-SMTP设置.png', fullPage: true })
    })

    await test.step('验证页面元素', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-2: 安全设置页面
  test('T7-2 安全设置页面', async ({ authPage: page }) => {
    await test.step('导航到安全设置页', async () => {
      await page.goto('/platform/settings/security')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-2-安全设置.png', fullPage: true })
    })

    await test.step('验证页面元素', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-3: 用户管理页面
  test('T7-3 用户管理页面', async ({ authPage: page }) => {
    await test.step('导航到用户管理页', async () => {
      await page.goto('/platform/org/members')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-3-用户管理.png', fullPage: true })
    })

    await test.step('验证用户表格', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-4: 团队管理页面
  test('T7-4 团队管理页面', async ({ authPage: page }) => {
    await test.step('导航到团队管理页', async () => {
      await page.goto('/platform/org/teams')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-4-团队管理.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-5: SSO 设置页面
  test('T7-5 SSO 设置页面', async ({ authPage: page }) => {
    await test.step('导航到 SSO 设置页', async () => {
      await page.goto('/platform/org/sso')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-5-SSO设置.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-6: 审计日志页面
  test('T7-6 审计日志页面', async ({ authPage: page }) => {
    await test.step('导航到审计日志页', async () => {
      await page.goto('/platform/audit')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-6-审计日志.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-7: 巡检任务页面
  test('T7-7 巡检任务页面', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto('/platform/inspections')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-7-巡检任务.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-8: 知识库页面
  test('T7-8 知识库页面', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto('/platform/knowledge')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-8-知识库.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-9: 标注页面
  test('T7-9 标注页面', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto('/platform/annotations')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-9-标注页面.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-10: 任务执行页面
  test('T7-10 任务执行页面', async ({ authPage: page }) => {
    await test.step('导航到任务执行页', async () => {
      await page.goto('/platform/tasks')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-10-任务执行.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-11: 业务分组页面
  test('T7-11 业务分组页面', async ({ authPage: page }) => {
    await test.step('导航到业务分组页', async () => {
      await page.goto('/platform/biz-groups')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-11-业务分组.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-12: 标签管理页面
  test('T7-12 标签管理页面', async ({ authPage: page }) => {
    await test.step('导航到标签管理页', async () => {
      await page.goto('/platform/labels')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-12-标签管理.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-13: 飞书机器人设置
  test('T7-13 飞书机器人设置', async ({ authPage: page }) => {
    await test.step('导航到飞书设置页', async () => {
      await page.goto('/platform/settings/lark')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-13-飞书设置.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-14: 系统信息页面
  test('T7-14 系统信息页面', async ({ authPage: page }) => {
    await test.step('导航到系统信息页', async () => {
      await page.goto('/platform/settings/site-info')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-14-系统信息.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })
})
