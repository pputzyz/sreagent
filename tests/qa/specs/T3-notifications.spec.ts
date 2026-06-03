import { test, expect } from '../fixtures/auth'

// T3: 通知管道 — 冒烟测试

test.describe('T3 - 通知管道', () => {

  test.beforeEach(async ({ authPage: page }) => {
    await page.goto('/oncall/notify/policies')
    await page.waitForLoadState('networkidle')
  })

  // T3-1: 通知策略列表
  test('T3-1 通知策略列表', async ({ authPage: page }) => {
    await expect(page.locator('body')).toBeVisible()
  })

  // T3-2: 新建通知策略
  test('T3-2 新建按钮', async ({ authPage: page }) => {
    const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
    if (await createBtn.isVisible()) {
      await expect(createBtn).toBeEnabled()
    }
  })
})

test.describe('T3 - 通知渠道', () => {

  test.beforeEach(async ({ authPage: page }) => {
    await page.goto('/oncall/notify/media')
    await page.waitForLoadState('networkidle')
  })

  // T3-10: 渠道列表
  test('T3-10 渠道列表', async ({ authPage: page }) => {
    await expect(page.locator('body')).toBeVisible()
  })
})

test.describe('T3 - 消息模板', () => {

  test.beforeEach(async ({ authPage: page }) => {
    await page.goto('/oncall/notify/templates')
    await page.waitForLoadState('networkidle')
  })

  // T3-20: 模板列表
  test('T3-20 模板列表', async ({ authPage: page }) => {
    await expect(page.locator('body')).toBeVisible()
  })
})
