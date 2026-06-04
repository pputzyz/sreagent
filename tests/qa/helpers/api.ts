import { Page } from '@playwright/test'

/**
 * API 辅助工具 — 通过前端代理调用后端 API
 */
export class API {
  private static async request(page: Page, method: string, path: string, body?: unknown): Promise<any> {
    // 等待 token 存在
    const token = await page.evaluate(() => localStorage.getItem('token'))
    if (!token) {
      throw new Error('No auth token in localStorage - user not logged in')
    }

    const resp = await page.evaluate(async ({ method, url, body, token }) => {
      try {
        const options: RequestInit = {
          method,
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
          }
        }
        if (body && method !== 'GET') {
          options.body = JSON.stringify(body)
        }
        const res = await fetch(url, options)
        const text = await res.text()
        try {
          return JSON.parse(text)
        } catch {
          return { code: res.status, message: text || res.statusText }
        }
      } catch (e) {
        return { code: -1, message: String(e) }
      }
    }, { method, url: path, body, token })
    return resp
  }

  static async post(page: Page, path: string, body?: unknown): Promise<any> {
    return this.request(page, 'POST', path, body)
  }

  static async get(page: Page, path: string): Promise<any> {
    return this.request(page, 'GET', path)
  }

  static async put(page: Page, path: string, body?: unknown): Promise<any> {
    return this.request(page, 'PUT', path, body)
  }

  static async del(page: Page, path: string): Promise<any> {
    return this.request(page, 'DELETE', path)
  }

  static async patch(page: Page, path: string, body?: unknown): Promise<any> {
    return this.request(page, 'PATCH', path, body)
  }
}
