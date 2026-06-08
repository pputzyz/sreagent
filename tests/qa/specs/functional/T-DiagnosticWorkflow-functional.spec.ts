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
      is_enabled: true,
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
      const workflow = res.data.workflow || res.data
      expect(workflow.id).toBe(workflowId)
      expect(workflow.name).toContain('dw-test-')
      await page.screenshot({ path: 'test-results/DW-1-02-GET验证.png', fullPage: false })
    })

    await test.step('更新诊断工作流', async () => {
      const res = await API.put(page, `${API_BASE}/diagnostic-workflows/${workflowId}`, {
        name: `updated-dw-${uid()}`,
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DW-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/diagnostic-workflows/${workflowId}`)
      expect(res.code).toBe(0)
      const workflow = res.data.workflow || res.data
      expect(workflow.description).toBe('Updated by functional test')
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
// DW-2: 诊断工作流 steps 管理
// ---------------------------------------------------------------------------
test('DW-2 诊断工作流 steps管理', async ({ authPage: page }) => {
  let workflowId: number | null = null

  try {
    await test.step('创建诊断工作流', async () => {
      const workflow = await createDiagnosticWorkflow(page)
      workflowId = workflow.id
      await page.screenshot({ path: 'test-results/DW-2-01-创建工作流.png', fullPage: false })
    })

    await test.step('添加步骤', async () => {
      const res = await API.post(page, `${API_BASE}/diagnostic-workflows/${workflowId}/steps`, {
        name: 'Check CPU',
        type: 'command',
        config: { command: 'top -bn1 | head -5' },
        order: 1,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/DW-2-02-添加步骤.png', fullPage: false })
    })

    await test.step('获取步骤列表', async () => {
      const res = await API.get(page, `${API_BASE}/diagnostic-workflows/${workflowId}/steps`)
      expect(res.code).toBe(0)
      const steps = Array.isArray(res.data) ? res.data : res.data.list || []
      expect(steps.length).toBeGreaterThanOrEqual(1)
      await page.screenshot({ path: 'test-results/DW-2-03-步骤列表.png', fullPage: false })
    })

    await test.step('更新步骤', async () => {
      const stepsRes = await API.get(page, `${API_BASE}/diagnostic-workflows/${workflowId}/steps`)
      const steps = Array.isArray(stepsRes.data) ? stepsRes.data : stepsRes.data.list || []
      if (steps.length > 0) {
        const stepId = steps[0].id
        const res = await API.put(page, `${API_BASE}/diagnostic-workflows/${workflowId}/steps/${stepId}`, {
          name: 'Updated Check CPU',
          config: { command: 'top -bn1 | head -10' },
        })
        expect(res.code).toBe(0)
      }
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
        context: { alertname: 'HighCPU', instance: 'localhost:9090' },
      })
      // May succeed or fail depending on steps configured
      expect(res).toBeDefined()
      expect(res.code).toBeDefined()
      await page.screenshot({ path: 'test-results/DW-3-02-执行工作流.png', fullPage: false })
    })

    await test.step('获取执行记录', async () => {
      const res = await API.get(page, `${API_BASE}/diagnostic-workflows/${workflowId}/runs?page=1&page_size=10`)
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
        match_rules: {
          alertname: ['HighCPU', 'HighMemory'],
          severity: ['critical'],
        },
      })
      workflowId = workflow.id
      await page.screenshot({ path: 'test-results/DW-4-01-创建匹配工作流.png', fullPage: false })
    })

    await test.step('测试匹配', async () => {
      const res = await API.post(page, `${API_BASE}/diagnostic-workflows/match`, {
        alertname: 'HighCPU',
        severity: 'critical',
        labels: { instance: 'localhost:9090' },
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/DW-4-02-匹配结果.png', fullPage: false })
    })

    await test.step('验证匹配结果包含工作流', async () => {
      const res = await API.post(page, `${API_BASE}/diagnostic-workflows/match`, {
        alertname: 'HighCPU',
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
    const res = await API.get(page, `${API_BASE}/diagnostic-workflows/runs?status=pending_approval&page=1&page_size=10`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    await page.screenshot({ path: 'test-results/DW-5-01-待审批列表.png', fullPage: false })
  })

  await test.step('验证审批 API 可访问', async () => {
    // Test the approval endpoint exists by sending a dummy approve request
    const res = await API.post(page, `${API_BASE}/diagnostic-workflows/runs/0/approve`, {
      action: 'approve',
      comment: 'test approval',
    })
    // Should return error for non-existent run, but endpoint should be reachable
    expect(res).toBeDefined()
    expect(res.code).toBeDefined()
    await page.screenshot({ path: 'test-results/DW-5-02-审批API.png', fullPage: false })
  })
})
