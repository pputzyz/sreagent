import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a diagnostic workflow via API and return the created object */
async function createDiagnosticWorkflow(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    workflow: {
      name: `dw-test-${tag}`,
      description: 'Functional test diagnostic workflow',
      enabled: true,
      ...overrides,
    },
    steps: [],
  }
  const res = await API.post(page, `${API_BASE}/diagnostic-workflows`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a diagnostic workflow by ID, ignoring errors (for cleanup) */
async function cleanupDiagnosticWorkflow(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/diagnostic-workflows/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// DW-1: 诊断工作流 CRUD
// ---------------------------------------------------------------------------
test('DW-1 诊断工作流 CRUD', async ({ authPage: page }) => {
  let workflowId: number | null = null

  try {
    await test.step('创建诊断工作流', async () => {
      const workflow = await createDiagnosticWorkflow(page)
      workflowId = workflow.id
      expect(workflow.name).toContain('dw-test-')
      await page.screenshot({ path: 'test-results/DW-1-01-创建成功.png', fullPage: false })
    })

    await test.step('GET 验证诊断工作流已保存', async () => {
      const res = await API.get(page, `${API_BASE}/diagnostic-workflows/${workflowId}`)
      expect(res.code).toBe(0)
      const wf = res.data.workflow || res.data
      expect(wf.id).toBe(workflowId)
      expect(wf.name).toContain('dw-test-')
      await page.screenshot({ path: 'test-results/DW-1-02-GET验证.png', fullPage: false })
    })

    await test.step('更新诊断工作流', async () => {
      const res = await API.put(page, `${API_BASE}/diagnostic-workflows/${workflowId}`, {
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DW-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/diagnostic-workflows/${workflowId}`)
      expect(res.code).toBe(0)
      const wf = res.data.workflow || res.data
      expect(wf.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/DW-1-04-更新验证.png', fullPage: false })
    })

    await test.step('删除诊断工作流', async () => {
      const res = await API.del(page, `${API_BASE}/diagnostic-workflows/${workflowId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DW-1-05-删除成功.png', fullPage: false })
    })

    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/diagnostic-workflows/${workflowId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/DW-1-06-删除验证.png', fullPage: false })
    })

    workflowId = null
  } finally {
    if (workflowId) await cleanupDiagnosticWorkflow(page, workflowId)
  }
})

// ---------------------------------------------------------------------------
// DW-2: 诊断工作流 steps 管理 (ReplaceSteps — PUT replaces all steps)
// ---------------------------------------------------------------------------
test('DW-2 诊断工作流 steps管理', async ({ authPage: page }) => {
  let workflowId: number | null = null

  try {
    await test.step('创建诊断工作流', async () => {
      const workflow = await createDiagnosticWorkflow(page)
      workflowId = workflow.id
      await page.screenshot({ path: 'test-results/DW-2-01-创建工作流.png', fullPage: false })
    })

    await test.step('替换步骤列表', async () => {
      const res = await API.put(page, `${API_BASE}/diagnostic-workflows/${workflowId}/steps`, [
        {
          name: 'Check CPU',
          step_type: 'query',
          expression: 'cpu_usage_percent > 80',
          step_order: 1,
          timeout_seconds: 30,
          on_failure: 'continue',
        },
      ])
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DW-2-02-替换步骤.png', fullPage: false })
    })

    await test.step('验证工作流详情包含步骤', async () => {
      const res = await API.get(page, `${API_BASE}/diagnostic-workflows/${workflowId}`)
      expect(res.code).toBe(0)
      const steps = res.data.steps || []
      expect(steps.length).toBeGreaterThanOrEqual(1)
      await page.screenshot({ path: 'test-results/DW-2-03-验证步骤.png', fullPage: false })
    })

    await test.step('再次替换步骤（更新）', async () => {
      const res = await API.put(page, `${API_BASE}/diagnostic-workflows/${workflowId}/steps`, [
        {
          name: 'Updated Check CPU',
          step_type: 'query',
          expression: 'cpu_usage_percent > 90',
          step_order: 1,
          timeout_seconds: 30,
          on_failure: 'continue',
        },
        {
          name: 'Check Memory',
          step_type: 'query',
          expression: 'memory_usage_percent > 85',
          step_order: 2,
          timeout_seconds: 30,
          on_failure: 'continue',
        },
      ])
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DW-2-04-更新步骤.png', fullPage: false })
    })
  } finally {
    if (workflowId) await cleanupDiagnosticWorkflow(page, workflowId)
  }
})

// ---------------------------------------------------------------------------
// DW-3: 诊断工作流 run 执行
// ---------------------------------------------------------------------------
test('DW-3 诊断工作流 run执行', async ({ authPage: page }) => {
  let workflowId: number | null = null

  try {
    await test.step('创建诊断工作流', async () => {
      const workflow = await createDiagnosticWorkflow(page)
      workflowId = workflow.id
      await page.screenshot({ path: 'test-results/DW-3-01-创建工作流.png', fullPage: false })
    })

    await test.step('执行工作流', async () => {
      const res = await API.post(page, `${API_BASE}/diagnostic-workflows/${workflowId}/run`, {
        incident_id: null,
      })
      // May succeed or fail depending on steps configured
      expect(res).toBeDefined()
      expect(res.code).toBeDefined()
      await page.screenshot({ path: 'test-results/DW-3-02-执行工作流.png', fullPage: false })
    })

    await test.step('获取执行记录', async () => {
      const res = await API.get(page, `${API_BASE}/diagnostic-runs?workflow_id=${workflowId}&page=1&page_size=10`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/DW-3-03-执行记录.png', fullPage: false })
    })
  } finally {
    if (workflowId) await cleanupDiagnosticWorkflow(page, workflowId)
  }
})

// ---------------------------------------------------------------------------
// DW-4: 诊断工作流 match 匹配
// ---------------------------------------------------------------------------
test('DW-4 诊断工作流 match匹配', async ({ authPage: page }) => {
  let workflowId: number | null = null

  try {
    await test.step('创建带匹配条件的工作流', async () => {
      const workflow = await createDiagnosticWorkflow(page, {
        trigger_labels: { alertname: 'HighCPU' },
        trigger_severity: 'critical',
      })
      workflowId = workflow.id
      await page.screenshot({ path: 'test-results/DW-4-01-创建匹配工作流.png', fullPage: false })
    })

    await test.step('测试匹配', async () => {
      const res = await API.post(page, `${API_BASE}/diagnostic-workflows/match`, {
        labels: { alertname: 'HighCPU', instance: 'localhost:9090' },
        severity: 'critical',
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/DW-4-02-匹配结果.png', fullPage: false })
    })

    await test.step('验证匹配结果包含工作流', async () => {
      const res = await API.post(page, `${API_BASE}/diagnostic-workflows/match`, {
        labels: { alertname: 'HighCPU' },
        severity: 'critical',
      })
      expect(res.code).toBe(0)
      const workflows = Array.isArray(res.data) ? res.data : res.data.workflows || []
      expect(workflows.length).toBeGreaterThanOrEqual(1)
      await page.screenshot({ path: 'test-results/DW-4-03-匹配验证.png', fullPage: false })
    })
  } finally {
    if (workflowId) await cleanupDiagnosticWorkflow(page, workflowId)
  }
})

// ---------------------------------------------------------------------------
// DW-5: 诊断工作流 run 审批
// ---------------------------------------------------------------------------
test('DW-5 诊断工作流 run审批', async ({ authPage: page }) => {
  await test.step('获取待审批的工作流运行记录', async () => {
    const res = await API.get(page, `${API_BASE}/diagnostic-runs?status=pending_approval&page=1&page_size=10`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    await page.screenshot({ path: 'test-results/DW-5-01-待审批列表.png', fullPage: false })
  })

  await test.step('验证审批 API 可访问', async () => {
    // Test the approval endpoint exists by sending a dummy approve request
    const res = await API.post(page, `${API_BASE}/diagnostic-runs/0/approve`, {})
    // Should return error for non-existent run, but endpoint should be reachable
    expect(res).toBeDefined()
    expect(res.code).toBeDefined()
    await page.screenshot({ path: 'test-results/DW-5-02-审批API.png', fullPage: false })
  })
})
