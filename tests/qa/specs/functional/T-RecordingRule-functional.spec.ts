import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: get the first available datasource ID */
async function getDatasourceId(page: any): Promise<number> {
  const res = await API.get(page, '/api/v1/datasources?page=1&page_size=100')
  expect(res.code).toBe(0)
  const ds = res.data.list?.[0]
  expect(ds).toBeDefined()
  return ds.id
}

/** Helper: create a recording rule via API and return the created object */
async function createRecordingRule(page: any, datasourceId: number, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `rr-test-${tag}`,
    prom_ql: `sum(rate(http_requests_total{job="test-${tag}"}[5m]))`,
    group_id: 0,
    datasource_ids: [datasourceId],
    cron_pattern: '@every 60s',
    disabled: 0,
    note: 'Functional test recording rule',
    append_tags: [`env:test`, `run:${tag}`],
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

/** Helper: find a recording rule by ID from the list */
async function findRecordingRule(page: any, id: number): Promise<any | null> {
  const res = await API.get(page, `${API_BASE}/recording-rules?page=1&page_size=100`)
  if (res.code !== 0) return null
  const list = res.data?.list || res.data || []
  return list.find((r: any) => r.id === id) || null
}

// ---------------------------------------------------------------------------
// RR-1: 录制规则 CRUD
// ---------------------------------------------------------------------------
test('RR-1 录制规则 CRUD', async ({ authPage: page }) => {
  let ruleId: number | null = null
  let datasourceId: number

  try {
    await test.step('获取数据源 ID', async () => {
      datasourceId = await getDatasourceId(page)
    })

    // ---- 1. 创建录制规则 ----
    await test.step('创建录制规则', async () => {
      const rule = await createRecordingRule(page, datasourceId, {
        note: 'CRUD 测试录制规则',
      })
      ruleId = rule.id
      expect(rule.name).toContain('rr-test-')
      expect(rule.prom_ql).toContain('rate(http_requests_total')
      expect(rule.disabled).toBe(0)
      await page.screenshot({ path: 'test-results/RR-1-01-创建成功.png', fullPage: false })
    })

    // ---- 2. 验证录制规则存在于列表 ----
    await test.step('验证录制规则存在于列表', async () => {
      const found = await findRecordingRule(page, ruleId!)
      expect(found).toBeTruthy()
      expect(found.id).toBe(ruleId)
      expect(found.disabled).toBe(0)
      expect(found.prom_ql).toContain('rate(http_requests_total')
      await page.screenshot({ path: 'test-results/RR-1-02-列表验证.png', fullPage: false })
    })

    // ---- 3. 更新录制规则 ----
    await test.step('更新录制规则', async () => {
      const res = await API.put(page, `${API_BASE}/recording-rules/${ruleId}`, {
        name: `updated-rr-${uid()}`,
        prom_ql: `sum(rate(http_requests_total{job="updated"}[5m]))`,
        datasource_ids: [datasourceId],
        cron_pattern: '@every 30s',
        note: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/RR-1-03-更新成功.png', fullPage: false })
    })

    // ---- 4. 验证更新生效 ----
    await test.step('验证更新生效', async () => {
      const found = await findRecordingRule(page, ruleId!)
      expect(found).toBeTruthy()
      expect(found.cron_pattern).toBe('@every 30s')
      expect(found.note).toBe('Updated by functional test')
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
      const found = await findRecordingRule(page, ruleId!)
      expect(found).toBeFalsy()
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
  let datasourceId: number

  try {
    await test.step('获取数据源 ID', async () => {
      datasourceId = await getDatasourceId(page)
    })

    await test.step('批量创建3个录制规则', async () => {
      const rules = Array.from({ length: 3 }, (_, i) => ({
        name: `rr-batch-${uid()}`,
        prom_ql: `sum(rate(http_requests_total{job="batch-${i}"}[5m]))`,
        group_id: 0,
        datasource_ids: [datasourceId],
        cron_pattern: `@every ${(i + 1) * 10}s`,
        disabled: 0,
        note: `Batch test rule ${i}`,
      }))
      const res = await API.post(page, `${API_BASE}/recording-rules/batch`, { group_id: 0, rules })
      expect(res.code).toBe(0)
      const results = Array.isArray(res.data) ? res.data : []
      for (const r of results) {
        if (r.id) ruleIds.push(r.id)
      }
      expect(ruleIds.length).toBeGreaterThanOrEqual(1)
      await page.screenshot({ path: 'test-results/RR-2-01-批量创建成功.png', fullPage: false })
    })

    await test.step('验证批量创建的规则均存在于列表', async () => {
      const res = await API.get(page, `${API_BASE}/recording-rules?page=1&page_size=100`)
      expect(res.code).toBe(0)
      const list = res.data?.list || res.data || []
      for (const id of ruleIds) {
        const found = list.find((r: any) => r.id === id)
        expect(found).toBeTruthy()
      }
      await page.screenshot({ path: 'test-results/RR-2-02-批量验证.png', fullPage: false })
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
  let datasourceId: number

  try {
    await test.step('获取数据源 ID', async () => {
      datasourceId = await getDatasourceId(page)
    })

    await test.step('创建3个录制规则用于批量删除', async () => {
      for (let i = 0; i < 3; i++) {
        const rule = await createRecordingRule(page, datasourceId)
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
      const res = await API.get(page, `${API_BASE}/recording-rules?page=1&page_size=100`)
      const list = res.data?.list || res.data || []
      for (const id of ruleIds) {
        const found = list.find((r: any) => r.id === id)
        expect(found).toBeFalsy()
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
  let datasourceId: number

  try {
    await test.step('获取数据源 ID', async () => {
      datasourceId = await getDatasourceId(page)
    })

    await test.step('创建录制规则', async () => {
      const rule = await createRecordingRule(page, datasourceId)
      ruleId = rule.id
      await page.screenshot({ path: 'test-results/RR-4-01-创建规则.png', fullPage: false })
    })

    await test.step('部分字段更新(cron_pattern)', async () => {
      const res = await API.put(page, `${API_BASE}/recording-rules/fields`, {
        ids: [ruleId],
        fields: { cron_pattern: '@every 120s' },
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/RR-4-02-字段更新.png', fullPage: false })
    })

    await test.step('验证字段更新生效', async () => {
      const found = await findRecordingRule(page, ruleId!)
      expect(found).toBeTruthy()
      expect(found.cron_pattern).toBe('@every 120s')
      await page.screenshot({ path: 'test-results/RR-4-03-字段验证.png', fullPage: false })
    })

    await test.step('更新 append_tags 字段', async () => {
      const res = await API.put(page, `${API_BASE}/recording-rules/fields`, {
        ids: [ruleId],
        fields: { append_tags: ['env:production', 'team:sre'] },
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/RR-4-04-Tags更新.png', fullPage: false })
    })

    await test.step('验证 append_tags 更新', async () => {
      const found = await findRecordingRule(page, ruleId!)
      expect(found).toBeTruthy()
      const tags = found.append_tags || []
      expect(tags).toContain('env:production')
      expect(tags).toContain('team:sre')
      await page.screenshot({ path: 'test-results/RR-4-05-Tags验证.png', fullPage: false })
    })
  } finally {
    if (ruleId) await cleanupRecordingRule(page, ruleId)
  }
})
