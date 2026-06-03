import { test as base, Page } from '@playwright/test'

const BASE_URL = 'http://localhost:3000'
const API_URL = 'http://localhost:8080'

export const test = base.extend<{ authPage: Page }>({
  authPage: async ({ page }, use) => {
    // API 登录获取 token（绕过 UI 验证码/限流）
    const res = await fetch(`${API_URL}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: 'admin', password: 'admin123' }),
    })
    const data = await res.json()
    if (data.code !== 0) throw new Error(`Login failed: ${data.message}`)
    const token = data.data.token

    // 注入 token 到浏览器
    await page.goto(BASE_URL)
    await page.evaluate((t) => {
      localStorage.setItem('token', t)
      localStorage.setItem('user_role', 'admin')
    }, token)

    // 导航到首页触发 auth store 初始化
    await page.goto(BASE_URL + '/')
    await page.waitForLoadState('networkidle')

    // 等待侧边栏出现（确认登录成功）
    await page.locator('nav, [class*="sidebar"], [class*="rail"]').first()
      .waitFor({ state: 'visible', timeout: 15000 })
      .catch(() => {})

    await use(page)
  },
})

export { expect } from '@playwright/test'
