import { test, expect } from '../fixtures/auth'

// T8: AI 助手 — 冒烟测试

test.describe('T8 - AI 助手', () => {

  test('T8-1 AI Agent 页', async ({ authenticatedPage: page }) => {
    await page.goto('/ai/agent')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })

  test('T8-2 AI 配置页', async ({ authenticatedPage: page }) => {
    await page.goto('/platform/ai-config')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })
})
