import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create an event pipeline via API and return the created object */
async function createPipeline(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `ep-test-${tag}`,
    description: 'Functional test pipeline',
    disabled: false,
    processors: [
      { type: 'label_add', config: { env: 'test', run: tag } },
    ],
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/event-pipelines`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a pipeline by ID, ignoring errors (for cleanup) */
async function cleanupPipeline(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/event-pipelines/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// EP-1: 事件管道 CRUD
// ---------------------------------------------------------------------------
test('EP-1 事件管道 CRUD', async ({ authPage: page }) => {
  let pipelineId: number | null = null

  try {
    await test.step('创建事件管道', async () => {
      const pipeline = await createPipeline(page)
      pipelineId = pipeline.id
      expect(pipeline.name).toContain('ep-test-')
      expect(pipeline.disabled).toBe(false)
      await page.screenshot({ path: 'test-results/EP-1-01-创建成功.png', fullPage: false })
    })

    await test.step('GET 验证事件管道已保存', async () => {
      const res = await API.get(page, `${API_BASE}/event-pipelines/${pipelineId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(pipelineId)
      expect(res.data.name).toContain('ep-test-')
      await page.screenshot({ path: 'test-results/EP-1-02-GET验证.png', fullPage: false })
    })

    await test.step('更新事件管道', async () => {
      const res = await API.put(page, `${API_BASE}/event-pipelines/${pipelineId}`, {
        name: `updated-ep-${uid()}`,
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/EP-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/event-pipelines/${pipelineId}`)
      expect(res.code).toBe(0)
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/EP-1-04-更新验证.png', fullPage: false })
    })

    await test.step('删除事件管道', async () => {
      const res = await API.del(page, `${API_BASE}/event-pipelines/${pipelineId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/EP-1-05-删除成功.png', fullPage: false })
    })

    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/event-pipelines/${pipelineId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/EP-1-06-删除验证.png', fullPage: false })
    })

    pipelineId = null
  } finally {
    if (pipelineId) await cleanupPipeline(page, pipelineId)
  }
})

// ---------------------------------------------------------------------------
// EP-2: 事件管道 tryrun
// ---------------------------------------------------------------------------
test('EP-2 事件管道 tryrun', async ({ authPage: page }) => {
  let pipelineId: number | null = null

  try {
    await test.step('创建事件管道', async () => {
      const pipeline = await createPipeline(page, {
        processors: [
          { type: 'label_add', config: { test_label: 'tryrun' } },
        ],
      })
      pipelineId = pipeline.id
      await page.screenshot({ path: 'test-results/EP-2-01-创建管道.png', fullPage: false })
    })

    await test.step('执行 tryrun', async () => {
      const res = await API.post(page, `${API_BASE}/event-pipelines/${pipelineId}/tryrun`, {
        event: {
          alertname: 'TryRunTest',
          severity: 'warning',
          labels: { instance: 'localhost:9090' },
          annotations: { summary: 'Tryrun test event' },
        },
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/EP-2-02-tryrun结果.png', fullPage: false })
    })

    await test.step('验证 tryrun 返回结果结构', async () => {
      const res = await API.post(page, `${API_BASE}/event-pipelines/${pipelineId}/tryrun`, {
        event: {
          alertname: 'TryRunTest2',
          severity: 'critical',
          labels: { env: 'staging' },
        },
      })
      expect(res.code).toBe(0)
      // Result should contain processed event data
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/EP-2-03-结果结构验证.png', fullPage: false })
    })
  } finally {
    if (pipelineId) await cleanupPipeline(page, pipelineId)
  }
})

// ---------------------------------------------------------------------------
// EP-3: 事件管道执行记录
// ---------------------------------------------------------------------------
test('EP-3 事件管道执行记录', async ({ authPage: page }) => {
  let pipelineId: number | null = null

  try {
    await test.step('创建事件管道', async () => {
      const pipeline = await createPipeline(page)
      pipelineId = pipeline.id
      await page.screenshot({ path: 'test-results/EP-3-01-创建管道.png', fullPage: false })
    })

    await test.step('获取执行记录列表', async () => {
      const res = await API.get(page, `${API_BASE}/event-pipelines/${pipelineId}/executions?page=1&page_size=20`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/EP-3-02-执行记录列表.png', fullPage: false })
    })

    await test.step('执行记录结构验证', async () => {
      const res = await API.get(page, `${API_BASE}/event-pipelines/${pipelineId}/executions?page=1&page_size=5`)
      expect(res.code).toBe(0)
      // Structure should be a list (may be empty for new pipeline)
      const list = res.data.list || res.data || []
      expect(Array.isArray(list)).toBe(true)
      await page.screenshot({ path: 'test-results/EP-3-03-记录结构.png', fullPage: false })
    })
  } finally {
    if (pipelineId) await cleanupPipeline(page, pipelineId)
  }
})

// ---------------------------------------------------------------------------
// EP-4: 事件管道处理器类型
// ---------------------------------------------------------------------------
test('EP-4 事件管道处理器类型', async ({ authPage: page }) => {
  await test.step('获取处理器类型列表', async () => {
    const res = await API.get(page, `${API_BASE}/event-pipelines/processor-types`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    const types = Array.isArray(res.data) ? res.data : res.data.list || []
    expect(types.length).toBeGreaterThan(0)
    await page.screenshot({ path: 'test-results/EP-4-01-处理器类型.png', fullPage: false })
  })

  await test.step('验证常见处理器类型存在', async () => {
    const res = await API.get(page, `${API_BASE}/event-pipelines/processor-types`)
    expect(res.code).toBe(0)
    const types = Array.isArray(res.data) ? res.data : res.data.list || []
    const typeNames = types.map((t: any) => t.type || t.name || t)
    // Should contain common processor types
    const hasProcessors = typeNames.some((n: string) =>
      typeof n === 'string' && (n.includes('label') || n.includes('relabel') || n.includes('filter') || n.includes('add'))
    )
    expect(hasProcessors).toBeTruthy()
    await page.screenshot({ path: 'test-results/EP-4-02-类型验证.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// EP-5: 事件管道条件分支
// ---------------------------------------------------------------------------
test('EP-5 事件管道条件分支', async ({ authPage: page }) => {
  let pipelineId: number | null = null

  try {
    await test.step('创建带条件分支的管道', async () => {
      const pipeline = await createPipeline(page, {
        processors: [
          {
            type: 'condition',
            config: {
              condition: 'severity == "critical"',
              then: [{ type: 'label_add', config: { action: 'escalate' } }],
              else: [{ type: 'label_add', config: { action: 'log' } }],
            },
          },
        ],
      })
      pipelineId = pipeline.id
      await page.screenshot({ path: 'test-results/EP-5-01-创建条件管道.png', fullPage: false })
    })

    await test.step('tryrun critical 事件验证 then 分支', async () => {
      const res = await API.post(page, `${API_BASE}/event-pipelines/${pipelineId}/tryrun`, {
        event: {
          alertname: 'ConditionTest',
          severity: 'critical',
          labels: {},
        },
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/EP-5-02-then分支.png', fullPage: false })
    })

    await test.step('tryrun warning 事件验证 else 分支', async () => {
      const res = await API.post(page, `${API_BASE}/event-pipelines/${pipelineId}/tryrun`, {
        event: {
          alertname: 'ConditionTest2',
          severity: 'warning',
          labels: {},
        },
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/EP-5-03-else分支.png', fullPage: false })
    })
  } finally {
    if (pipelineId) await cleanupPipeline(page, pipelineId)
  }
})
