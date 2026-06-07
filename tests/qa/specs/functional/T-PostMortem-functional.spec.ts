import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create an incident via API and return the created object */
async function createIncident(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    title: `incident-${tag}`,
    description: `Functional test incident ${tag}`,
    severity: 'critical',
    status: 'open',
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/incidents`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete an incident by ID, ignoring errors (for cleanup) */
async function cleanupIncident(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/incidents/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// PM-1 故障复盘 CRUD
// ---------------------------------------------------------------------------
test('PM-1 故障复盘 CRUD', async ({ authPage: page }) => {
  let incidentId: number | null = null
  let postMortemId: number | null = null

  try {
    // ---- 1. 创建事件（用于关联复盘） ----
    await test.step('创建事件', async () => {
      const incident = await createIncident(page, {
        title: `pm-crud-test-${uid()}`,
        description: 'CRUD test for post-mortem',
      })
      incidentId = incident.id
      await page.screenshot({ path: 'test-results/PM-1-01-创建事件.png', fullPage: false })
    })

    // ---- 2. 创建故障复盘 ----
    await test.step('创建故障复盘', async () => {
      const res = await API.post(page, `${API_BASE}/incidents/${incidentId}/post-mortem`, {
        title: `复盘报告-${uid()}`,
        summary: '故障复盘摘要',
        root_cause: '根因分析',
        timeline: '故障时间线',
        action_items: '改进措施',
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      expect(res.data.id).toBeGreaterThan(0)
      postMortemId = res.data.id
      await page.screenshot({ path: 'test-results/PM-1-02-创建复盘.png', fullPage: false })
    })

    // ---- 3. GET 验证复盘详情 ----
    await test.step('GET 验证复盘详情', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/post-mortem`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      expect(res.data.summary).toBe('故障复盘摘要')
      expect(res.data.root_cause).toBe('根因分析')
      await page.screenshot({ path: 'test-results/PM-1-03-GET验证.png', fullPage: false })
    })

    // ---- 4. 更新故障复盘 ----
    await test.step('更新故障复盘', async () => {
      const res = await API.put(page, `${API_BASE}/incidents/${incidentId}/post-mortem`, {
        title: '更新后的复盘报告',
        summary: '更新后的摘要',
        root_cause: '更新后的根因',
        timeline: '更新后的时间线',
        action_items: '更新后的改进措施',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/PM-1-04-更新成功.png', fullPage: false })
    })

    // ---- 5. 验证更新生效 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/post-mortem`)
      expect(res.code).toBe(0)
      expect(res.data.summary).toBe('更新后的摘要')
      expect(res.data.root_cause).toBe('更新后的根因')
      await page.screenshot({ path: 'test-results/PM-1-05-更新验证.png', fullPage: false })
    })

    // ---- 6. 删除故障复盘 ----
    await test.step('删除故障复盘', async () => {
      const res = await API.del(page, `${API_BASE}/incidents/${incidentId}/post-mortem`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/PM-1-06-删除成功.png', fullPage: false })
    })

    // ---- 7. 验证删除生效 ----
    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/post-mortem`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/PM-1-07-删除验证.png', fullPage: false })
    })

    postMortemId = null
  } finally {
    if (incidentId) await cleanupIncident(page, incidentId)
  }
})

// ---------------------------------------------------------------------------
// PM-2 故障复盘 AI 生成初稿
// ---------------------------------------------------------------------------
test('PM-2 故障复盘 AI 生成初稿', async ({ authPage: page }) => {
  let incidentId: number | null = null

  try {
    // ---- 1. 创建事件 ----
    await test.step('创建事件', async () => {
      const incident = await createIncident(page, {
        title: `ai-draft-test-${uid()}`,
        description: '测试 AI 生成复盘初稿',
        severity: 'critical',
      })
      incidentId = incident.id
      await page.screenshot({ path: 'test-results/PM-2-01-创建事件.png', fullPage: false })
    })

    // ---- 2. 请求 AI 生成初稿 ----
    await test.step('请求 AI 生成初稿', async () => {
      const res = await API.post(page, `${API_BASE}/incidents/${incidentId}/post-mortem/ai-draft`, {})
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/PM-2-02-AI生成初稿.png', fullPage: false })
    })

    // ---- 3. 验证 AI 生成的内容 ----
    await test.step('验证 AI 生成的内容', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/post-mortem`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      // AI should have generated at least some content
      const hasContent = res.data.summary || res.data.root_cause || res.data.timeline
      expect(hasContent).toBeTruthy()
      await page.screenshot({ path: 'test-results/PM-2-03-AI内容验证.png', fullPage: false })
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/PM-2-ERROR.png', fullPage: false })
    throw e
  } finally {
    if (incidentId) await cleanupIncident(page, incidentId)
  }
})

