import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a channel first, then create a dispatch policy under it */
async function createDispatchPolicy(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  // First create a channel
  const channelRes = await API.post(page, `${API_BASE}/channels`, {
    name: `channel-for-dispatch-${tag}`,
    description: 'Channel for dispatch policy test',
  })
  const channelId = channelRes.data?.id || channelRes.data?.ID
  if (!channelId) throw new Error('Failed to create channel for dispatch policy')

  const payload = {
    name: `dispatch-${tag}`,
    description: `Functional test dispatch policy ${tag}`,
    matchers: [{ name: 'severity', value: 'critical', is_regex: false }],
    delay_seconds: 0,
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/channels/${channelId}/dispatch-policies`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  const policyId = res.data.id || res.data.ID
  expect(policyId).toBeGreaterThan(0)
  return { ...res.data, id: policyId, _channelId: channelId, _tag: tag, _payload: payload }
}

/** Helper: delete a dispatch policy by ID, ignoring errors (for cleanup) */
async function cleanupDispatchPolicy(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/dispatch-policies/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// DP-1 分派策略 CRUD
// ---------------------------------------------------------------------------
test('DP-1 分派策略 CRUD', async ({ authPage: page }) => {
  let policyId: number | null = null
  let channelId: number | null = null

  try {
    // ---- 1. 创建分派策略 ----
    await test.step('创建分派策略', async () => {
      const policy = await createDispatchPolicy(page, {
        description: 'CRUD test dispatch policy',
        delay_seconds: 60,
      })
      policyId = policy.id
      channelId = policy._channelId
      expect(policy.name).toContain('dispatch-')
      expect(policy.is_enabled !== undefined || policy.status !== undefined).toBe(true)
      expect(policy.description).toBe('CRUD test dispatch policy')
      await page.screenshot({ path: 'test-results/DP-1-01-创建成功.png', fullPage: false })
    })

    // ---- 2. 验证策略通过channel列表 ----
    await test.step('验证策略通过channel列表', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/dispatch-policies`)
      expect(res.code).toBe(0)
      const policies = res.data?.list || res.data || []
      const found = Array.isArray(policies) && policies.some((p: any) => (p.id || p.ID) === policyId)
      expect(found).toBe(true)
      await page.screenshot({ path: 'test-results/DP-1-02-验证成功.png', fullPage: false })
    })

    // ---- 3. 更新策略（改名、改延迟） ----
    await test.step('更新策略名称和延迟', async () => {
      const res = await API.put(page, `${API_BASE}/dispatch-policies/${policyId}`, {
        name: `updated-dispatch-${uid()}`,
        delay_seconds: 120,
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DP-1-03-更新成功.png', fullPage: false })
    })

    // ---- 4. 验证更新生效 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/dispatch-policies`)
      expect(res.code).toBe(0)
      const policies = res.data?.list || res.data || []
      const found = Array.isArray(policies) && policies.find((p: any) => (p.id || p.ID) === policyId)
      expect(found).toBeTruthy()
      if (found) {
        expect(found.name).toContain('updated-dispatch-')
        expect(found.description).toBe('Updated by functional test')
      }
      await page.screenshot({ path: 'test-results/DP-1-04-更新验证.png', fullPage: false })
    })

    // ---- 5. 删除策略 ----
    await test.step('删除策略', async () => {
      const res = await API.del(page, `${API_BASE}/dispatch-policies/${policyId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DP-1-05-删除成功.png', fullPage: false })
    })

    // ---- 6. 验证删除生效 ----
    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/dispatch-policies`)
      expect(res.code).toBe(0)
      const policies = res.data?.list || res.data || []
      const found = Array.isArray(policies) && policies.find((p: any) => (p.id || p.ID) === policyId)
      expect(found).toBeFalsy()
      await page.screenshot({ path: 'test-results/DP-1-06-删除验证.png', fullPage: false })
    })

    policyId = null
  } finally {
    if (policyId) await cleanupDispatchPolicy(page, policyId)
  }
})

