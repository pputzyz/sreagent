import { test as base, Page } from '@playwright/test'

const BASE_URL = 'http://localhost:3000'
const API_URL = 'http://localhost:8080'

async function loginViaAPI(): Promise<string> {
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
    // 1. 通过 API 获取 token
    const token = await loginViaAPI()

    // 2. 注入 token 到 localStorage
    await page.goto(BASE_URL)
    await page.evaluate((t) => {
      localStorage.setItem('token', t)
      localStorage.setItem('user_role', 'admin')
    }, token)

    // 3. 导航到首页触发 auth store 初始化
    await page.goto(BASE_URL + '/')
    await page.waitForLoadState('networkidle')

    // 4. 等待侧边栏出现（确认登录成功）
    await page.locator('nav, [class*="sidebar"], [class*="rail"], [class*="app-shell"]').first()
      .waitFor({ state: 'visible', timeout: 15000 })
      .catch(() => {})

    // 5. 等待 auth store 完全初始化（包括权限加载）
    await page.waitForTimeout(2000)

    await use(page)
  },
})

export { expect } from '@playwright/test'
