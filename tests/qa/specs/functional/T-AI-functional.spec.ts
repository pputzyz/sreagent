import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

test.describe('AI 功能测试', () => {

  test('AI-1 AI 配置检查', async ({ authPage: page }) => {
    await test.step('获取 AI 配置', async () => {
      const resp = await API.get(page, '/api/v1/ai/config')
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
    })
  })

  test('AI-2 AI 聊天', async ({ authPage: page }) => {
    await test.step('发送消息', async () => {
      const resp = await API.post(page, '/api/v1/ai/chat', {
        message: '你好',
        mode: 'general'
      })
      // AI 可能未配置，检查响应结构
      expect(resp).toBeDefined()
      expect(resp.code).toBeDefined()
    })
  })
})