// ---------------------------------------------------------------------------
// DP-2 分派策略触发条件
// ---------------------------------------------------------------------------
test('DP-2 分派策略触发条件', async ({ authPage: page }) => {
  let policyId: number | null = null

  try {
    // ---- 1. 创建带触发条件的分派策略 ----
    await test.step('创建带触发条件的分派策略', async () => {
      const policy = await createDispatchPolicy(page, {
        matchers: [
          { name: 'severity', value: 'critical', is_regex: false },
          { name: 'env', value: 'production', is_regex: false },
        ],
      })
      policyId = policy.id
      expect(policy.matchers).toBeTruthy()
      expect(policy.matchers.length).toBe(2)
      await page.screenshot({ path: 'test-results/DP-2-01-创建带条件策略.png', fullPage: false })
    })

    // ---- 2. 验证触发条件保存正确 ----
    await test.step('验证触发条件保存正确', async () => {
      const res = await API.get(page, `${API_BASE}/dispatch-policies/${policyId}`)
      expect(res.code).toBe(0)
      expect(res.data.matchers.length).toBe(2)
      expect(res.data.matchers[0].name).toBe('severity')
      expect(res.data.matchers[0].value).toBe('critical')
      expect(res.data.matchers[1].name).toBe('env')
      expect(res.data.matchers[1].value).toBe('production')
      await page.screenshot({ path: 'test-results/DP-2-02-条件验证.png', fullPage: false })
    })

    // ---- 3. 更新触发条件 ----
    await test.step('更新触发条件', async () => {
      const res = await API.put(page, `${API_BASE}/dispatch-policies/${policyId}`, {
        matchers: [
          { name: 'severity', value: 'warning', is_regex: false },
          { name: 'job', value: 'api-.*', is_regex: true },
        ],
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DP-2-03-更新条件.png', fullPage: false })
    })

    // ---- 4. 验证更新后的触发条件 ----
    await test.step('验证更新后的触发条件', async () => {
      const res = await API.get(page, `${API_BASE}/dispatch-policies/${policyId}`)
      expect(res.code).toBe(0)
      expect(res.data.matchers.length).toBe(2)
      expect(res.data.matchers[0].value).toBe('warning')
      expect(res.data.matchers[1].is_regex).toBe(true)
      await page.screenshot({ path: 'test-results/DP-2-04-更新后验证.png', fullPage: false })
    })

    // ---- 5. 测试正则匹配 ----
    await test.step('测试正则匹配条件', async () => {
      const res = await API.get(page, `${API_BASE}/dispatch-policies/${policyId}`)
      expect(res.code).toBe(0)
      const regexMatcher = res.data.matchers.find((m: any) => m.is_regex)
      expect(regexMatcher).toBeTruthy()
      expect(regexMatcher.value).toBe('api-.*')
      await page.screenshot({ path: 'test-results/DP-2-05-正则匹配验证.png', fullPage: false })
    })
  } finally {
    if (policyId) await cleanupDispatchPolicy(page, policyId)
  }
})

// ---------------------------------------------------------------------------
// DP-3 分派策略延迟配置
// ---------------------------------------------------------------------------
test('DP-3 分派策略延迟配置', async ({ authPage: page }) => {
  let policyId: number | null = null

  try {
    // ---- 1. 创建带延迟的分派策略 ----
    await test.step('创建带延迟的分派策略', async () => {
      const policy = await createDispatchPolicy(page, {
        delay_seconds: 300,
      })
      policyId = policy.id
      expect(policy.delay_seconds).toBe(300)
      await page.screenshot({ path: 'test-results/DP-3-01-创建带延迟策略.png', fullPage: false })
    })

    // ---- 2. 验证延迟配置保存 ----
    await test.step('验证延迟配置保存', async () => {
      const res = await API.get(page, `${API_BASE}/dispatch-policies/${policyId}`)
      expect(res.code).toBe(0)
      expect(res.data.delay_seconds).toBe(300)
      await page.screenshot({ path: 'test-results/DP-3-02-延迟配置验证.png', fullPage: false })
    })

    // ---- 3. 更新延迟配置 ----
    await test.step('更新延迟配置', async () => {
      const res = await API.put(page, `${API_BASE}/dispatch-policies/${policyId}`, {
        delay_seconds: 600,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DP-3-03-更新延迟.png', fullPage: false })
    })

    // ---- 4. 验证延迟更新生效 ----
    await test.step('验证延迟更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/dispatch-policies/${policyId}`)
      expect(res.code).toBe(0)
      expect(res.data.delay_seconds).toBe(600)
      await page.screenshot({ path: 'test-results/DP-3-04-延迟更新验证.png', fullPage: false })
    })

    // ---- 5. 设置延迟为 0（立即分派） ----
    await test.step('设置延迟为 0', async () => {
      const res = await API.put(page, `${API_BASE}/dispatch-policies/${policyId}`, {
        delay_seconds: 0,
      })
      expect(res.code).toBe(0)
      const verifyRes = await API.get(page, `${API_BASE}/dispatch-policies/${policyId}`)
      expect(verifyRes.code).toBe(0)
      expect(verifyRes.data.delay_seconds).toBe(0)
      await page.screenshot({ path: 'test-results/DP-3-05-立即分派.png', fullPage: false })
    })
  } finally {
    if (policyId) await cleanupDispatchPolicy(page, policyId)
  }
})

// ---------------------------------------------------------------------------
// DP-4 分派日志查看
// ---------------------------------------------------------------------------
test('DP-4 分派日志查看', async ({ authPage: page }) => {
  try {
    // ---- 1. 获取分派日志列表 ----
    await test.step('获取分派日志列表', async () => {
      const res = await API.get(page, `${API_BASE}/dispatch-logs?page_size=20`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/DP-4-01-分派日志列表.png', fullPage: false })
    })

    // ---- 2. 验证日志列表结构 ----
    await test.step('验证日志列表结构', async () => {
      const res = await API.get(page, `${API_BASE}/dispatch-logs?page_size=10`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      // Log entries should exist (even if empty)
      expect(Array.isArray(list)).toBe(true)
      await page.screenshot({ path: 'test-results/DP-4-02-日志结构验证.png', fullPage: false })
    })

    // ---- 3. 按时间范围筛选日志 ----
    await test.step('按时间范围筛选日志', async () => {
      const now = new Date()
      const oneDayAgo = new Date(now.getTime() - 24 * 3600 * 1000)
      const res = await API.get(
        page,
        `${API_BASE}/dispatch-logs?start_time=${oneDayAgo.toISOString()}&end_time=${now.toISOString()}&page_size=20`
      )
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DP-4-03-时间范围筛选.png', fullPage: false })
    })

    // ---- 4. 按关键词搜索日志 ----
    await test.step('按关键词搜索日志', async () => {
      const res = await API.get(page, `${API_BASE}/dispatch-logs?keyword=test&page_size=10`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DP-4-04-关键词搜索.png', fullPage: false })
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/DP-4-ERROR.png', fullPage: false })
    throw e
  }
})
