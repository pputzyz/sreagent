import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

test.describe('告警规则功能测试', () => {

  let ruleId: number

  test('AR-1 创建告警规则', async ({ authPage: page }) => {
    await test.step('获取数据源', async () => {
      const ds = await API.get(page, '/api/v1/datasources')
      expect(ds.code).toBe(0)
    })

    await test.step('创建新规则', async () => {
      const resp = await API.post(page, '/api/v1/alert-rules', {
        name: '测试规则-' + Date.now(),
        expression: 'up == 0',
        severity: 'critical',
        for_duration: '5m',
        datasource_type: 'prometheus'
      })
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      ruleId = resp.data.id
    })
  })

  test('AR-2 获取规则详情', async ({ authPage: page }) => {
    await test.step('获取规则', async () => {
      const resp = await API.get(page, `/api/v1/alert-rules/${ruleId}`)
      expect(resp.code).toBe(0)
      expect(resp.data.name).toContain('测试规则')
    })
  })

  test('AR-3 更新规则', async ({ authPage: page }) => {
    await test.step('修改规则', async () => {
      const resp = await API.put(page, `/api/v1/alert-rules/${ruleId}`, {
        name: '测试规则-已更新',
        expression: 'up == 0',
        severity: 'warning',
        for_duration: '10m',
        datasource_type: 'prometheus'
      })
      expect(resp.code).toBe(0)
    })
  })

  test('AR-4 删除规则', async ({ authPage: page }) => {
    await test.step('删除规则', async () => {
      const resp = await API.del(page, `/api/v1/alert-rules/${ruleId}`)
      expect(resp.code).toBe(0)
    })
  })
})
