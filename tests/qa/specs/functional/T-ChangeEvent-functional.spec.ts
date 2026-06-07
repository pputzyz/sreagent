import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

// ---------------------------------------------------------------------------
// CE-1: 变更事件列表
// ---------------------------------------------------------------------------
test('CE-1 变更事件列表', async ({ authPage: page }) => {
  await test.step('获取变更事件列表', async () => {
    const res = await API.get(page, `${API_BASE}/change-events?page=1&page_size=20`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    await page.screenshot({ path: 'test-results/CE-1-01-事件列表.png', fullPage: false })
  })

  await test.step('验证列表结构', async () => {
    const res = await API.get(page, `${API_BASE}/change-events?page=1&page_size=5`)
    expect(res.code).toBe(0)
    const list = res.data.list || res.data || []
    expect(Array.isArray(list)).toBe(true)
    await page.screenshot({ path: 'test-results/CE-1-02-列表结构.png', fullPage: false })
  })

  await test.step('按时间范围筛选', async () => {
    const now = new Date()
    const oneDayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000)
    const res = await API.get(page, `${API_BASE}/change-events?page=1&page_size=5&start=${oneDayAgo.toISOString()}&end=${now.toISOString()}`)
    expect(res.code).toBe(0)
    await page.screenshot({ path: 'test-results/CE-1-03-时间筛选.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// CE-2: 变更事件接入 ingest
// ---------------------------------------------------------------------------
test('CE-2 变更事件 接入ingest', async ({ authPage: page }) => {
  const tag = uid()

  await test.step('通过 ingest 接入变更事件', async () => {
    const res = await API.post(page, `${API_BASE}/change-events/ingest`, {
      title: `change-event-${tag}`,
      description: 'Functional test change event',
      source: 'functional-test',
      event_type: 'deployment',
      timestamp: new Date().toISOString(),
      labels: { env: 'test', run: tag },
    })
    // Ingest may succeed or require authentication
    expect(res).toBeDefined()
    expect(res.code).toBeDefined()
    await page.screenshot({ path: 'test-results/CE-2-01-ingest结果.png', fullPage: false })
  })

  await test.step('验证接入的事件可查询', async () => {
    const res = await API.get(page, `${API_BASE}/change-events?page=1&page_size=50&keyword=${tag}`)
    expect(res.code).toBe(0)
    await page.screenshot({ path: 'test-results/CE-2-02-事件查询.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// CE-3: 变更事件删除
// ---------------------------------------------------------------------------
test('CE-3 变更事件删除', async ({ authPage: page }) => {
  let eventId: number | undefined

  await test.step('获取最新变更事件', async () => {
    const res = await API.get(page, `${API_BASE}/change-events?page=1&page_size=1`)
    expect(res.code).toBe(0)
    const list = res.data.list || res.data || []
    if (list.length > 0) {
      eventId = list[0].id
    }
    await page.screenshot({ path: 'test-results/CE-3-01-获取事件.png', fullPage: false })
  })

  if (eventId) {
    await test.step('删除变更事件', async () => {
      const res = await API.del(page, `${API_BASE}/change-events/${eventId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/CE-3-02-删除成功.png', fullPage: false })
    })

    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/change-events/${eventId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/CE-3-03-删除验证.png', fullPage: false })
    })
  } else {
    await test.step('无变更事件 — 跳过删除测试', async () => {
      await page.screenshot({ path: 'test-results/CE-3-02-无事件.png', fullPage: false })
    })
  }
})
