import { Page } from '@playwright/test'

/**
 * API 辅助工具 — 使用 Playwright 内置 API 调用后端
 */
export class API {
  private static async request(page: Page, method: string, path: string, body?: unknown): Promise<any> {
    // 从 localStorage 获取 token
    const token = await page.evaluate(() => localStorage.getItem('token'))
    if (!token) {
      throw new Error('No auth token in localStorage - user not logged in')
    }

    const url = `http://localhost:3000${path}`
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    }

    try {
      let response
      const bodyStr = body ? JSON.stringify(body) : undefined

      switch (method) {
        case 'GET':
          response = await page.request.get(url, { headers })
          break
        case 'POST':
          response = await page.request.post(url, { headers, data: body })
          break
        case 'PUT':
          response = await page.request.put(url, { headers, data: body })
          break
        case 'DELETE':
          response = await page.request.delete(url, { headers })
          break
        case 'PATCH':
          response = await page.request.patch(url, { headers, data: body })
          break
        default:
          throw new Error(`Unsupported method: ${method}`)
      }

      const text = await response.text()
      try {
        return JSON.parse(text)
      } catch {
        return { code: response.status(), message: text || response.statusText() }
      }
    } catch (e) {
      return { code: -1, message: String(e) }
    }
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
