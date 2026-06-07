import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a rule template via API and return the created object */
async function createTemplate(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `template-${tag}`,
    description: `Functional test template ${tag}`,
    category: 'node',
    expression: `up{job="test-${tag}"} == 0`,
    severity: 'warning',
    for_duration: '5m',
    labels: { env: 'test', run: tag },
    annotations: { summary: 'Template test rule' },
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/alert-rule-templates`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a template by ID, ignoring errors (for cleanup) */
async function cleanupTemplate(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/alert-rule-templates/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// RT-1 规则模板 CRUD
// ---------------------------------------------------------------------------
test('RT-1 规则模板 CRUD', async ({ authPage: page }) => {
  let templateId: number | null = null

  try {
    // ---- 1. 创建规则模板 ----
    await test.step('创建规则模板', async () => {
      const tpl = await createTemplate(page, {
        description: 'CRUD test template',
        category: 'application',
      })
      templateId = tpl.id
      expect(tpl.name).toContain('template-')
      expect(tpl.description).toBe('CRUD test template')
      expect(tpl.category).toBe('application')
      await page.screenshot({ path: 'test-results/RT-1-01-创建成功.png', fullPage: false })
    })

    // ---- 2. GET 验证所有字段 ----
    await test.step('GET 验证模板已保存', async () => {
      const res = await API.get(page, `${API_BASE}/alert-rule-templates/${templateId}`)
      expect(res.code).toBe(0)
      const r = res.data
      expect(r.id).toBe(templateId)
      expect(r.name).toContain('template-')
      expect(r.severity).toBe('warning')
      expect(r.expression).toContain('up{job="test-')
      expect(r.category).toBe('application')
      await page.screenshot({ path: 'test-results/RT-1-02-GET验证.png', fullPage: false })
    })

    // ---- 3. 更新模板（改名、改分类） ----
    await test.step('更新模板名称和分类', async () => {
      const res = await API.put(page, `${API_BASE}/alert-rule-templates/${templateId}`, {
        name: `updated-template-${uid()}`,
        category: 'database',
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/RT-1-03-更新成功.png', fullPage: false })
    })

    // ---- 4. 验证更新生效 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/alert-rule-templates/${templateId}`)
      expect(res.code).toBe(0)
      expect(res.data.name).toContain('updated-template-')
      expect(res.data.category).toBe('database')
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/RT-1-04-更新验证.png', fullPage: false })
    })

    // ---- 5. 删除模板 ----
    await test.step('删除模板', async () => {
      const res = await API.del(page, `${API_BASE}/alert-rule-templates/${templateId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/RT-1-05-删除成功.png', fullPage: false })
    })

    // ---- 6. 验证删除生效 ----
    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/alert-rule-templates/${templateId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/RT-1-06-删除验证.png', fullPage: false })
    })

    templateId = null
  } finally {
    if (templateId) await cleanupTemplate(page, templateId)
  }
})

// ---------------------------------------------------------------------------
// RT-2 规则模板 apply 创建规则
// ---------------------------------------------------------------------------
test('RT-2 规则模板 apply 创建规则', async ({ authPage: page }) => {
  let templateId: number | null = null
  let createdRuleId: number | null = null

  try {
    // ---- 1. 创建规则模板 ----
    await test.step('创建规则模板', async () => {
      const tpl = await createTemplate(page, {
        category: 'infrastructure',
        severity: 'critical',
      })
      templateId = tpl.id
      await page.screenshot({ path: 'test-results/RT-2-01-创建模板.png', fullPage: false })
    })

    // ---- 2. Apply 模板创建规则 ----
    await test.step('Apply 模板创建规则', async () => {
      const res = await API.post(page, `${API_BASE}/alert-rule-templates/${templateId}/apply`, {
        datasource_id: 1,
        overrides: { name: `applied-rule-${uid()}` },
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      expect(res.data.id).toBeGreaterThan(0)
      createdRuleId = res.data.id
      await page.screenshot({ path: 'test-results/RT-2-02-Apply成功.png', fullPage: false })
    })

    // ---- 3. 验证创建的规则存在 ----
    await test.step('验证规则已创建', async () => {
      const res = await API.get(page, `${API_BASE}/alert-rules/${createdRuleId}`)
      expect(res.code).toBe(0)
      expect(res.data.expression).toContain('up{job="test-')
      expect(res.data.severity).toBe('critical')
      await page.screenshot({ path: 'test-results/RT-2-03-规则验证.png', fullPage: false })
    })

    // ---- 4. 验证规则字段继承自模板 ----
    await test.step('验证规则字段继承', async () => {
      const tplRes = await API.get(page, `${API_BASE}/alert-rule-templates/${templateId}`)
      const ruleRes = await API.get(page, `${API_BASE}/alert-rules/${createdRuleId}`)
      expect(tplRes.code).toBe(0)
      expect(ruleRes.code).toBe(0)
      expect(ruleRes.data.severity).toBe(tplRes.data.severity)
      expect(ruleRes.data.for_duration).toBe(tplRes.data.for_duration)
      await page.screenshot({ path: 'test-results/RT-2-04-字段继承验证.png', fullPage: false })
    })
  } finally {
    if (createdRuleId) {
      try { await API.del(page, `${API_BASE}/alert-rules/${createdRuleId}`) } catch { /* ignore */ }
    }
    if (templateId) await cleanupTemplate(page, templateId)
  }
})

// ---------------------------------------------------------------------------
// RT-3 规则模板分类列表
// ---------------------------------------------------------------------------
test('RT-3 规则模板分类列表', async ({ authPage: page }) => {
  const templateIds: number[] = []
  const uniqueCategory = `tpl-cat-${uid()}`

  try {
    // ---- 1. 创建带分类的模板 ----
    await test.step('创建带分类的模板', async () => {
      for (let i = 0; i < 2; i++) {
        const tpl = await createTemplate(page, { category: uniqueCategory })
        templateIds.push(tpl.id)
        expect(tpl.category).toBe(uniqueCategory)
      }
      await page.screenshot({ path: 'test-results/RT-3-01-创建分类模板.png', fullPage: false })
    })

    // ---- 2. 获取分类列表 ----
    await test.step('获取分类列表', async () => {
      const res = await API.get(page, `${API_BASE}/alert-rule-templates/categories`)
      expect(res.code).toBe(0)
      expect(Array.isArray(res.data)).toBe(true)
      expect(res.data).toContain(uniqueCategory)
      await page.screenshot({ path: 'test-results/RT-3-02-分类列表.png', fullPage: false })
    })

    // ---- 3. 按分类筛选验证结果 ----
    await test.step('按分类筛选验证', async () => {
      const res = await API.get(page, `${API_BASE}/alert-rule-templates?category=${uniqueCategory}&page_size=100`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      expect(list.length).toBeGreaterThanOrEqual(2)
      for (const t of list) {
        expect(t.category).toBe(uniqueCategory)
      }
      await page.screenshot({ path: 'test-results/RT-3-03-筛选结果.png', fullPage: false })
    })
  } finally {
    for (const id of templateIds) await cleanupTemplate(page, id)
  }
})

// ---------------------------------------------------------------------------
// RT-4 规则模板批量应用
// ---------------------------------------------------------------------------
test('RT-4 规则模板批量应用', async ({ authPage: page }) => {
  const templateIds: number[] = []
  const createdRuleIds: number[] = []

  try {
    // ---- 1. 创建 3 个规则模板 ----
    await test.step('创建 3 个规则模板', async () => {
      for (let i = 0; i < 3; i++) {
        const tpl = await createTemplate(page, { severity: 'warning' })
        templateIds.push(tpl.id)
      }
      expect(templateIds.length).toBe(3)
      await page.screenshot({ path: 'test-results/RT-4-01-创建3模板.png', fullPage: false })
    })

    // ---- 2. 批量应用模板 ----
    await test.step('批量应用模板', async () => {
      const res = await API.post(page, `${API_BASE}/alert-rule-templates/batch-apply`, {
        template_ids: templateIds,
        datasource_id: 1,
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      if (res.data.created_ids) {
        createdRuleIds.push(...res.data.created_ids)
      }
      await page.screenshot({ path: 'test-results/RT-4-02-批量应用.png', fullPage: false })
    })

    // ---- 3. 验证创建的规则数量 ----
    await test.step('验证创建的规则数量', async () => {
      if (createdRuleIds.length > 0) {
        expect(createdRuleIds.length).toBeGreaterThanOrEqual(3)
        for (const ruleId of createdRuleIds) {
          const res = await API.get(page, `${API_BASE}/alert-rules/${ruleId}`)
          expect(res.code).toBe(0)
          expect(res.data.severity).toBe('warning')
        }
      }
      await page.screenshot({ path: 'test-results/RT-4-03-验证规则数量.png', fullPage: false })
    })
  } finally {
    for (const id of createdRuleIds) {
      try { await API.del(page, `${API_BASE}/alert-rules/${id}`) } catch { /* ignore */ }
    }
    for (const id of templateIds) await cleanupTemplate(page, id)
  }
})
