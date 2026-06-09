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
  let channelId: number | null = null

  try {
    // ---- 1. 创建排除规则（会自动关联到 channel） ----
    await test.step('创建排除规则', async () => {
      const rule = await createExclusionRule(page, {})
      ruleId = rule.id
      channelId = rule._channelId
      await page.screenshot({ path: 'test-results/ER-2-01-创建规则.png', fullPage: false })
    })

    // ---- 2. 通过 channel 获取排除规则列表验证关联 ----
    await test.step('验证排除规则已关联到 channel', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/exclusion-rules`)
      expect(res.code).toBe(0)
      const rules = res.data?.list || res.data || []
      const found = Array.isArray(rules) && rules.some((r: any) => (r.id || r.ID) === ruleId)
      expect(found).toBe(true)
      await page.screenshot({ path: 'test-results/ER-2-02-关联验证.png', fullPage: false })
    })

    // ---- 3. 更新排除规则（修改名称和条件） ----
    await test.step('更新排除规则', async () => {
      const res = await API.put(page, `${API_BASE}/exclusion-rules/${ruleId}`, {
        name: `updated-exclusion-${uid()}`,
        description: 'Updated for channel association test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ER-2-03-更新规则.png', fullPage: false })
    })

    // ---- 4. 验证更新后规则仍在 channel 列表中 ----
    await test.step('验证更新后规则仍在 channel 列表中', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/exclusion-rules`)
      expect(res.code).toBe(0)
      const rules = res.data?.list || res.data || []
      const found = Array.isArray(rules) && rules.find((r: any) => (r.id || r.ID) === ruleId)
      expect(found).toBeTruthy()
      expect(found.description).toBe('Updated for channel association test')
      await page.screenshot({ path: 'test-results/ER-2-04-更新验证.png', fullPage: false })
    })

    // ---- 5. 删除排除规则 ----
    await test.step('删除排除规则', async () => {
      const res = await API.del(page, `${API_BASE}/exclusion-rules/${ruleId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ER-2-05-删除规则.png', fullPage: false })
    })

    // ---- 6. 验证删除后规则不在 channel 列表中 ----
    await test.step('验证删除后规则不在 channel 列表中', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/exclusion-rules`)
      expect(res.code).toBe(0)
      const rules = res.data?.list || res.data || []
      const found = Array.isArray(rules) && rules.find((r: any) => (r.id || r.ID) === ruleId)
      expect(found).toBeFalsy()
      await page.screenshot({ path: 'test-results/ER-2-06-删除验证.png', fullPage: false })
    })

    ruleId = null
  } finally {
    if (ruleId) await cleanupExclusionRule(page, ruleId)
  }
})

// ---------------------------------------------------------------------------
// ER-3 排除规则条件管理
// ---------------------------------------------------------------------------
test('ER-3 排除规则条件管理', async ({ authPage: page }) => {
  let ruleId: number | null = null
  let channelId: number | null = null

  try {
    // ---- 1. 创建带条件的排除规则 ----
    await test.step('创建带条件的排除规则', async () => {
      const rule = await createExclusionRule(page, {
        conditions: JSON.stringify([
          { name: 'env', value: 'production', is_regex: false },
          { name: 'job', value: 'api-.*', is_regex: true },
        ]),
        description: 'Conditions management test',
      })
      ruleId = rule.id
      channelId = rule._channelId
      await page.screenshot({ path: 'test-results/ER-3-01-创建规则.png', fullPage: false })
    })

    // ---- 2. 验证条件已保存 ----
    await test.step('验证条件已保存', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/exclusion-rules`)
      expect(res.code).toBe(0)
      const rules = res.data?.list || res.data || []
      const found = Array.isArray(rules) && rules.find((r: any) => (r.id || r.ID) === ruleId)
      expect(found).toBeTruthy()
      if (found) {
        const conditions = JSON.parse(found.conditions)
        expect(Array.isArray(conditions)).toBe(true)
        expect(conditions.length).toBe(2)
        expect(conditions[0].name).toBe('env')
        expect(conditions[0].value).toBe('production')
        expect(conditions[1].is_regex).toBe(true)
      }
      await page.screenshot({ path: 'test-results/ER-3-02-条件验证.png', fullPage: false })
    })

    // ---- 3. 更新条件 ----
    await test.step('更新排除规则条件', async () => {
      const res = await API.put(page, `${API_BASE}/exclusion-rules/${ruleId}`, {
        conditions: JSON.stringify([
          { name: 'env', value: 'staging', is_regex: false },
          { name: 'severity', value: 'warning', is_regex: false },
        ]),
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ER-3-03-更新条件.png', fullPage: false })
    })

    // ---- 4. 验证条件更新生效 ----
    await test.step('验证条件更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/exclusion-rules`)
      expect(res.code).toBe(0)
      const rules = res.data?.list || res.data || []
      const found = Array.isArray(rules) && rules.find((r: any) => (r.id || r.ID) === ruleId)
      expect(found).toBeTruthy()
      if (found) {
        const conditions = JSON.parse(found.conditions)
        expect(conditions.length).toBe(2)
        expect(conditions[0].name).toBe('env')
        expect(conditions[0].value).toBe('staging')
        expect(conditions[1].name).toBe('severity')
      }
      await page.screenshot({ path: 'test-results/ER-3-04-更新验证.png', fullPage: false })
    })

    // ---- 5. 启用/禁用排除规则 ----
    await test.step('禁用排除规则', async () => {
      const res = await API.put(page, `${API_BASE}/exclusion-rules/${ruleId}`, {
        is_enabled: false,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ER-3-05-禁用规则.png', fullPage: false })
    })

    // ---- 6. 验证禁用生效 ----
    await test.step('验证禁用生效', async () => {
      const res = await API.get(page, `${API_BASE}/channels/${channelId}/exclusion-rules`)
      expect(res.code).toBe(0)
      const rules = res.data?.list || res.data || []
      const found = Array.isArray(rules) && rules.find((r: any) => (r.id || r.ID) === ruleId)
      expect(found).toBeTruthy()
      if (found) {
        expect(found.is_enabled).toBe(false)
      }
      await page.screenshot({ path: 'test-results/ER-3-06-禁用验证.png', fullPage: false })
    })
  } finally {
    if (ruleId) await cleanupExclusionRule(page, ruleId)
  }
})
