import { test, expect } from '../fixtures/auth'

// T9: 平台功能 — 冒烟测试

test.describe('T9 - 平台功能', () => {

  test('T9-1 巡检任务页', async ({ authenticatedPage: page }) => {
    await page.goto('/platform/inspections')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })

  test('T9-2 知识库页', async ({ authenticatedPage: page }) => {
    await page.goto('/platform/knowledge')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })

  test('T9-3 标注页', async ({ authenticatedPage: page }) => {
    await page.goto('/platform/annotations')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })

  test('T9-4 任务执行页', async ({ authenticatedPage: page }) => {
    await page.goto('/platform/tasks')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('body')).toBeVisible()
  })
})
