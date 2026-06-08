import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a channel via API and return the created object */
async function createChannel(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `channel-${tag}`,
    description: `Functional test channel ${tag}`,
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/channels`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a channel by ID, ignoring errors (for cleanup) */
async function cleanupChannel(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/channels/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// CH-1 协作空间 CRUD
// ---------------------------------------------------------------------------
test('CH-1 协作空间 CRUD', async ({ authPage: page }) => {
  let channelId: number | null = null

  try {
    // ---- 1. 创建协作空间 ----
    await test.step('创建协作空间', async () => {
      const ch = await createChannel(page, {
        description: 'CRUD test channel',
      })
      channelId = ch.id
      expect(ch.name).toContain('channel-')
      expect(ch.description).toBe('CRUD test channel')
      await page.screenshot({ path: 'test-results/CH-1-01-创建成功.png', fullPage: false })
    })

    // ---- 2. GET 验证所有字段 ----
    await test.step('GET 验证空间已保存', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}`)
      expect(res.code).toBe(0)
      const r = res.data
      expect(r.id).toBe(channelId)
      expect(r.name).toContain('channel-')
      expect(r.description).toBe('CRUD test channel')
      await page.screenshot({ path: 'test-results/CH-1-02-GET验证.png', fullPage: false })
    })

    // ---- 3. 更新空间（改名、改描述） ----
    await test.step('更新空间名称和描述', async () => {
      const res = await API.put(page, `${API_BASE}/channels/${channelId}`, {
        name: `updated-channel-${uid()}`,
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/CH-1-03-更新成功.png', fullPage: false })
    })

    // ---- 4. 验证更新生效 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}`)
      expect(res.code).toBe(0)
      expect(res.data.name).toContain('updated-channel-')
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/CH-1-04-更新验证.png', fullPage: false })
    })

    // ---- 5. 删除空间 ----
    await test.step('删除空间', async () => {
      const res = await API.del(page, `${API_BASE}/channels/${channelId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/CH-1-05-删除成功.png', fullPage: false })
    })

    // ---- 6. 验证删除生效 ----
    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/CH-1-06-删除验证.png', fullPage: false })
    })

    channelId = null
  } finally {
    if (channelId) await cleanupChannel(page, channelId)
  }
})

// ---------------------------------------------------------------------------
// CH-2 协作空间 star/unstar
// ---------------------------------------------------------------------------
test('CH-2 协作空间 star/unstar', async ({ authPage: page }) => {
  let channelId: number | null = null

  try {
    // ---- 1. 创建协作空间 ----
    await test.step('创建协作空间', async () => {
      const ch = await createChannel(page)
      channelId = ch.id
      await page.screenshot({ path: 'test-results/CH-2-01-创建空间.png', fullPage: false })
    })

    // ---- 2. Star 协作空间 ----
    await test.step('Star 协作空间', async () => {
      const res = await API.post(page, `${API_BASE}/channels/${channelId}/star`, {})
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/CH-2-02-Star成功.png', fullPage: false })
    })

    // ---- 3. 验证 Star 状态 ----
    await test.step('验证 Star 状态', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}`)
      expect(res.code).toBe(0)
      expect(res.data.is_starred).toBe(true)
      await page.screenshot({ path: 'test-results/CH-2-03-Star验证.png', fullPage: false })
    })

    // ---- 4. Unstar 协作空间 ----
    await test.step('Unstar 协作空间', async () => {
      const res = await API.del(page, `${API_BASE}/channels/${channelId}/star`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/CH-2-04-Unstar成功.png', fullPage: false })
    })

    // ---- 5. 验证 Unstar 状态 ----
    await test.step('验证 Unstar 状态', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}`)
      expect(res.code).toBe(0)
      expect(res.data.is_starred).toBe(false)
      await page.screenshot({ path: 'test-results/CH-2-05-Unstar验证.png', fullPage: false })
    })

    // ---- 6. 再次 Star 验证可切换 ----
    await test.step('再次 Star 验证可切换', async () => {
      const res = await API.post(page, `${API_BASE}/channels/${channelId}/star`, {})
      expect(res.code).toBe(0)
      const verifyRes = await API.get(page, `${API_BASE}/channels/${channelId}`)
      expect(verifyRes.code).toBe(0)
      expect(verifyRes.data.is_starred).toBe(true)
      await page.screenshot({ path: 'test-results/CH-2-06-再次Star.png', fullPage: false })
    })
  } finally {
    if (channelId) await cleanupChannel(page, channelId)
  }
})

// ---------------------------------------------------------------------------
// CH-3 协作空间排除规则关联
// ---------------------------------------------------------------------------
test('CH-3 协作空间排除规则关联', async ({ authPage: page }) => {
  let channelId: number | null = null
  let exclusionRuleId: number | null = null

  try {
    // ---- 1. 创建协作空间 ----
    await test.step('创建协作空间', async () => {
      const ch = await createChannel(page)
      channelId = ch.id
      await page.screenshot({ path: 'test-results/CH-3-01-创建空间.png', fullPage: false })
    })

    // ---- 2. 创建排除规则 ----
    await test.step('创建排除规则', async () => {
      const tag = uid()
      const res = await API.post(page, `${API_BASE}/channels/${channelId}/exclusion-rules`, {
        channel_id: channelId,
        name: `ch-exclusion-${tag}`,
        description: 'Channel exclusion rule test',
        conditions: JSON.stringify([{ name: 'env', value: 'test', is_regex: false }]),
        is_enabled: true,
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      exclusionRuleId = res.data.id || res.data.ID
      await page.screenshot({ path: 'test-results/CH-3-02-创建排除规则.png', fullPage: false })
    })

    // ---- 3. 验证排除规则关联到空间 ----
    await test.step('验证排除规则关联', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/exclusion-rules`)
      expect(res.code).toBe(0)
      const rules = res.data?.list || res.data || []
      const found = Array.isArray(rules) && rules.find((r: any) => (r.id || r.ID) === exclusionRuleId)
      expect(found).toBeTruthy()
      await page.screenshot({ path: 'test-results/CH-3-03-关联验证.png', fullPage: false })
    })

    // ---- 4. 获取空间关联的排除规则 ----
    await test.step('获取空间关联的排除规则', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/exclusion-rules`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/CH-3-04-空间排除规则.png', fullPage: false })
    })

    // ---- 5. 删除排除规则 ----
    await test.step('删除排除规则', async () => {
      const res = await API.del(page, `${API_BASE}/exclusion-rules/${exclusionRuleId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/CH-3-05-删除成功.png', fullPage: false })
    })

    // ---- 6. 验证删除生效 ----
    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/exclusion-rules`)
      expect(res.code).toBe(0)
      const rules = res.data?.list || res.data || []
      const found = Array.isArray(rules) && rules.find((r: any) => (r.id || r.ID) === exclusionRuleId)
      expect(found).toBeFalsy()
      await page.screenshot({ path: 'test-results/CH-3-06-删除验证.png', fullPage: false })
    })
  } finally {
    if (exclusionRuleId) {
      try { await API.del(page, `${API_BASE}/exclusion-rules/${exclusionRuleId}`) } catch { /* ignore */ }
    }
    if (channelId) await cleanupChannel(page, channelId)
  }
})

// ---------------------------------------------------------------------------
// CH-4 协作空间分派策略关联
// ---------------------------------------------------------------------------
test('CH-4 协作空间分派策略关联', async ({ authPage: page }) => {
  let channelId: number | null = null
  let dispatchPolicyId: number | null = null

  try {
    // ---- 1. 创建协作空间 ----
    await test.step('创建协作空间', async () => {
      const ch = await createChannel(page)
      channelId = ch.id
      await page.screenshot({ path: 'test-results/CH-4-01-创建空间.png', fullPage: false })
    })

    // ---- 2. 创建关联到空间的分派策略 ----
    await test.step('创建关联到空间的分派策略', async () => {
      const tag = uid()
      const res = await API.post(page, `${API_BASE}/channels/${channelId}/dispatch-policies`, {
        name: `ch-dispatch-${tag}`,
        description: 'Channel dispatch policy test',
        match_conditions: JSON.stringify([{name:'severity',value:'critical',is_regex:false}]),
        delay_seconds: 0,
        is_enabled: true,
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      dispatchPolicyId = res.data.id
      await page.screenshot({ path: 'test-results/CH-4-02-创建分派策略.png', fullPage: false })
    })

    // ---- 3. 验证分派策略关联到空间 ----
    await test.step('验证分派策略关联', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/dispatch-policies`)
      expect(res.code).toBe(0)
      const policies = res.data?.list || res.data || []
      const found = Array.isArray(policies) && policies.find((p: any) => (p.id || p.ID) === dispatchPolicyId)
      expect(found).toBeTruthy()
      await page.screenshot({ path: 'test-results/CH-4-03-关联验证.png', fullPage: false })
    })

    // ---- 4. 获取空间关联的分派策略 ----
    await test.step('获取空间关联的分派策略', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/dispatch-policies`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/CH-4-04-空间分派策略.png', fullPage: false })
    })

    // ---- 5. 删除分派策略 ----
    await test.step('删除分派策略', async () => {
      const res = await API.del(page, `${API_BASE}/dispatch-policies/${dispatchPolicyId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/CH-4-05-删除成功.png', fullPage: false })
    })

    // ---- 6. 验证删除生效 ----
    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/dispatch-policies`)
      expect(res.code).toBe(0)
      const policies = res.data?.list || res.data || []
      const found = Array.isArray(policies) && policies.find((p: any) => (p.id || p.ID) === dispatchPolicyId)
      expect(found).toBeFalsy()
      await page.screenshot({ path: 'test-results/CH-4-06-删除验证.png', fullPage: false })
    })
  } finally {
    if (dispatchPolicyId) {
      try { await API.del(page, `${API_BASE}/dispatch-policies/${dispatchPolicyId}`) } catch { /* ignore */ }
    }
    if (channelId) await cleanupChannel(page, channelId)
  }
})
