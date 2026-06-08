import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

// ---------------------------------------------------------------------------
// PR-1 预设规则列表分类
// ---------------------------------------------------------------------------
test('PR-1 预设规则列表分类', async ({ authPage: page }) => {
  try {
    // ---- 1. 获取预设规则列表 ----
    await test.step('获取预设规则列表', async () => {
      const res = await API.get(page, `${API_BASE}/preset-rules?page_size=100`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/PR-1-01-预设规则列表.png', fullPage: false })
    })

    // ---- 2. 获取预设规则分类列表 ----
    await test.step('获取预设规则分类列表', async () => {
      const res = await API.get(page, `${API_BASE}/preset-rules/categories`)
      expect(res.code).toBe(0)
      expect(Array.isArray(res.data)).toBe(true)
      await page.screenshot({ path: 'test-results/PR-1-02-分类列表.png', fullPage: false })
    })

    // ---- 3. 按分类筛选预设规则 ----
    await test.step('按分类筛选预设规则', async () => {
      const catRes = await API.get(page, `${API_BASE}/preset-rules/categories`)
      expect(catRes.code).toBe(0)
      if (catRes.data.length > 0) {
        const category = catRes.data[0]
        const res = await API.get(page, `${API_BASE}/preset-rules?category=${encodeURIComponent(category)}&page_size=100`)
        expect(res.code).toBe(0)
        const list = res.data.list || []
        for (const r of list) {
          expect(r.category).toBe(category)
        }
        await page.screenshot({ path: 'test-results/PR-1-03-分类筛选结果.png', fullPage: false })
      } else {
        await page.screenshot({ path: 'test-results/PR-1-03-无分类数据.png', fullPage: false })
      }
    })

    // ---- 4. 按关键词搜索预设规则 ----
    await test.step('按关键词搜索预设规则', async () => {
      const res = await API.get(page, `${API_BASE}/preset-rules?keyword=cpu&page_size=10`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/PR-1-04-关键词搜索.png', fullPage: false })
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/PR-1-ERROR.png', fullPage: false })
    throw e
  }
})

// ---------------------------------------------------------------------------
// PR-2 预设规则详情
// ---------------------------------------------------------------------------
test('PR-2 预设规则详情', async ({ authPage: page }) => {
  let presetId: number | null = null

  try {
    // ---- 1. 获取预设规则列表找到一个 ID ----
    await test.step('获取预设规则列表', async () => {
      const res = await API.get(page, `${API_BASE}/preset-rules?page_size=10`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      expect(list.length).toBeGreaterThan(0)
      presetId = list[0].id
      await page.screenshot({ path: 'test-results/PR-2-01-预设规则列表.png', fullPage: false })
    })

    // ---- 2. 获取预设规则详情 ----
    await test.step('获取预设规则详情', async () => {
      const res = await API.get(page, `${API_BASE}/preset-rules/${presetId}`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      expect(res.data.id).toBe(presetId)
      expect(res.data.name).toBeTruthy()
      expect(res.data.expression).toBeTruthy()
      await page.screenshot({ path: 'test-results/PR-2-02-预设规则详情.png', fullPage: false })
    })

    // ---- 3. 验证详情包含完整字段 ----
    await test.step('验证详情包含完整字段', async () => {
      const res = await API.get(page, `${API_BASE}/preset-rules/${presetId}`)
      expect(res.code).toBe(0)
      const r = res.data
      expect(r.name).toBeTruthy()
      expect(r.expression).toBeTruthy()
      expect(r.severity).toBeTruthy()
      expect(r.category).toBeTruthy()
      await page.screenshot({ path: 'test-results/PR-2-03-完整字段验证.png', fullPage: false })
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/PR-2-ERROR.png', fullPage: false })
    throw e
  }
})

// ---------------------------------------------------------------------------
// PR-3 预设规则 apply 应用
// ---------------------------------------------------------------------------
test('PR-3 预设规则 apply 应用', async ({ authPage: page }) => {
  let presetId: number | null = null
  let createdRuleId: number | null = null

  try {
    // ---- 1. 获取预设规则列表找到一个 ID ----
    await test.step('获取预设规则列表', async () => {
      const res = await API.get(page, `${API_BASE}/preset-rules?page_size=10`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      expect(list.length).toBeGreaterThan(0)
      presetId = list[0].id
      await page.screenshot({ path: 'test-results/PR-3-01-预设规则列表.png', fullPage: false })
    })

    // ---- 2. Apply 预设规则创建告警规则 ----
    await test.step('Apply 预设规则创建告警规则', async () => {
      const res = await API.post(page, `${API_BASE}/preset-rules/${presetId}/apply`, {
        datasource_id: 1,
        overrides: { name: `preset-applied-${uid()}` },
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      expect(res.data.id).toBeGreaterThan(0)
      createdRuleId = res.data.id
      await page.screenshot({ path: 'test-results/PR-3-02-Apply成功.png', fullPage: false })
    })

    // ---- 3. 验证创建的规则存在 ----
    await test.step('验证规则已创建', async () => {
      const res = await API.get(page, `${API_BASE}/alert-rules/${createdRuleId}`)
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      expect(res.data.id).toBe(createdRuleId)
      await page.screenshot({ path: 'test-results/PR-3-03-规则验证.png', fullPage: false })
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/PR-3-ERROR.png', fullPage: false })
    throw e
  } finally {
    if (createdRuleId) {
      try { await API.del(page, `${API_BASE}/alert-rules/${createdRuleId}`) } catch { /* ignore */ }
    }
  }
})

// ---------------------------------------------------------------------------
// PR-4 预设规则 batch-apply
// ---------------------------------------------------------------------------
test('PR-4 预设规则 batch-apply', async ({ authPage: page }) => {
  const presetIds: number[] = []
  const createdRuleIds: number[] = []

  try {
    // ---- 1. 获取预设规则列表找到多个 ID ----
    await test.step('获取预设规则列表', async () => {
      const res = await API.get(page, `${API_BASE}/preset-rules?page_size=10`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      expect(list.length).toBeGreaterThanOrEqual(2)
      presetIds.push(list[0].id, list[1].id)
      await page.screenshot({ path: 'test-results/PR-4-01-预设规则列表.png', fullPage: false })
    })

    // ---- 2. 批量 Apply 预设规则 ----
    await test.step('批量 Apply 预设规则', async () => {
      const res = await API.post(page, `${API_BASE}/preset-rules/batch-apply`, {
        preset_ids: presetIds,
        datasource_id: 1,
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      if (res.data.created_ids) {
        createdRuleIds.push(...res.data.created_ids)
      }
      await page.screenshot({ path: 'test-results/PR-4-02-批量Apply.png', fullPage: false })
    })

    // ---- 3. 验证创建的规则数量 ----
    await test.step('验证创建的规则数量', async () => {
      if (createdRuleIds.length > 0) {
        expect(createdRuleIds.length).toBeGreaterThanOrEqual(2)
        for (const ruleId of createdRuleIds) {
          const res = await API.get(page, `${API_BASE}/alert-rules/${ruleId}`)
          expect(res.code).toBe(0)
        }
      }
      await page.screenshot({ path: 'test-results/PR-4-03-验证规则数量.png', fullPage: false })
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/PR-4-ERROR.png', fullPage: false })
    throw e
  } finally {
    for (const id of createdRuleIds) {
      try { await API.del(page, `${API_BASE}/alert-rules/${id}`) } catch { /* ignore */ }
    }
  }
})

// ---------------------------------------------------------------------------
// PR-5 预设规则 YAML 导入
// ---------------------------------------------------------------------------
test('PR-5 预设规则 YAML 导入', async ({ authPage: page }) => {
  const createdRuleIds: number[] = []

  try {
    // ---- 1. 准备 YAML 内容 ----
    await test.step('准备 YAML 内容', async () => {
      await page.screenshot({ path: 'test-results/PR-5-01-准备YAML.png', fullPage: false })
    })

    // ---- 2. 通过 YAML 导入预设规则 ----
    await test.step('YAML 导入预设规则', async () => {
      const tag = uid()
      const yamlContent = `groups:
  - name: preset-import-group-${tag}
    rules:
      - alert: preset-import-${tag}
        expr: up{job="preset-test-${tag}"} == 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: Preset import test
`
      const token = await page.evaluate(() => localStorage.getItem('token'))
      const resp = await page.request.post(`http://localhost:3000${API_BASE}/preset-rules/import`, {
        headers: { Authorization: `Bearer ${token}` },
        multipart: {
          file: {
            name: 'preset_import.yaml',
            mimeType: 'text/yaml',
            buffer: Buffer.from(yamlContent),
          },
        },
      })
      const importResult = await resp.json()
      expect(importResult.code).toBe(0)
      expect(importResult.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/PR-5-02-导入成功.png', fullPage: false })
    })

    // ---- 3. 验证导入的预设规则存在 ----
    await test.step('验证导入的预设规则', async () => {
      const res = await API.get(page, `${API_BASE}/preset-rules?keyword=preset-import&page_size=10`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      expect(list.length).toBeGreaterThanOrEqual(1)
      await page.screenshot({ path: 'test-results/PR-5-03-导入验证.png', fullPage: false })
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/PR-5-ERROR.png', fullPage: false })
    throw e
  } finally {
    for (const id of createdRuleIds) {
      try { await API.del(page, `${API_BASE}/alert-rules/${id}`) } catch { /* ignore */ }
    }
  }
})
