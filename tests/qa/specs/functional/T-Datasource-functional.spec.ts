import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

test.describe('数据源功能测试', () => {

  let datasourceId: number

  test('DS-1 数据源列表', async ({ authPage: page }) => {
    await test.step('获取数据源列表', async () => {
      const resp = await API.get(page, '/api/v1/datasources')
      expect(resp.code).toBe(0)
      expect(resp.data.list).toBeDefined()
      expect(resp.data.list.length).toBeGreaterThan(0)
      datasourceId = resp.data.list[0].id
    })
  })

  test('DS-2 数据源详情', async ({ authPage: page }) => {
    await test.step('获取数据源详情', async () => {
      const resp = await API.get(page, `/api/v1/datasources/${datasourceId}`)
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
    })
  })

  test('DS-3 测试连接', async ({ authPage: page }) => {
    await test.step('测试数据源连接', async () => {
      const resp = await API.post(page, `/api/v1/datasources/test-connection`, {
        type: 'victoriametrics',
        endpoint: 'http://localhost:8428',
        auth_type: 'none',
        auth_config: ''
      })
      expect(resp).toBeDefined()
      expect(resp.code).toBeDefined()
    })
  })
})