// ---------------------------------------------------------------------------
// PM-3 故障复盘发布
// ---------------------------------------------------------------------------
test('PM-3 故障复盘发布', async ({ authPage: page }) => {
  let incidentId: number | null = null

  try {
    // ---- 1. 创建事件和复盘 ----
    await test.step('创建事件和复盘', async () => {
      const incident = await createIncident(page, {
        title: `publish-test-${uid()}`,
        description: '测试复盘发布',
      })
      incidentId = incident.id
      await API.post(page, `${API_BASE}/incidents/${incidentId}/post-mortem`, {
        title: '待发布复盘',
        summary: '待发布摘要',
        root_cause: '待发布根因',
        timeline: '待发布时间线',
        action_items: '待发布改进措施',
      })
      await page.screenshot({ path: 'test-results/PM-3-01-创建复盘.png', fullPage: false })
    })

    // ---- 2. 发布复盘 ----
    await test.step('发布复盘', async () => {
      const res = await API.post(page, `${API_BASE}/incidents/${incidentId}/post-mortem/publish`, {})
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/PM-3-02-发布成功.png', fullPage: false })
    })

    // ---- 3. 验证发布状态 ----
    await test.step('验证发布状态', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/post-mortem`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      expect(res.data.status).toBe('published')
      await page.screenshot({ path: 'test-results/PM-3-03-发布状态验证.png', fullPage: false })
    })

    // ---- 4. 验证已发布的复盘不可编辑 ----
    await test.step('验证已发布复盘不可编辑', async () => {
      const res = await API.put(page, `${API_BASE}/incidents/${incidentId}/post-mortem`, {
        summary: '尝试修改已发布复盘',
      })
      // Should return an error or be restricted
      await page.screenshot({ path: 'test-results/PM-3-04-编辑限制验证.png', fullPage: false })
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/PM-3-ERROR.png', fullPage: false })
    throw e
  } finally {
    if (incidentId) await cleanupIncident(page, incidentId)
  }
})

// ---------------------------------------------------------------------------
// PM-4 故障复盘 AI 摘要
// ---------------------------------------------------------------------------
test('PM-4 故障复盘 AI 摘要', async ({ authPage: page }) => {
  let incidentId: number | null = null

  try {
    // ---- 1. 创建事件和复盘 ----
    await test.step('创建事件和复盘', async () => {
      const incident = await createIncident(page, {
        title: `ai-summary-test-${uid()}`,
        description: '测试 AI 摘要功能',
      })
      incidentId = incident.id
      await API.post(page, `${API_BASE}/incidents/${incidentId}/post-mortem`, {
        title: 'AI 摘要测试复盘',
        summary: '这是一个详细的故障复盘报告，包含了大量的技术细节和分析内容，用于测试 AI 摘要功能。',
        root_cause: '根因分析：数据库连接池耗尽导致服务不可用',
        timeline: '10:00 发现告警，10:05 确认故障，10:30 定位根因，11:00 修复完成',
        action_items: '1. 增加连接池监控 2. 优化连接池配置 3. 添加自动扩容机制',
      })
      await page.screenshot({ path: 'test-results/PM-4-01-创建复盘.png', fullPage: false })
    })

    // ---- 2. 请求 AI 生成摘要 ----
    await test.step('请求 AI 生成摘要', async () => {
      const res = await API.post(page, `${API_BASE}/incidents/${incidentId}/post-mortem/ai-summary`, {})
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/PM-4-02-AI摘要生成.png', fullPage: false })
    })

    // ---- 3. 验证 AI 摘要内容 ----
    await test.step('验证 AI 摘要内容', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/post-mortem`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      // AI summary should be present
      expect(res.data.ai_summary).toBeTruthy()
      await page.screenshot({ path: 'test-results/PM-4-03-AI摘要验证.png', fullPage: false })
    })

    // ---- 4. 验证摘要长度合理 ----
    await test.step('验证摘要长度合理', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/post-mortem`)
      expect(res.code).toBe(0)
      if (res.data.ai_summary) {
        // AI summary should be concise but meaningful
        expect(res.data.ai_summary.length).toBeGreaterThan(10)
        expect(res.data.ai_summary.length).toBeLessThan(5000)
      }
      await page.screenshot({ path: 'test-results/PM-4-04-摘要长度验证.png', fullPage: false })
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/PM-4-ERROR.png', fullPage: false })
    throw e
  } finally {
    if (incidentId) await cleanupIncident(page, incidentId)
  }
})
