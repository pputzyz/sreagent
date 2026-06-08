import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a mute rule via API and return the created object */
async function createMuteRule(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const now = new Date()
  const later = new Date(Date.now() + 3600 * 1000)
  const payload = {
    name: `mute-test-${tag}`,
    match_labels: { env: 'test' },
    start_time: now.toISOString(),
    end_time: later.toISOString(),
    comment: `Functional test mute rule ${tag}`,
    is_enabled: true,
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/mute-rules`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a mute rule by ID, ignoring errors (for cleanup) */
async function cleanupMuteRule(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/mute-rules/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// MR-1 静默规则 CRUD
// ---------------------------------------------------------------------------
test('MR-1 静默规则 CRUD', async ({ authPage: page }) => {
  let ruleId: number | null = null

  try {
    // ---- 1. 创建静默规则 ----
    await test.step('创建静默规则', async () => {
      const rule = await createMuteRule(page, {
        description: 'CRUD test mute rule',
      })
      ruleId = rule.id
      expect(rule.name).toContain('mute-test-')
      expect(rule.is_enabled).toBe(true)
      expect(rule.description).toBe('CRUD test mute rule')
      await page.screenshot({ path: 'test-results/MR-1-01-创建成功.png', fullPage: false })
    })

    // ---- 2. GET 验证所有字段 ----
    await test.step('GET 验证规则已保存', async () => {
      const res = await API.get(page, `${API_BASE}/mute-rules/${ruleId}`)
      expect(res.code).toBe(0)
      const r = res.data
      expect(r.id).toBe(ruleId)
      expect(r.name).toContain('mute-test-')
      expect(r.is_enabled).toBe(true)
      expect(r.description).toBe('CRUD test mute rule')
      await page.screenshot({ path: 'test-results/MR-1-02-GET验证.png', fullPage: false })
    })

    // ---- 3. 更新规则（改名、改评论） ----
    await test.step('更新规则名称和评论', async () => {
      const res = await API.put(page, `${API_BASE}/mute-rules/${ruleId}`, {
        name: `updated-mute-${uid()}`,
        description: 'Updated by functional test',
        match_labels: { env: 'test' },
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MR-1-03-更新成功.png', fullPage: false })
    })

    // ---- 4. 验证更新生效 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/mute-rules/${ruleId}`)
      expect(res.code).toBe(0)
      expect(res.data.name).toContain('updated-mute-')
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/MR-1-04-更新验证.png', fullPage: false })
    })

    // ---- 5. 删除规则 ----
    await test.step('删除规则', async () => {
      const res = await API.del(page, `${API_BASE}/mute-rules/${ruleId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MR-1-05-删除成功.png', fullPage: false })
    })

    // ---- 6. 验证删除生效 ----
    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/mute-rules/${ruleId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/MR-1-06-删除验证.png', fullPage: false })
    })

    ruleId = null
  } finally {
    if (ruleId) await cleanupMuteRule(page, ruleId)
  }
})

// ---------------------------------------------------------------------------
// MR-2 静默规则时间窗口预览
// ---------------------------------------------------------------------------
test('MR-2 静默规则时间窗口预览', async ({ authPage: page }) => {
  let ruleId: number | null = null

  try {
    // ---- 1. 创建带时间窗口的静默规则 ----
    await test.step('创建带时间窗口的静默规则', async () => {
      const startsAt = new Date(Date.now() - 60 * 1000).toISOString()
      const endsAt = new Date(Date.now() + 7200 * 1000).toISOString()
      const rule = await createMuteRule(page, {
        starts_at: startsAt,
        ends_at: endsAt,
        comment: 'Time window preview test',
      })
      ruleId = rule.id
      expect(rule.starts_at).toBeTruthy()
      expect(rule.ends_at).toBeTruthy()
      await page.screenshot({ path: 'test-results/MR-2-01-创建时间窗口规则.png', fullPage: false })
    })

    // ---- 2. 获取时间窗口预览 ----
    await test.step('获取时间窗口预览', async () => {
      const res = await API.get(page, `${API_BASE}/mute-rules/${ruleId}/preview`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/MR-2-02-时间窗口预览.png', fullPage: false })
    })

    // ---- 3. 更新时间窗口 ----
    await test.step('更新时间窗口', async () => {
      const newEndsAt = new Date(Date.now() + 14400 * 1000).toISOString()
      const res = await API.put(page, `${API_BASE}/mute-rules/${ruleId}`, {
        ends_at: newEndsAt,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MR-2-03-更新时间窗口.png', fullPage: false })
    })

    // ---- 4. 验证更新后的时间窗口预览 ----
    await test.step('验证更新后的时间窗口预览', async () => {
      const res = await API.get(page, `${API_BASE}/mute-rules/${ruleId}`)
      expect(res.code).toBe(0)
      expect(res.data.ends_at).toBeTruthy()
      await page.screenshot({ path: 'test-results/MR-2-04-更新后预览.png', fullPage: false })
    })
  } finally {
    if (ruleId) await cleanupMuteRule(page, ruleId)
  }
})

// ---------------------------------------------------------------------------
// MR-3 静默规则批量操作
// ---------------------------------------------------------------------------
test('MR-3 静默规则批量操作', async ({ authPage: page }) => {
  const ruleIds: number[] = []

  try {
    // ---- 1. 创建 3 个静默规则 ----
    await test.step('创建 3 个静默规则', async () => {
      for (let i = 0; i < 3; i++) {
        const rule = await createMuteRule(page, { comment: `Batch test rule ${i}` })
        ruleIds.push(rule.id)
      }
      expect(ruleIds.length).toBe(3)
      await page.screenshot({ path: 'test-results/MR-3-01-创建3规则.png', fullPage: false })
    })

    // ---- 2. 批量启用 ----
    await test.step('批量启用', async () => {
      const res = await API.post(page, `${API_BASE}/mute-rules/batch/enable`, { ids: ruleIds })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MR-3-02-批量启用.png', fullPage: false })
    })

    // ---- 3. 验证全部启用 ----
    await test.step('验证全部已启用', async () => {
      for (const id of ruleIds) {
        const res = await API.get(page, `${API_BASE}/mute-rules/${id}`)
        expect(res.code).toBe(0)
        expect(res.data.is_enabled).toBe(true)
      }
      await page.screenshot({ path: 'test-results/MR-3-03-启用验证.png', fullPage: false })
    })

    // ---- 4. 批量禁用 ----
    await test.step('批量禁用', async () => {
      const res = await API.post(page, `${API_BASE}/mute-rules/batch/disable`, { ids: ruleIds })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MR-3-04-批量禁用.png', fullPage: false })
    })

    // ---- 5. 验证全部禁用 ----
    await test.step('验证全部已禁用', async () => {
      for (const id of ruleIds) {
        const res = await API.get(page, `${API_BASE}/mute-rules/${id}`)
        expect(res.code).toBe(0)
        expect(res.data.status).toBe('disabled')
      }
      await page.screenshot({ path: 'test-results/MR-3-05-禁用验证.png', fullPage: false })
    })

    // ---- 6. 批量删除 ----
    await test.step('批量删除', async () => {
      const res = await API.post(page, `${API_BASE}/mute-rules/batch/delete`, { ids: ruleIds })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MR-3-06-批量删除.png', fullPage: false })
    })

    // ---- 7. 验证全部删除 ----
    await test.step('验证全部已删除', async () => {
      for (const id of ruleIds) {
        const res = await API.get(page, `${API_BASE}/mute-rules/${id}`)
        expect(res.code).not.toBe(0)
      }
      await page.screenshot({ path: 'test-results/MR-3-07-删除验证.png', fullPage: false })
    })

    ruleIds.length = 0
  } finally {
    for (const id of ruleIds) await cleanupMuteRule(page, id)
  }
})

// ---------------------------------------------------------------------------
// MR-4 静默规则启用禁用
// ---------------------------------------------------------------------------
test('MR-4 静默规则启用禁用', async ({ authPage: page }) => {
  let ruleId: number | null = null

  try {
    // ---- 1. 创建静默规则 ----
    await test.step('创建静默规则', async () => {
      const rule = await createMuteRule(page, { status: 'active' })
      ruleId = rule.id
      expect(rule.is_enabled).toBe(true)
      await page.screenshot({ path: 'test-results/MR-4-01-创建规则.png', fullPage: false })
    })

    // ---- 2. 禁用规则 ----
    await test.step('禁用规则', async () => {
      const res = await API.patch(page, `${API_BASE}/mute-rules/${ruleId}/status`, { status: 'disabled' })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MR-4-02-禁用成功.png', fullPage: false })
    })

    // ---- 3. 验证禁用生效 ----
    await test.step('验证禁用生效', async () => {
      const res = await API.get(page, `${API_BASE}/mute-rules/${ruleId}`)
      expect(res.code).toBe(0)
      expect(res.data.status).toBe('disabled')
      await page.screenshot({ path: 'test-results/MR-4-03-禁用验证.png', fullPage: false })
    })

    // ---- 4. 启用规则 ----
    await test.step('启用规则', async () => {
      const res = await API.patch(page, `${API_BASE}/mute-rules/${ruleId}/status`, { status: 'active' })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MR-4-04-启用成功.png', fullPage: false })
    })

    // ---- 5. 验证启用生效 ----
    await test.step('验证启用生效', async () => {
      const res = await API.get(page, `${API_BASE}/mute-rules/${ruleId}`)
      expect(res.code).toBe(0)
      expect(res.data.is_enabled).toBe(true)
      await page.screenshot({ path: 'test-results/MR-4-05-启用验证.png', fullPage: false })
    })
  } finally {
    if (ruleId) await cleanupMuteRule(page, ruleId)
  }
})

// ---------------------------------------------------------------------------
// MR-5 静默规则命中预览
// ---------------------------------------------------------------------------
test('MR-5 静默规则命中预览', async ({ authPage: page }) => {
  let ruleId: number | null = null

  try {
    // ---- 1. 创建静默规则 ----
    await test.step('创建静默规则', async () => {
      const rule = await createMuteRule(page, {
        matchers: [
          { name: 'env', value: 'production', is_regex: false },
          { name: 'severity', value: 'critical', is_regex: false },
        ],
        comment: 'Hit preview test',
      })
      ruleId = rule.id
      await page.screenshot({ path: 'test-results/MR-5-01-创建规则.png', fullPage: false })
    })

    // ---- 2. 获取命中预览 ----
    await test.step('获取命中预览', async () => {
      const res = await API.post(page, `${API_BASE}/mute-rules/${ruleId}/preview`, {
        labels: { env: 'production', severity: 'critical', job: 'api-server' },
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/MR-5-02-命中预览.png', fullPage: false })
    })

    // ---- 3. 测试不命中场景 ----
    await test.step('测试不命中场景', async () => {
      const res = await API.post(page, `${API_BASE}/mute-rules/${ruleId}/preview`, {
        labels: { env: 'staging', severity: 'warning', job: 'api-server' },
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MR-5-03-不命中预览.png', fullPage: false })
    })

    // ---- 4. 测试正则匹配 ----
    await test.step('测试正则匹配', async () => {
      // Update matchers to use regex
      await API.put(page, `${API_BASE}/mute-rules/${ruleId}`, {
        matchers: [{ name: 'job', value: 'api-.*', is_regex: true }],
      })
      const res = await API.post(page, `${API_BASE}/mute-rules/${ruleId}/preview`, {
        labels: { env: 'production', job: 'api-server' },
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MR-5-04-正则匹配.png', fullPage: false })
    })
  } finally {
    if (ruleId) await cleanupMuteRule(page, ruleId)
  }
})
