import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a channel first, then create an exclusion rule under it */
async function createExclusionRule(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  // First create a channel
  const channelRes = await API.post(page, `${API_BASE}/channels`, {
    name: `channel-for-exclusion-${tag}`,
    description: 'Channel for exclusion rule test',
  })
  const channelId = channelRes.data?.id || channelRes.data?.ID
  if (!channelId) throw new Error('Failed to create channel for exclusion rule')

  const payload = {
    channel_id: channelId,
    name: `exclusion-${tag}`,
    description: `Functional test exclusion rule ${tag}`,
    conditions: JSON.stringify([{ name: 'env', value: 'test', is_regex: false }]),
    is_enabled: true,
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/channels/${channelId}/exclusion-rules`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  const ruleId = res.data.id || res.data.ID
  expect(ruleId).toBeGreaterThan(0)
  return { ...res.data, id: ruleId, _channelId: channelId, _tag: tag, _payload: payload }
}

/** Helper: delete an exclusion rule by ID, ignoring errors (for cleanup) */
async function cleanupExclusionRule(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/exclusion-rules/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// ER-1 排除规则 CRUD
// ---------------------------------------------------------------------------
test('ER-1 排除规则 CRUD', async ({ authPage: page }) => {
  let ruleId: number | null = null
  let channelId: number | null = null

  try {
    // ---- 1. 创建排除规则 ----
    await test.step('创建排除规则', async () => {
      const rule = await createExclusionRule(page, {
        description: 'CRUD test exclusion rule',
      })
      ruleId = rule.id
      channelId = rule._channelId
      expect(rule.name).toContain('exclusion-')
      expect(rule.is_enabled !== undefined || rule.status !== undefined).toBe(true)
      expect(rule.description).toBe('CRUD test exclusion rule')
      await page.screenshot({ path: 'test-results/ER-1-01-创建成功.png', fullPage: false })
    })

    // ---- 2. 验证规则通过channel列表 ----
    await test.step('验证规则通过channel列表', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/exclusion-rules`)
      expect(res.code).toBe(0)
      const rules = res.data?.list || res.data || []
      const found = Array.isArray(rules) && rules.some((r: any) => (r.id || r.ID) === ruleId)
      expect(found).toBe(true)
      await page.screenshot({ path: 'test-results/ER-1-02-验证成功.png', fullPage: false })
    })

    // ---- 3. 更新规则（改名、改描述） ----
    await test.step('更新规则名称和描述', async () => {
      const res = await API.put(page, `${API_BASE}/exclusion-rules/${ruleId}`, {
        name: `updated-exclusion-${uid()}`,
        description: 'Updated by functional test',
        conditions: JSON.stringify([{ name: 'env', value: 'production', is_regex: false }]),
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ER-1-03-更新成功.png', fullPage: false })
    })

    // ---- 4. 验证更新生效 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/exclusion-rules`)
      expect(res.code).toBe(0)
      const rules = res.data?.list || res.data || []
      const found = Array.isArray(rules) && rules.find((r: any) => (r.id || r.ID) === ruleId)
      expect(found).toBeTruthy()
      if (found) {
        expect(found.name).toContain('updated-exclusion-')
        expect(found.description).toBe('Updated by functional test')
      }
      await page.screenshot({ path: 'test-results/ER-1-04-更新验证.png', fullPage: false })
    })

    // ---- 5. 删除规则 ----
    await test.step('删除规则', async () => {
      const res = await API.del(page, `${API_BASE}/exclusion-rules/${ruleId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ER-1-05-删除成功.png', fullPage: false })
    })

    // ---- 6. 验证删除生效 ----
    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/exclusion-rules`)
      expect(res.code).toBe(0)
      const rules = res.data?.list || res.data || []
      const found = Array.isArray(rules) && rules.find((r: any) => (r.id || r.ID) === ruleId)
      expect(found).toBeFalsy()
      await page.screenshot({ path: 'test-results/ER-1-06-删除验证.png', fullPage: false })
    })

    ruleId = null
  } finally {
    if (ruleId) await cleanupExclusionRule(page, ruleId)
  }
})

