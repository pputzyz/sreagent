import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

test.describe('告警规则功能测试', () => {

  test('AR-1 告警规则列表', async ({ authPage: page }) => {
    await test.step('获取规则列表', async () => {
      const resp = await API.get(page, '/api/v1/alert-rules')
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      expect(resp.data.list).toBeDefined()
    })
  })

  test('AR-2 获取规则详情', async ({ authPage: page }) => {
    await test.step('获取第一条规则', async () => {
      const listResp = await API.get(page, '/api/v1/alert-rules')
      expect(listResp.code).toBe(0)
      if (listResp.data.list.length > 0) {
        const ruleId = listResp.data.list[0].id
        const resp = await API.get(page, `/api/v1/alert-rules/${ruleId}`)
        expect(resp.code).toBe(0)
        expect(resp.data).toBeDefined()
      }
    })
  })

  test('AR-3 告警规则分类', async ({ authPage: page }) => {
    await test.step('获取分类列表', async () => {
      const resp = await API.get(page, '/api/v1/alert-rules/categories')
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
    })
  })
})
