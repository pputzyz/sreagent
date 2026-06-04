import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// 通知功能测试

test.describe('通知功能测试', () => {

  test('NOTIF-1 通知渠道列表', async ({ authPage: page }) => {
    await test.step('获取渠道列表', async () => {
      const resp = await API.get(page, '/api/v1/notify-media')
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
    })
  })

  test('NOTIF-2 消息模板列表', async ({ authPage: page }) => {
    await test.step('获取模板列表', async () => {
      const resp = await API.get(page, '/api/v1/message-templates')
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
    })
  })

  test('NOTIF-3 通知规则列表', async ({ authPage: page }) => {
    await test.step('获取规则列表', async () => {
      const resp = await API.get(page, '/api/v1/notify-rules')
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
    })
  })
})
