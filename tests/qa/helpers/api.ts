import { Page } from '@playwright/test'

/**
 * API 辅助工具 — 使用 Playwright 内置 API 调用后端
 */
export class API {
  private static async request(page: Page, method: string, path: string, body?: unknown, timeout?: number): Promise<any> {
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

    const options: any = { headers, timeout: timeout || 60000 }

    try {
      let response

      switch (method) {
        case 'GET':
          response = await page.request.get(url, options)
          break
        case 'POST':
          response = await page.request.post(url, { ...options, data: body })
          break
        case 'PUT':
          response = await page.request.put(url, { ...options, data: body })
          break
        case 'DELETE':
          response = await page.request.delete(url, options)
          break
        case 'PATCH':
          response = await page.request.patch(url, { ...options, data: body })
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

  static async post(page: Page, path: string, body?: unknown, timeout?: number): Promise<any> {
    return this.request(page, 'POST', path, body, timeout)
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
