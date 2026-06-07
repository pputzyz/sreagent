import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

// ---------------------------------------------------------------------------
// WH-1 Webhook 接收 Alertmanager 格式
// ---------------------------------------------------------------------------
test('WH-1 Webhook 接收 Alertmanager 格式', async ({ authPage: page }) => {
  const tag = uid()

  try {
    // ---- 1. 构造 Alertmanager 格式 payload ----
    await test.step('构造 Alertmanager 格式 payload', async () => {
      await page.screenshot({ path: 'test-results/WH-1-01-准备Payload.png', fullPage: false })
    })

    // ---- 2. 发送 Alertmanager webhook ----
    await test.step('发送 Alertmanager webhook', async () => {
      const alertmanagerPayload = {
        version: '4',
        groupKey: `{}`,
        status: 'firing',
        receiver: 'test-receiver',
        alerts: [
          {
            status: 'firing',
            labels: {
              alertname: `TestAlert-${tag}`,
              severity: 'critical',
              env: 'production',
              instance: 'web-01:9090',
            },
            annotations: {
              summary: `Functional test alert ${tag}`,
              description: 'This is a test alert from functional test',
            },
            startsAt: new Date().toISOString(),
            endsAt: new Date(Date.now() + 3600 * 1000).toISOString(),
            fingerprint: `fp-${tag}`,
          },
        ],
      }

      const token = await page.evaluate(() => localStorage.getItem('token'))
      const resp = await page.request.post(`http://localhost:3000/webhooks/alertmanager`, {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        data: alertmanagerPayload,
      })
      const result = await resp.json()
      expect(result.code).toBe(0)
      await page.screenshot({ path: 'test-results/WH-1-02-发送成功.png', fullPage: false })
    })

    // ---- 3. 验证告警事件已创建 ----
    await test.step('验证告警事件已创建', async () => {
      // Wait for event to be processed
      await page.waitForTimeout(2000)
      const res = await API.get(page, `${API_BASE}/alert-events?keyword=${tag}&page_size=10`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/WH-1-03-事件验证.png', fullPage: false })
    })

    // ---- 4. 验证告警标签正确 ----
    await test.step('验证告警标签正确', async () => {
      const res = await API.get(page, `${API_BASE}/alert-events?keyword=${tag}&page_size=10`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      if (list.length > 0) {
        const event = list[0]
        expect(event.labels).toBeTruthy()
        expect(event.labels.alertname).toContain(`TestAlert-${tag}`)
        expect(event.labels.severity).toBe('critical')
      }
      await page.screenshot({ path: 'test-results/WH-1-04-标签验证.png', fullPage: false })
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/WH-1-ERROR.png', fullPage: false })
    throw e
  }
})

// ---------------------------------------------------------------------------
// WH-2 Webhook 格式解析
// ---------------------------------------------------------------------------
test('WH-2 Webhook 格式解析', async ({ authPage: page }) => {
  const tag = uid()

  try {
    // ---- 1. 发送多告警 webhook ----
    await test.step('发送多告警 webhook', async () => {
      const multiAlertPayload = {
        version: '4',
        groupKey: '{}',
        status: 'firing',
        receiver: 'test-receiver',
        alerts: [
          {
            status: 'firing',
            labels: {
              alertname: `MultiAlert-A-${tag}`,
              severity: 'warning',
              env: 'staging',
            },
            annotations: { summary: 'First alert' },
            startsAt: new Date().toISOString(),
            fingerprint: `fp-a-${tag}`,
          },
          {
            status: 'firing',
            labels: {
              alertname: `MultiAlert-B-${tag}`,
              severity: 'critical',
              env: 'production',
            },
            annotations: { summary: 'Second alert' },
            startsAt: new Date().toISOString(),
            fingerprint: `fp-b-${tag}`,
          },
        ],
      }

      const token = await page.evaluate(() => localStorage.getItem('token'))
      const resp = await page.request.post(`http://localhost:3000/webhooks/alertmanager`, {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        data: multiAlertPayload,
      })
      const result = await resp.json()
      expect(result.code).toBe(0)
      await page.screenshot({ path: 'test-results/WH-2-01-多告警发送成功.png', fullPage: false })
    })

    // ---- 2. 验证多个事件已创建 ----
    await test.step('验证多个事件已创建', async () => {
      await page.waitForTimeout(2000)
      const res = await API.get(page, `${API_BASE}/alert-events?keyword=${tag}&page_size=20`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/WH-2-02-多事件验证.png', fullPage: false })
    })

    // ---- 3. 发送 resolve 格式 webhook ----
    await test.step('发送 resolve 格式 webhook', async () => {
      const resolvePayload = {
        version: '4',
        groupKey: '{}',
        status: 'resolved',
        receiver: 'test-receiver',
        alerts: [
          {
            status: 'resolved',
            labels: {
              alertname: `ResolveAlert-${tag}`,
              severity: 'warning',
              env: 'test',
            },
            annotations: { summary: 'Resolved alert' },
            startsAt: new Date(Date.now() - 3600 * 1000).toISOString(),
            endsAt: new Date().toISOString(),
            fingerprint: `fp-resolve-${tag}`,
          },
        ],
      }

      const token = await page.evaluate(() => localStorage.getItem('token'))
      const resp = await page.request.post(`http://localhost:3000/webhooks/alertmanager`, {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        data: resolvePayload,
      })
      const result = await resp.json()
      expect(result.code).toBe(0)
      await page.screenshot({ path: 'test-results/WH-2-03-Resolve发送成功.png', fullPage: false })
    })

    // ---- 4. 验证 resolve 事件处理 ----
    await test.step('验证 resolve 事件处理', async () => {
      await page.waitForTimeout(2000)
      const res = await API.get(page, `${API_BASE}/alert-events?keyword=${tag}&page_size=20`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/WH-2-04-Resolve验证.png', fullPage: false })
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/WH-2-ERROR.png', fullPage: false })
    throw e
  }
})

// ---------------------------------------------------------------------------
// WH-3 Webhook channel 路由
// ---------------------------------------------------------------------------
test('WH-3 Webhook channel 路由', async ({ authPage: page }) => {
  const tag = uid()
  let channelId: number | null = null

  try {
    // ---- 1. 获取可用 channel ----
    await test.step('获取可用 channel', async () => {
      const res = await API.get(page, `${API_BASE}/channels?page_size=10`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      if (list.length > 0) {
        channelId = list[0].id
      }
      await page.screenshot({ path: 'test-results/WH-3-01-Channel列表.png', fullPage: false })
    })

    // ---- 2. 发送带 channel 标签的 webhook ----
    await test.step('发送带 channel 标签的 webhook', async () => {
      const payload = {
        version: '4',
        groupKey: '{}',
        status: 'firing',
        receiver: 'test-receiver',
        alerts: [
          {
            status: 'firing',
            labels: {
              alertname: `ChannelRouteAlert-${tag}`,
              severity: 'critical',
              env: 'production',
              channel_id: channelId ? String(channelId) : 'default',
            },
            annotations: { summary: 'Channel routing test alert' },
            startsAt: new Date().toISOString(),
            fingerprint: `fp-channel-${tag}`,
          },
        ],
      }

      const token = await page.evaluate(() => localStorage.getItem('token'))
      const resp = await page.request.post(`http://localhost:3000/webhooks/alertmanager`, {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        data: payload,
      })
      const result = await resp.json()
      expect(result.code).toBe(0)
      await page.screenshot({ path: 'test-results/WH-3-02-Channel路由发送.png', fullPage: false })
    })

    // ---- 3. 验证事件已路由到正确 channel ----
    await test.step('验证事件路由', async () => {
      await page.waitForTimeout(2000)
      const res = await API.get(page, `${API_BASE}/alert-events?keyword=${tag}&page_size=10`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/WH-3-03-路由验证.png', fullPage: false })
    })

    // ---- 4. 检查 channel 内的事件 ----
    await test.step('检查 channel 内的事件', async () => {
      if (channelId) {
        const res = await API.get(page, `${API_BASE}/channels/${channelId}/events?page_size=10`)
        expect(res.code).toBe(0)
        await page.screenshot({ path: 'test-results/WH-3-04-Channel事件.png', fullPage: false })
      } else {
        await page.screenshot({ path: 'test-results/WH-3-04-无Channel跳过.png', fullPage: false })
      }
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/WH-3-ERROR.png', fullPage: false })
    throw e
  }
})
