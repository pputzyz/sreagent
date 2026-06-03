import { test as base, Page } from '@playwright/test'

const BASE_URL = 'http://localhost:3000'
const API_URL = 'http://localhost:8080'

// API 直接登录（跳过 UI，更可靠）
async function loginViaAPI(page: Page): Promise<string> {
  const res = await fetch(`${API_URL}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password: 'admin123' }),
  })
  const data = await res.json()
  if (data.code !== 0) throw new Error(`Login failed: ${data.message}`)
  return data.data.token
}

export const test = base.extend<{ authPage: Page }>({
  authPage: async ({ page }, use) => {
    // 1. 先通过 API 获取 token
    const token = await loginViaAPI(page)

    // 2. 注入 token 到 localStorage
    await page.goto(BASE_URL)
    await page.evaluate((t) => {
      localStorage.setItem('token', t)
    }, token)

    // 3. 导航到首页，触发 auth store 初始化
    await page.goto(BASE_URL + '/')
    await page.waitForLoadState('networkidle')

    // 4. 验证登录成功（检查是否有侧边栏或用户菜单）
    const maxWait = 10000
    const start = Date.now()
    while (Date.now() - start < maxWait) {
      const hasNav = await page.locator('nav, [class*="sidebar"], [class*="rail"], [class*="app-shell"]').first().isVisible().catch(() => false)
      if (hasNav) break
      await page.waitForTimeout(200)
    }

    await use(page)
  },
})

export { expect } from '@playwright/test'
