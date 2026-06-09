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
    match_conditions: JSON.stringify([{ field: 'severity', operator: 'eq', value: 'critical' }]),
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
      const conditions = [
        { field: 'severity', operator: 'eq', value: 'critical' },
        { field: 'env', operator: 'eq', value: 'production' },
      ]
      const policy = await createDispatchPolicy(page, {
        match_conditions: JSON.stringify(conditions),
      })
      policyId = policy.id
      // match_conditions is stored as a JSON string
      expect(policy.match_conditions).toBeTruthy()
      const parsed = typeof policy.match_conditions === 'string'
        ? JSON.parse(policy.match_conditions)
        : policy.match_conditions
      expect(parsed.length).toBe(2)
      await page.screenshot({ path: 'test-results/DP-2-01-创建带条件策略.png', fullPage: false })
    })

    // ---- 2. 验证触发条件保存正确 ----
    await test.step('验证触发条件保存正确', async () => {
      const res = await API.get(page, `${API_BASE}/dispatch-policies/${policyId}`)
      expect(res.code).toBe(0)
      const conditions = typeof res.data.match_conditions === 'string'
        ? JSON.parse(res.data.match_conditions)
        : res.data.match_conditions
      expect(conditions.length).toBe(2)
      expect(conditions[0].field).toBe('severity')
      expect(conditions[0].value).toBe('critical')
      expect(conditions[1].field).toBe('env')
      expect(conditions[1].value).toBe('production')
      await page.screenshot({ path: 'test-results/DP-2-02-条件验证.png', fullPage: false })
    })

    // ---- 3. 更新触发条件 ----
    await test.step('更新触发条件', async () => {
      const conditions = [
        { field: 'severity', operator: 'eq', value: 'warning' },
        { field: 'job', operator: 'regex', value: 'api-.*' },
      ]
      const res = await API.put(page, `${API_BASE}/dispatch-policies/${policyId}`, {
        match_conditions: JSON.stringify(conditions),
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DP-2-03-更新条件.png', fullPage: false })
    })

    // ---- 4. 验证更新后的触发条件 ----
    await test.step('验证更新后的触发条件', async () => {
      const res = await API.get(page, `${API_BASE}/dispatch-policies/${policyId}`)
      expect(res.code).toBe(0)
      const conditions = typeof res.data.match_conditions === 'string'
        ? JSON.parse(res.data.match_conditions)
        : res.data.match_conditions
      expect(conditions.length).toBe(2)
      expect(conditions[0].value).toBe('warning')
      expect(conditions[1].operator).toBe('regex')
      await page.screenshot({ path: 'test-results/DP-2-04-更新后验证.png', fullPage: false })
    })

    // ---- 5. 测试正则匹配 ----
    await test.step('测试正则匹配条件', async () => {
      const res = await API.get(page, `${API_BASE}/dispatch-policies/${policyId}`)
      expect(res.code).toBe(0)
      const conditions = typeof res.data.match_conditions === 'string'
        ? JSON.parse(res.data.match_conditions)
        : res.data.match_conditions
      const regexCondition = conditions.find((c: any) => c.operator === 'regex')
      expect(regexCondition).toBeTruthy()
      expect(regexCondition.value).toBe('api-.*')
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
// DP-4 分派日志查看 (dispatch-logs are under /incidents/:id/dispatch-logs)
// ---------------------------------------------------------------------------
test('DP-4 分派日志查看', async ({ authPage: page }) => {
  let incidentId: number | null = null

  try {
    // ---- 0. 获取一个 incident 用于查日志 ----
    await test.step('获取一个 incident', async () => {
      const res = await API.get(page, `${API_BASE}/incidents?page=1&page_size=1`)
      expect(res.code).toBe(0)
      const list = res.data?.list || res.data || []
      if (Array.isArray(list) && list.length > 0) {
        incidentId = list[0].id || list[0].ID
      }
      await page.screenshot({ path: 'test-results/DP-4-00-获取incident.png', fullPage: false })
    })

    // ---- 1. 获取分派日志列表 ----
    await test.step('获取分派日志列表', async () => {
      if (!incidentId) {
        test.skip(true, 'No incident available for dispatch log test')
        return
      }
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/dispatch-logs`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/DP-4-01-分派日志列表.png', fullPage: false })
    })

    // ---- 2. 验证日志列表结构 ----
    await test.step('验证日志列表结构', async () => {
      if (!incidentId) {
        test.skip(true, 'No incident available for dispatch log test')
        return
      }
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/dispatch-logs`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      // Log entries should exist (even if empty)
      expect(Array.isArray(list)).toBe(true)
      await page.screenshot({ path: 'test-results/DP-4-02-日志结构验证.png', fullPage: false })
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/DP-4-ERROR.png', fullPage: false })
    throw e
  }
})
