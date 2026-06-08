import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a recording rule via API and return the created object */
async function createRecordingRule(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `rr-test-${tag}`,
    prom_ql: `sum(rate(http_requests_total{job="test-${tag}"}[5m]))`,
    interval: '15s',
    datasource_type: 'prometheus',
    status: 'active',
    labels: { env: 'test', run: tag },
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/recording-rules`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a recording rule by ID, ignoring errors (for cleanup) */
async function cleanupRecordingRule(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/recording-rules/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// RR-1: 录制规则 CRUD
// ---------------------------------------------------------------------------
test('RR-1 录制规则 CRUD', async ({ authPage: page }) => {
  let ruleId: number | null = null

  try {
    // ---- 1. 创建录制规则 ----
    await test.step('创建录制规则', async () => {
      const rule = await createRecordingRule(page, {
        description: 'CRUD 测试录制规则',
      })
      ruleId = rule.id
      expect(rule.name).toContain('rr-test-')
      expect(rule.prom_ql).toContain('rate(http_requests_total')
      expect(rule.status).toBe('active')
      await page.screenshot({ path: 'test-results/RR-1-01-创建成功.png', fullPage: false })
    })

    // ---- 2. GET 验证 ----
    await test.step('GET 验证录制规则已保存', async () => {
      const res = await API.get(page, `${API_BASE}/recording-rules/${ruleId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(ruleId)
      expect(res.data.status).toBe('active')
      expect(res.data.prom_ql).toContain('rate(http_requests_total')
      await page.screenshot({ path: 'test-results/RR-1-02-GET验证.png', fullPage: false })
    })

    // ---- 3. 更新录制规则 ----
    await test.step('更新录制规则', async () => {
      const res = await API.put(page, `${API_BASE}/recording-rules/${ruleId}`, {
        name: `updated-rr-${uid()}`,
        interval: '30s',
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/RR-1-03-更新成功.png', fullPage: false })
    })

    // ---- 4. 验证更新生效 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/recording-rules/${ruleId}`)
      expect(res.code).toBe(0)
      expect(res.data.interval).toBe('30s')
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/RR-1-04-更新验证.png', fullPage: false })
    })

    // ---- 5. 删除录制规则 ----
    await test.step('删除录制规则', async () => {
      const res = await API.del(page, `${API_BASE}/recording-rules/${ruleId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/RR-1-05-删除成功.png', fullPage: false })
    })

    // ---- 6. 验证删除生效 ----
    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/recording-rules/${ruleId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/RR-1-06-删除验证.png', fullPage: false })
    })

    ruleId = null
  } finally {
    if (ruleId) await cleanupRecordingRule(page, ruleId)
  }
})

// ---------------------------------------------------------------------------
// RR-2: 录制规则批量创建
// ---------------------------------------------------------------------------
test('RR-2 录制规则批量创建', async ({ authPage: page }) => {
  const ruleIds: number[] = []

  try {
    await test.step('批量创建3个录制规则', async () => {
      for (let i = 0; i < 3; i++) {
        const rule = await createRecordingRule(page, {
          interval: `${(i + 1) * 10}s`,
        })
        ruleIds.push(rule.id)
      }
      expect(ruleIds.length).toBe(3)
      await page.screenshot({ path: 'test-results/RR-2-01-批量创建成功.png', fullPage: false })
    })

    await test.step('验证批量创建的规则均存在', async () => {
      for (const id of ruleIds) {
        const res = await API.get(page, `${API_BASE}/recording-rules/${id}`)
        expect(res.code).toBe(0)
        expect(res.data.id).toBe(id)
      }
      await page.screenshot({ path: 'test-results/RR-2-02-批量验证.png', fullPage: false })
    })

    await test.step('列表查询包含批量创建的规则', async () => {
      const res = await API.get(page, `${API_BASE}/recording-rules?page=1&page_size=100`)
      expect(res.code).toBe(0)
      const list = res.data.list || res.data || []
      expect(list.length).toBeGreaterThanOrEqual(3)
      await page.screenshot({ path: 'test-results/RR-2-03-列表查询.png', fullPage: false })
    })
  } finally {
    for (const id of ruleIds) await cleanupRecordingRule(page, id)
  }
})

// ---------------------------------------------------------------------------
// RR-3: 录制规则 batch-delete
// ---------------------------------------------------------------------------
test('RR-3 录制规则 batch-delete', async ({ authPage: page }) => {
  const ruleIds: number[] = []

  try {
    await test.step('创建3个录制规则用于批量删除', async () => {
      for (let i = 0; i < 3; i++) {
        const rule = await createRecordingRule(page)
        ruleIds.push(rule.id)
      }
      expect(ruleIds.length).toBe(3)
      await page.screenshot({ path: 'test-results/RR-3-01-创建待删除规则.png', fullPage: false })
    })

    await test.step('批量删除', async () => {
      const res = await API.post(page, `${API_BASE}/recording-rules/batch-delete`, { ids: ruleIds })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/RR-3-02-批量删除成功.png', fullPage: false })
    })

    await test.step('验证全部已删除', async () => {
      for (const id of ruleIds) {
        const res = await API.get(page, `${API_BASE}/recording-rules/${id}`)
        expect(res.code).not.toBe(0)
      }
      await page.screenshot({ path: 'test-results/RR-3-03-删除验证.png', fullPage: false })
    })

    ruleIds.length = 0
  } finally {
    for (const id of ruleIds) await cleanupRecordingRule(page, id)
  }
})

// ---------------------------------------------------------------------------
// RR-4: 录制规则 fields 更新
// ---------------------------------------------------------------------------
test('RR-4 录制规则 fields 更新', async ({ authPage: page }) => {
  let ruleId: number | null = null

  try {
    await test.step('创建录制规则', async () => {
      const rule = await createRecordingRule(page)
      ruleId = rule.id
      await page.screenshot({ path: 'test-results/RR-4-01-创建规则.png', fullPage: false })
    })

    await test.step('部分字段更新(interval)', async () => {
      const res = await API.put(page, `${API_BASE}/recording-rules/${ruleId}`, {
        interval: '60s',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/RR-4-02-字段更新.png', fullPage: false })
    })

    await test.step('验证字段更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/recording-rules/${ruleId}`)
      expect(res.code).toBe(0)
      expect(res.data.interval).toBe('60s')
      await page.screenshot({ path: 'test-results/RR-4-03-字段验证.png', fullPage: false })
    })

    await test.step('更新 labels 字段', async () => {
      const res = await API.put(page, `${API_BASE}/recording-rules/${ruleId}`, {
        labels: { env: 'production', team: 'sre' },
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/RR-4-04-Labels更新.png', fullPage: false })
    })

    await test.step('验证 labels 更新', async () => {
      const res = await API.get(page, `${API_BASE}/recording-rules/${ruleId}`)
      expect(res.code).toBe(0)
      expect(res.data.labels.env).toBe('production')
      expect(res.data.labels.team).toBe('sre')
      await page.screenshot({ path: 'test-results/RR-4-05-Labels验证.png', fullPage: false })
    })
  } finally {
    if (ruleId) await cleanupRecordingRule(page, ruleId)
  }
})
