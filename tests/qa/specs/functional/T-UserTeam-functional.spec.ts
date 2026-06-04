import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// 用户与团队功能测试

test.describe('用户与团队功能测试', () => {

  test('USER-1 用户列表', async ({ authPage: page }) => {
    await test.step('获取用户列表', async () => {
      const resp = await API.get(page, '/api/v1/users')
      expect(resp.code).toBe(0)
      expect(resp.data.list.length).toBeGreaterThan(0)
    })
  })

  test('USER-2 当前用户信息', async ({ authPage: page }) => {
    await test.step('获取当前用户', async () => {
      const resp = await API.get(page, '/api/v1/auth/profile')
      expect(resp.code).toBe(0)
      expect(resp.data.username).toBe('admin')
    })
  })

  test('USER-3 团队列表', async ({ authPage: page }) => {
    await test.step('获取团队列表', async () => {
      const resp = await API.get(page, '/api/v1/teams')
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
    })
  })
})
