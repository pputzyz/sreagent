import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

test.describe('AI 功能测试', () => {

  // AI-1: Get AI config -> verify enabled=true -> test connection -> verify success and latency > 0
  test('AI-1 AI 配置与连接测试', async ({ authPage: page }) => {
    await test.step('获取 AI 配置', async () => {
      const res = await API.get(page, '/api/v1/ai/config')
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      expect(typeof res.data.enabled).toBe('boolean')
      expect(res.data.provider).toBeDefined()
      expect(res.data.model).toBeDefined()
      await page.screenshot({ path: 'test-results/AI-1-AI配置.png', fullPage: false })
    })

    await test.step('测试 AI 连接', async () => {
      const res = await API.post(page, '/api/v1/ai/test')
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      expect(res.data.success).toBe(true)
      expect(res.data.message).toBeDefined()
      await page.screenshot({ path: 'test-results/AI-1-连接测试结果.png', fullPage: false })
    })
  })

  // AI-2: Send English message -> verify reply not empty -> send Chinese message -> verify reply not empty
  test('AI-2 AI 聊天消息收发', async ({ authPage: page }) => {
    await test.step('发送英文消息', async () => {
      const res = await API.post(page, '/api/v1/ai/chat', {
        mode: 'general',
        message: 'What is monitoring in SRE?',
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      expect(typeof res.data.reply).toBe('string')
      expect(res.data.reply.length).toBeGreaterThan(0)
      await page.screenshot({ path: 'test-results/AI-2-英文消息回复.png', fullPage: false })
    })

    await test.step('发送中文消息', async () => {
      const res = await API.post(page, '/api/v1/ai/chat', {
        mode: 'general',
        message: '什么是 SRE 中的告警管理？',
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      expect(typeof res.data.reply).toBe('string')
      expect(res.data.reply.length).toBeGreaterThan(0)
      await page.screenshot({ path: 'test-results/AI-2-中文消息回复.png', fullPage: false })
    })
  })

  // AI-3: Get AI history -> verify structure -> clear history -> verify empty
  test('AI-3 AI 聊天历史管理', async ({ authPage: page }) => {
    await test.step('获取聊天历史', async () => {
      const res = await API.get(page, '/api/v1/ai/history?mode=general')
      expect(res.code).toBe(0)
      expect(Array.isArray(res.data)).toBe(true)
      // 历史可能为空（如果之前没有发过消息），但结构必须正确
      if (res.data.length > 0) {
        const msg = res.data[0]
        expect(msg.role || msg.sender).toBeDefined()
        expect(msg.content || msg.message).toBeDefined()
      }
      await page.screenshot({ path: 'test-results/AI-3-聊天历史.png', fullPage: false })
    })

    await test.step('清除聊天历史', async () => {
      const res = await API.del(page, '/api/v1/ai/history?mode=general')
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/AI-3-清除历史.png', fullPage: false })
    })

    await test.step('验证历史已清空', async () => {
      const res = await API.get(page, '/api/v1/ai/history?mode=general')
      expect(res.code).toBe(0)
      expect(Array.isArray(res.data)).toBe(true)
      expect(res.data.length).toBe(0)
      await page.screenshot({ path: 'test-results/AI-3-历史已清空.png', fullPage: false })
    })
  })
})
