import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a task template via API and return the created object */
async function createTaskTpl(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `task-tpl-${tag}`,
    script: 'echo "hello world"',
    args: '',
    batch: 0,
    tolerance: 0,
    timeout: 60,
    hosts: JSON.stringify(['localhost']),
    note: 'Functional test task template',
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/task-tpls`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a task template by ID, ignoring errors (for cleanup) */
async function cleanupTaskTpl(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/task-tpls/${id}`)
  } catch { /* ignore */ }
}

/** Helper: cleanup a task by ID */
async function cleanupTask(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/tasks/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// TK-1: 任务模板 CRUD
// ---------------------------------------------------------------------------
test('TK-1 任务模板 CRUD', async ({ authPage: page }) => {
  let tplId: number | null = null

  try {
    await test.step('创建任务模板', async () => {
      const tpl = await createTaskTpl(page)
      tplId = tpl.id
      expect(tpl.name).toContain('task-tpl-')
      expect(tpl.script).toContain('echo')
      await page.screenshot({ path: 'test-results/TK-1-01-创建成功.png', fullPage: false })
    })

    await test.step('GET 验证任务模板已保存', async () => {
      const res = await API.get(page, `${API_BASE}/task-tpls/${tplId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(tplId)
      expect(res.data.script).toContain('echo')
      await page.screenshot({ path: 'test-results/TK-1-02-GET验证.png', fullPage: false })
    })

    await test.step('更新任务模板', async () => {
      const res = await API.put(page, `${API_BASE}/task-tpls/${tplId}`, {
        name: `updated-tpl-${uid()}`,
        script: 'echo "updated script"',
        note: 'Updated by functional test',
        timeout: 120,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/TK-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/task-tpls/${tplId}`)
      expect(res.code).toBe(0)
      expect(res.data.note).toBe('Updated by functional test')
      expect(res.data.timeout).toBe(120)
      await page.screenshot({ path: 'test-results/TK-1-04-更新验证.png', fullPage: false })
    })

    await test.step('删除任务模板', async () => {
      const res = await API.del(page, `${API_BASE}/task-tpls/${tplId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/TK-1-05-删除成功.png', fullPage: false })
    })

    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/task-tpls/${tplId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/TK-1-06-删除验证.png', fullPage: false })
    })

    tplId = null
  } finally {
    if (tplId) await cleanupTaskTpl(page, tplId)
  }
})

// ---------------------------------------------------------------------------
// TK-2: 任务直接执行
// ---------------------------------------------------------------------------
test('TK-2 任务直接执行', async ({ authPage: page }) => {
  let tplId: number | null = null
  let taskId: number | null = null

  try {
    await test.step('创建任务模板', async () => {
      const tpl = await createTaskTpl(page, {
        script: 'echo "direct execution test"',
      })
      tplId = tpl.id
      await page.screenshot({ path: 'test-results/TK-2-01-创建模板.png', fullPage: false })
    })

    await test.step('直接执行任务', async () => {
      const res = await API.post(page, `${API_BASE}/tasks`, {
        tpl_id: tplId,
        hosts: ['localhost'],
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      taskId = res.data.id
      await page.screenshot({ path: 'test-results/TK-2-02-执行任务.png', fullPage: false })
    })

    await test.step('验证任务已创建', async () => {
      const res = await API.get(page, `${API_BASE}/tasks/${taskId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(taskId)
      await page.screenshot({ path: 'test-results/TK-2-03-任务验证.png', fullPage: false })
    })
  } finally {
    if (taskId) await cleanupTask(page, taskId)
    if (tplId) await cleanupTaskTpl(page, tplId)
  }
})

// ---------------------------------------------------------------------------
// TK-3: 任务执行记录
// ---------------------------------------------------------------------------
test('TK-3 任务执行记录', async ({ authPage: page }) => {
  await test.step('获取任务列表', async () => {
    const res = await API.get(page, `${API_BASE}/tasks?page=1&page_size=20`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    await page.screenshot({ path: 'test-results/TK-3-01-任务列表.png', fullPage: false })
  })

  await test.step('验证任务列表结构', async () => {
    const res = await API.get(page, `${API_BASE}/tasks?page=1&page_size=5`)
    expect(res.code).toBe(0)
    const list = res.data?.list || res.data || []
    expect(Array.isArray(list)).toBe(true)
    await page.screenshot({ path: 'test-results/TK-3-02-列表结构.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// TK-4: 任务主机结果
// ---------------------------------------------------------------------------
test('TK-4 任务主机结果', async ({ authPage: page }) => {
  let taskId: number | undefined

  await test.step('获取最新任务', async () => {
    const res = await API.get(page, `${API_BASE}/tasks?page=1&page_size=1`)
    expect(res.code).toBe(0)
    const list = res.data?.list || res.data || []
    if (list.length > 0) {
      taskId = list[0].id
    }
    await page.screenshot({ path: 'test-results/TK-4-01-获取任务.png', fullPage: false })
  })

  if (taskId) {
    await test.step('获取任务主机结果', async () => {
      const res = await API.get(page, `${API_BASE}/tasks/${taskId}/hosts`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/TK-4-02-主机结果.png', fullPage: false })
    })

    await test.step('验证主机结果结构', async () => {
      const res = await API.get(page, `${API_BASE}/tasks/${taskId}/hosts`)
      expect(res.code).toBe(0)
      const hosts = Array.isArray(res.data) ? res.data : res.data?.list || []
      expect(Array.isArray(hosts)).toBe(true)
      await page.screenshot({ path: 'test-results/TK-4-03-结果结构.png', fullPage: false })
    })
  } else {
    await test.step('无任务 -- 跳过主机结果测试', async () => {
      await page.screenshot({ path: 'test-results/TK-4-02-无任务.png', fullPage: false })
    })
  }
})

// ---------------------------------------------------------------------------
// TK-5: 任务批量策略
// ---------------------------------------------------------------------------
test('TK-5 任务批量策略', async ({ authPage: page }) => {
  let tplId: number | null = null
  let taskId: number | null = null

  try {
    await test.step('创建任务模板', async () => {
      const tpl = await createTaskTpl(page, {
        script: 'echo "batch test"',
        batch: 5,
        tolerance: 1,
      })
      tplId = tpl.id
      expect(tpl.batch).toBe(5)
      await page.screenshot({ path: 'test-results/TK-5-01-创建模板.png', fullPage: false })
    })

    await test.step('使用模板执行任务', async () => {
      const res = await API.post(page, `${API_BASE}/tasks`, {
        tpl_id: tplId,
        hosts: ['localhost'],
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      taskId = res.data.id
      await page.screenshot({ path: 'test-results/TK-5-02-批量执行.png', fullPage: false })
    })

    await test.step('验证任务已保存', async () => {
      const res = await API.get(page, `${API_BASE}/tasks/${taskId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(taskId)
      await page.screenshot({ path: 'test-results/TK-5-03-策略验证.png', fullPage: false })
    })
  } finally {
    if (taskId) await cleanupTask(page, taskId)
    if (tplId) await cleanupTaskTpl(page, tplId)
  }
})
