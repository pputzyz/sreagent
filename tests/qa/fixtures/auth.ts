import { test as base, Page } from '@playwright/test'

export const test = base.extend<{ authenticatedPage: Page }>({
  authenticatedPage: async ({ page }, use) => {
    // Login
    await page.goto('/login')
    await page.fill('input[placeholder*="admin"], input[type="text"]', 'admin')
    await page.fill('input[type="password"]', 'admin123')
    await page.click('button[type="submit"]')
    await page.waitForURL('**/dashboard**', { timeout: 15000 })
    await use(page)
  },
})

export { expect } from '@playwright/test'
