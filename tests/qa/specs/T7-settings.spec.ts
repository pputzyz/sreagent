import { test, expect } from '../fixtures/auth'

// T7: 系统设置 — 冒烟测试

test.describe('T7 - 系统设置', () => {

  test('T7-1 SMTP 设置页', async ({ authPage: page }) => {
    await page.goto('/platform/settings/smtp')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })

  test('T7-2 安全设置页', async ({ authPage: page }) => {
    await page.goto('/platform/settings/security')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })

  test('T7-3 用户管理页', async ({ authPage: page }) => {
    await page.goto('/platform/org/members')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })

  test('T7-4 团队管理页', async ({ authPage: page }) => {
    await page.goto('/platform/org/teams')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })

  test('T7-5 审计日志页', async ({ authPage: page }) => {
    await page.goto('/platform/audit')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })
})