// ---------------------------------------------------------------------------
// ER-2 排除规则关联 channel
// ---------------------------------------------------------------------------
test('ER-2 排除规则关联 channel', async ({ authPage: page }) => {
  let ruleId: number | null = null

  try {
    // ---- 1. 创建排除规则 ----
    await test.step('创建排除规则', async () => {
      const rule = await createExclusionRule(page, {
        channel_ids: [],
      })
      ruleId = rule.id
      await page.screenshot({ path: 'test-results/ER-2-01-创建规则.png', fullPage: false })
    })

    // ---- 2. 获取可用 channel 列表 ----
    let channelId: number | null = null
    await test.step('获取可用 channel 列表', async () => {
      const res = await API.get(page, `${API_BASE}/channels?page_size=10`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      if (list.length > 0) {
        channelId = list[0].id
      }
      await page.screenshot({ path: 'test-results/ER-2-02-Channel列表.png', fullPage: false })
    })

    // ---- 3. 关联 channel 到排除规则 ----
    await test.step('关联 channel 到排除规则', async () => {
      if (channelId) {
        const res = await API.put(page, `${API_BASE}/exclusion-rules/${ruleId}`, {
          channel_ids: [channelId],
        })
        expect(res.code).toBe(0)
      }
      await page.screenshot({ path: 'test-results/ER-2-03-关联Channel.png', fullPage: false })
    })

    // ---- 4. 验证关联生效 ----
    await test.step('验证关联生效', async () => {
      const res = await API.get(page, `${API_BASE}/exclusion-rules/${ruleId}`)
      expect(res.code).toBe(0)
      expect(res.data.channel_ids).toBeTruthy()
      await page.screenshot({ path: 'test-results/ER-2-04-关联验证.png', fullPage: false })
    })

    // ---- 5. 解除关联 ----
    await test.step('解除 channel 关联', async () => {
      const res = await API.put(page, `${API_BASE}/exclusion-rules/${ruleId}`, {
        channel_ids: [],
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ER-2-05-解除关联.png', fullPage: false })
    })

    // ---- 6. 验证解除关联生效 ----
    await test.step('验证解除关联生效', async () => {
      const res = await API.get(page, `${API_BASE}/exclusion-rules/${ruleId}`)
      expect(res.code).toBe(0)
      expect(res.data.channel_ids).toHaveLength(0)
      await page.screenshot({ path: 'test-results/ER-2-06-解除验证.png', fullPage: false })
    })
  } finally {
    if (ruleId) await cleanupExclusionRule(page, ruleId)
  }
})

// ---------------------------------------------------------------------------
// ER-3 排除规则匹配预览
// ---------------------------------------------------------------------------
test('ER-3 排除规则匹配预览', async ({ authPage: page }) => {
  let ruleId: number | null = null

  try {
    // ---- 1. 创建排除规则 ----
    await test.step('创建排除规则', async () => {
      const rule = await createExclusionRule(page, {
        matchers: [
          { name: 'env', value: 'production', is_regex: false },
          { name: 'job', value: 'api-.*', is_regex: true },
        ],
        description: 'Match preview test',
      })
      ruleId = rule.id
      await page.screenshot({ path: 'test-results/ER-3-01-创建规则.png', fullPage: false })
    })

    // ---- 2. 测试命中场景 ----
    await test.step('测试命中场景', async () => {
      const res = await API.post(page, `${API_BASE}/exclusion-rules/${ruleId}/preview`, {
        labels: { env: 'production', job: 'api-server', severity: 'critical' },
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/ER-3-02-命中预览.png', fullPage: false })
    })

    // ---- 3. 测试不命中场景 ----
    await test.step('测试不命中场景', async () => {
      const res = await API.post(page, `${API_BASE}/exclusion-rules/${ruleId}/preview`, {
        labels: { env: 'staging', job: 'web-server', severity: 'warning' },
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ER-3-03-不命中预览.png', fullPage: false })
    })

    // ---- 4. 测试部分匹配场景 ----
    await test.step('测试部分匹配场景', async () => {
      const res = await API.post(page, `${API_BASE}/exclusion-rules/${ruleId}/preview`, {
        labels: { env: 'production', job: 'web-server' },
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ER-3-04-部分匹配.png', fullPage: false })
    })
  } finally {
    if (ruleId) await cleanupExclusionRule(page, ruleId)
  }
})
