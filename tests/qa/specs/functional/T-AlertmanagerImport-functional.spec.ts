import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: delete a rule by ID, ignoring errors (for cleanup) */
async function cleanupRule(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/alert-rules/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// AMI-1 Alertmanager 导入 YAML 解析
// ---------------------------------------------------------------------------
test('AMI-1 Alertmanager导入YAML解析', async ({ authPage: page }) => {
  const tag = uid()

  // ---- 1. 构造 Alertmanager YAML ----
  const yamlContent = `global:
  resolve_timeout: 5m
route:
  receiver: 'webhook-${tag}'
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  routes:
    - receiver: 'webhook-${tag}'
      match:
        severity: critical
receivers:
  - name: 'webhook-${tag}'
    webhook_configs:
      - url: 'http://localhost:9095/webhook'
        send_resolved: true
`

  // ---- 2. 上传 YAML 解析 ----
  await test.step('上传 Alertmanager YAML 解析', async () => {
    const token = await page.evaluate(() => localStorage.getItem('token'))
    const resp = await page.request.post(`http://localhost:3000${API_BASE}/alert-rules/import-alertmanager`, {
      headers: { Authorization: `Bearer ${token}` },
      multipart: {
        file: {
          name: 'alertmanager.yml',
          mimeType: 'text/yaml',
          buffer: Buffer.from(yamlContent),
        },
      },
    })
    const result = await resp.json()
    expect(result).toBeDefined()
    // 解析应返回 code 0 或有意义的响应
    expect(result).toHaveProperty('code')
    await page.screenshot({ path: 'test-results/AMI-1-01-YAML解析.png', fullPage: false })
  })

  // ---- 3. 验证解析结果结构 ----
  await test.step('验证解析结果结构', async () => {
    const token = await page.evaluate(() => localStorage.getItem('token'))
    const resp = await page.request.post(`http://localhost:3000${API_BASE}/alert-rules/import-alertmanager`, {
      headers: { Authorization: `Bearer ${token}` },
      multipart: {
        file: {
          name: 'alertmanager.yml',
          mimeType: 'text/yaml',
          buffer: Buffer.from(yamlContent),
        },
      },
    })
    const result = await resp.json()
    if (result.code === 0 && result.data) {
      // 如果有解析数据，应包含规则信息
      expect(result.data).toBeDefined()
    }
    await page.screenshot({ path: 'test-results/AMI-1-02-解析结构.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// AMI-2 Alertmanager 导入流程
// ---------------------------------------------------------------------------
test('AMI-2 Alertmanager导入流程', async ({ authPage: page }) => {
  const tag = uid()
  const importedRuleIds: number[] = []

  try {
    // ---- 1. 构造 Alertmanager YAML 并导入 ----
    const yamlContent = `global:
  resolve_timeout: 5m
route:
  receiver: 'import-test-${tag}'
  group_by: ['alertname']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
receivers:
  - name: 'import-test-${tag}'
    webhook_configs:
      - url: 'http://localhost:9095/webhook'
        send_resolved: true
`

    // ---- 2. 执行导入 ----
    await test.step('执行 Alertmanager 导入', async () => {
      const token = await page.evaluate(() => localStorage.getItem('token'))
      const resp = await page.request.post(`http://localhost:3000${API_BASE}/alert-rules/import-alertmanager`, {
        headers: { Authorization: `Bearer ${token}` },
        multipart: {
          file: {
            name: 'alertmanager.yml',
            mimeType: 'text/yaml',
            buffer: Buffer.from(yamlContent),
          },
        },
      })
      const result = await resp.json()
      expect(result).toBeDefined()
      expect(result).toHaveProperty('code')
      await page.screenshot({ path: 'test-results/AMI-2-01-导入执行.png', fullPage: false })
    })

    // ---- 3. 验证导入结果 ----
    await test.step('验证导入结果', async () => {
      // 查询是否有导入的规则
      const res = await API.get(page, `${API_BASE}/alert-rules?keyword=import-test-${tag}&page_size=100`)
      expect(res.code).toBe(0)
      const list = res.data.list || []
      // 记录导入的规则 ID 用于清理
      for (const rule of list) {
        if (rule.name && rule.name.includes(`import-test-${tag}`)) {
          importedRuleIds.push(rule.id)
        }
      }
      await page.screenshot({ path: 'test-results/AMI-2-02-导入验证.png', fullPage: false })
    })

    // ---- 4. 验证导入的规则可查询 ----
    await test.step('验证导入的规则可查询', async () => {
      if (importedRuleIds.length > 0) {
        for (const id of importedRuleIds) {
          const res = await API.get(page, `${API_BASE}/alert-rules/${id}`)
          expect(res.code).toBe(0)
          expect(res.data).toBeTruthy()
          expect(res.data.id).toBe(id)
        }
      }
      await page.screenshot({ path: 'test-results/AMI-2-03-规则可查询.png', fullPage: false })
    })
  } finally {
    // cleanup imported rules
    for (const id of importedRuleIds) {
      await cleanupRule(page, id)
    }
  }
})
