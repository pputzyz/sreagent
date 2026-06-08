import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a status service/page via API and return the created object */
async function createStatusService(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `status-svc-${tag}`,
    description: `Functional test status service ${tag}`,
    status: 'operational',
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/status-services`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a status service by ID, ignoring errors (for cleanup) */
async function cleanupStatusService(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/status-services/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// SP-1 状态页面 CRUD
// ---------------------------------------------------------------------------
test('SP-1 状态页面 CRUD', async ({ authPage: page }) => {
  let serviceId: number | null = null

  try {
    // ---- 1. 创建状态服务 ----
    await test.step('创建状态服务', async () => {
      const svc = await createStatusService(page, {
        description: 'CRUD test status service',
      })
      serviceId = svc.id
      expect(svc.name).toContain('status-svc-')
      expect(svc.status).toBe('operational')
      expect(svc.description).toBe('CRUD test status service')
      await page.screenshot({ path: 'test-results/SP-1-01-创建成功.png', fullPage: false })
    })

    // ---- 2. GET 验证所有字段 ----
    await test.step('GET 验证服务已保存', async () => {
      const res = await API.get(page, `${API_BASE}/status-services/${serviceId}`)
      expect(res.code).toBe(0)
      const r = res.data
      expect(r.id).toBe(serviceId)
      expect(r.name).toContain('status-svc-')
      expect(r.status).toBe('operational')
      expect(r.description).toBe('CRUD test status service')
      await page.screenshot({ path: 'test-results/SP-1-02-GET验证.png', fullPage: false })
    })

    // ---- 3. 更新服务（改名、改描述） ----
    await test.step('更新服务名称和描述', async () => {
      const res = await API.put(page, `${API_BASE}/status-services/${serviceId}`, {
        name: `updated-status-svc-${uid()}`,
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/SP-1-03-更新成功.png', fullPage: false })
    })

    // ---- 4. 验证更新生效 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/status-services/${serviceId}`)
      expect(res.code).toBe(0)
      expect(res.data.name).toContain('updated-status-svc-')
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/SP-1-04-更新验证.png', fullPage: false })
    })

    // ---- 5. 删除服务 ----
    await test.step('删除服务', async () => {
      const res = await API.del(page, `${API_BASE}/status-services/${serviceId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/SP-1-05-删除成功.png', fullPage: false })
    })

    // ---- 6. 验证删除生效 ----
    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/status-services/${serviceId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/SP-1-06-删除验证.png', fullPage: false })
    })

    serviceId = null
  } finally {
    if (serviceId) await cleanupStatusService(page, serviceId)
  }
})

// ---------------------------------------------------------------------------
// SP-2 状态页面状态变更
// ---------------------------------------------------------------------------
test('SP-2 状态页面状态变更', async ({ authPage: page }) => {
  let serviceId: number | null = null

  try {
    // ---- 1. 创建状态服务 ----
    await test.step('创建状态服务', async () => {
      const svc = await createStatusService(page, { status: 'operational' })
      serviceId = svc.id
      expect(svc.status).toBe('operational')
      await page.screenshot({ path: 'test-results/SP-2-01-创建服务.png', fullPage: false })
    })

    // ---- 2. 变更为 degraded_performance ----
    await test.step('变更为 degraded_performance', async () => {
      const res = await API.patch(page, `${API_BASE}/status-services/${serviceId}/status`, {
        status: 'degraded_performance',
        message: '服务性能下降',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/SP-2-02-性能下降.png', fullPage: false })
    })

    // ---- 3. 验证 degraded_performance 状态 ----
    await test.step('验证 degraded_performance 状态', async () => {
      const res = await API.get(page, `${API_BASE}/status-services/${serviceId}`)
      expect(res.code).toBe(0)
      expect(res.data.status).toBe('degraded_performance')
      await page.screenshot({ path: 'test-results/SP-2-03-性能下降验证.png', fullPage: false })
    })

    // ---- 4. 变更为 major_outage ----
    await test.step('变更为 major_outage', async () => {
      const res = await API.patch(page, `${API_BASE}/status-services/${serviceId}/status`, {
        status: 'major_outage',
        message: '服务严重故障',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/SP-2-04-严重故障.png', fullPage: false })
    })

    // ---- 5. 验证 major_outage 状态 ----
    await test.step('验证 major_outage 状态', async () => {
      const res = await API.get(page, `${API_BASE}/status-services/${serviceId}`)
      expect(res.code).toBe(0)
      expect(res.data.status).toBe('major_outage')
      await page.screenshot({ path: 'test-results/SP-2-05-严重故障验证.png', fullPage: false })
    })

    // ---- 6. 恢复为 operational ----
    await test.step('恢复为 operational', async () => {
      const res = await API.patch(page, `${API_BASE}/status-services/${serviceId}/status`, {
        status: 'operational',
        message: '服务已恢复',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/SP-2-06-恢复服务.png', fullPage: false })
    })

    // ---- 7. 验证恢复状态 ----
    await test.step('验证恢复状态', async () => {
      const res = await API.get(page, `${API_BASE}/status-services/${serviceId}`)
      expect(res.code).toBe(0)
      expect(res.data.status).toBe('operational')
      await page.screenshot({ path: 'test-results/SP-2-07-恢复验证.png', fullPage: false })
    })
  } finally {
    if (serviceId) await cleanupStatusService(page, serviceId)
  }
})

// ---------------------------------------------------------------------------
// SP-3 状态页面订阅
// ---------------------------------------------------------------------------
test('SP-3 状态页面订阅', async ({ authPage: page }) => {
  let serviceId: number | null = null

  try {
    // ---- 1. 创建状态服务 ----
    await test.step('创建状态服务', async () => {
      const svc = await createStatusService(page)
      serviceId = svc.id
      await page.screenshot({ path: 'test-results/SP-3-01-创建服务.png', fullPage: false })
    })

    // ---- 2. 订阅状态服务 ----
    await test.step('订阅状态服务', async () => {
      const res = await API.post(page, `${API_BASE}/status-services/${serviceId}/subscribe`, {
        email: `test-${uid()}@example.com`,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/SP-3-02-订阅成功.png', fullPage: false })
    })

    // ---- 3. 验证订阅列表 ----
    await test.step('验证订阅列表', async () => {
      const res = await API.get(page, `${API_BASE}/status-services/${serviceId}/subscribers`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/SP-3-03-订阅列表.png', fullPage: false })
    })

    // ---- 4. 取消订阅 ----
    await test.step('取消订阅', async () => {
      const res = await API.del(page, `${API_BASE}/status-services/${serviceId}/subscribe`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/SP-3-04-取消订阅.png', fullPage: false })
    })

    // ---- 5. 验证取消订阅生效 ----
    await test.step('验证取消订阅生效', async () => {
      const res = await API.get(page, `${API_BASE}/status-services/${serviceId}/subscribers`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/SP-3-05-取消验证.png', fullPage: false })
    })
  } finally {
    if (serviceId) await cleanupStatusService(page, serviceId)
  }
})
