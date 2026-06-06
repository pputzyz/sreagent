import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// 真正的功能测试 — 验证功能是否真正工作

test.describe('真实功能测试', () => {

  // 测试 1: 创建告警规则并验证数据库
  test('REAL-1 创建告警规则并验证', async ({ authPage: page }) => {
    let ruleId: number

    await test.step('创建告警规则', async () => {
      const resp = await API.post(page, '/api/v1/alert-rules', {
        name: '真实测试规则-' + Date.now(),
        expression: 'up == 0',
        severity: 'critical',
        for_duration: '5m',
        datasource_type: 'prometheus',
        labels: { env: 'test', team: 'sre' },
        annotations: { summary: '测试告警' }
      })
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      expect(resp.data.id).toBeGreaterThan(0)
      ruleId = resp.data.id
      expect(resp.data.name).toContain('真实测试规则')
      expect(resp.data.severity).toBe('critical')
      expect(resp.data.expression).toBe('up == 0')
    })

    await test.step('验证规则已保存到数据库', async () => {
      const resp = await API.get(page, `/api/v1/alert-rules/${ruleId}`)
      expect(resp.code).toBe(0)
      expect(resp.data.name).toContain('真实测试规则')
      expect(resp.data.severity).toBe('critical')
      expect(resp.data.expression).toBe('up == 0')
      expect(resp.data.labels.env).toBe('test')
      expect(resp.data.labels.team).toBe('sre')
    })

    await test.step('更新规则并验证', async () => {
      const resp = await API.put(page, `/api/v1/alert-rules/${ruleId}`, {
        name: '真实测试规则-已更新',
        expression: 'up == 0',
        severity: 'warning',
        for_duration: '10m',
        datasource_type: 'prometheus'
      })
      expect(resp.code).toBe(0)

      const getResp = await API.get(page, `/api/v1/alert-rules/${ruleId}`)
      expect(getResp.data.name).toBe('真实测试规则-已更新')
      expect(getResp.data.severity).toBe('warning')
    })

    await test.step('删除规则并验证', async () => {
      const resp = await API.del(page, `/api/v1/alert-rules/${ruleId}`)
      expect(resp.code).toBe(0)

      const getResp = await API.get(page, `/api/v1/alert-rules/${ruleId}`)
      expect(getResp.code).not.toBe(0) // 应该返回错误
    })
  })

  // 测试 2: 创建通知渠道并验证
  test('REAL-2 创建通知渠道并验证', async ({ authPage: page }) => {
    let channelId: number

    await test.step('创建 Webhook 渠道', async () => {
      const resp = await API.post(page, '/api/v1/notify-media', {
        name: '测试Webhook-' + Date.now(),
        type: 'webhook',
        config: JSON.stringify({ url: 'https://httpbin.org/post' }),
        is_enabled: true
      })
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      channelId = resp.data.id
    })

    await test.step('验证渠道已保存', async () => {
      const resp = await API.get(page, `/api/v1/notify-media/${channelId}`)
      expect(resp.code).toBe(0)
      expect(resp.data.name).toContain('测试Webhook')
      expect(resp.data.type).toBe('webhook')
    })

    await test.step('删除渠道', async () => {
      const resp = await API.del(page, `/api/v1/notify-media/${channelId}`)
      expect(resp.code).toBe(0)
    })
  })

  // 测试 3: 创建团队并验证
  test('REAL-3 创建团队并验证', async ({ authPage: page }) => {
    let teamId: number

    await test.step('创建团队', async () => {
      const resp = await API.post(page, '/api/v1/teams', {
        name: '测试团队-' + Date.now(),
        description: '自动化测试创建的团队'
      })
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      teamId = resp.data.id
    })

    await test.step('验证团队已保存', async () => {
      const resp = await API.get(page, `/api/v1/teams/${teamId}`)
      expect(resp.code).toBe(0)
      expect(resp.data.name).toContain('测试团队')
    })

    await test.step('删除团队', async () => {
      const resp = await API.del(page, `/api/v1/teams/${teamId}`)
      expect(resp.code).toBe(0)
    })
  })

  // 测试 4: 创建故障并验证状态流转
  test('REAL-4 创建故障并验证状态流转', async ({ authPage: page }) => {
    let incidentId: number

    await test.step('获取渠道列表', async () => {
      const resp = await API.get(page, '/api/v1/channels')
      expect(resp.code).toBe(0)
      expect(resp.data.list.length).toBeGreaterThan(0)
    })

    await test.step('创建故障', async () => {
      const resp = await API.post(page, '/api/v1/incidents', {
        title: '测试故障-' + Date.now(),
        severity: 'critical',
        channel_id: 1,
        description: '自动化测试创建的故障'
      })
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      incidentId = resp.data.id
      expect(resp.data.status).toBe('triggered')
    })

    await test.step('确认故障', async () => {
      const resp = await API.post(page, `/api/v1/incidents/${incidentId}/acknowledge`, {
        note: '已确认'
      })
      expect(resp.code).toBe(0)

      const getResp = await API.get(page, `/api/v1/incidents/${incidentId}`)
      expect(getResp.data.status).toBe('processing')
    })

    await test.step('关闭故障', async () => {
      const resp = await API.post(page, `/api/v1/incidents/${incidentId}/close`, {
        resolution: '问题已解决'
      })
      expect(resp.code).toBe(0)

      const getResp = await API.get(page, `/api/v1/incidents/${incidentId}`)
      expect(getResp.data.status).toBe('closed')
    })
  })

  // 测试 5: LLM 配置和 AI 聊天
  test('REAL-5 LLM 配置和 AI 聊天', async ({ authPage: page }) => {
    await test.step('获取 AI 配置', async () => {
      const resp = await API.get(page, '/api/v1/ai/config')
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      expect(resp.data.enabled).toBe(true)
    })

    await test.step('测试 LLM 连接', async () => {
      const resp = await API.post(page, '/api/v1/ai/test')
      expect(resp.code).toBe(0)
      expect(resp.data.success).toBe(true)
      expect(resp.data.latency_ms).toBeGreaterThan(0)
    })

    await test.step('AI 聊天', async () => {
      const resp = await API.post(page, '/api/v1/ai/chat', {
        message: 'What is 1+1?',
        mode: 'general'
      })
      expect(resp.code).toBe(0)
      expect(resp.data.reply).toBeTruthy()
      expect(resp.data.reply.length).toBeGreaterThan(10)
    })
  })

  // 测试 6: 数据源健康检查
  test('REAL-6 数据源健康检查', async ({ authPage: page }) => {
    let datasourceId: number

    await test.step('获取数据源列表', async () => {
      const resp = await API.get(page, '/api/v1/datasources')
      expect(resp.code).toBe(0)
      expect(resp.data.list.length).toBeGreaterThan(0)
      datasourceId = resp.data.list[0].id
    })

    await test.step('健康检查', async () => {
      const resp = await API.post(page, `/api/v1/datasources/${datasourceId}/health-check`)
      expect(resp.code).toBe(0)
      expect(resp.data.status).toBeDefined()
    })

    await test.step('标签查询', async () => {
      const resp = await API.get(page, `/api/v1/datasources/${datasourceId}/labels/keys`)
      expect(resp.code).toBe(0)
    })

    await test.step('指标查询', async () => {
      const resp = await API.get(page, `/api/v1/datasources/${datasourceId}/metrics`)
      expect(resp.code).toBe(0)
    })
  })

  // 测试 7: 用户权限验证
  test('REAL-7 用户权限验证', async ({ authPage: page }) => {
    await test.step('获取当前用户', async () => {
      const resp = await API.get(page, '/api/v1/auth/profile')
      expect(resp.code).toBe(0)
      expect(resp.data.username).toBe('admin')
      expect(resp.data.role).toBe('admin')
    })

    await test.step('获取权限列表', async () => {
      const resp = await API.get(page, '/api/v1/me/permissions')
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
    })
  })

  // 测试 8: 排班功能验证
  test('REAL-8 排班功能验证', async ({ authPage: page }) => {
    let scheduleId: number

    await test.step('创建排班', async () => {
      const resp = await API.post(page, '/api/v1/schedules', {
        name: '测试排班-' + Date.now(),
        rotation_type: 'daily',
        timezone: 'Asia/Shanghai',
        handoff_time: '09:00',
        is_enabled: true
      })
      expect(resp.code).toBe(0)
      scheduleId = resp.data.id
    })

    await test.step('验证排班已保存', async () => {
      const resp = await API.get(page, `/api/v1/schedules/${scheduleId}`)
      expect(resp.code).toBe(0)
      expect(resp.data.name).toContain('测试排班')
      expect(resp.data.rotation_type).toBe('daily')
    })

    await test.step('删除排班', async () => {
      const resp = await API.del(page, `/api/v1/schedules/${scheduleId}`)
      expect(resp.code).toBe(0)
    })
  })
})
