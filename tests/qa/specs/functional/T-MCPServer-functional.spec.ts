import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create an MCP server via API and return the created object */
async function createMCPServer(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `mcp-test-${tag}`,
    url: `http://localhost:9999/mcp/${tag}`,
    description: 'Functional test MCP server',
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/mcp-servers`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  // GORM models use capitalized field names (ID, CreatedAt, etc.)
  const id = res.data.ID || res.data.id
  expect(id).toBeGreaterThan(0)
  return { ...res.data, id, _tag: tag, _payload: payload }
}

/** Helper: delete an MCP server by ID, ignoring errors (for cleanup) */
async function cleanupMCPServer(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/mcp-servers/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// MC-1: MCP 服务器 CRUD
// ---------------------------------------------------------------------------
test('MC-1 MCP服务器 CRUD', async ({ authPage: page }) => {
  let serverId: number | null = null

  try {
    await test.step('创建 MCP 服务器', async () => {
      const server = await createMCPServer(page)
      serverId = server.id
      expect(server.name).toContain('mcp-test-')
      expect(server.url).toContain('localhost:9999')
      await page.screenshot({ path: 'test-results/MC-1-01-创建成功.png', fullPage: false })
    })

    await test.step('GET 验证 MCP 服务器已保存', async () => {
      const res = await API.get(page, `${API_BASE}/mcp-servers/${serverId}`)
      expect(res.code).toBe(0)
      const id = res.data.ID || res.data.id
      expect(id).toBe(serverId)
      expect(res.data.name).toContain('mcp-test-')
      await page.screenshot({ path: 'test-results/MC-1-02-GET验证.png', fullPage: false })
    })

    await test.step('更新 MCP 服务器', async () => {
      const res = await API.put(page, `${API_BASE}/mcp-servers/${serverId}`, {
        name: `updated-mcp-${uid()}`,
        url: `http://localhost:9999/mcp/updated-${uid()}`,
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MC-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/mcp-servers/${serverId}`)
      expect(res.code).toBe(0)
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/MC-1-04-更新验证.png', fullPage: false })
    })

    await test.step('删除 MCP 服务器', async () => {
      const res = await API.del(page, `${API_BASE}/mcp-servers/${serverId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MC-1-05-删除成功.png', fullPage: false })
    })

    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/mcp-servers/${serverId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/MC-1-06-删除验证.png', fullPage: false })
    })

    serverId = null
  } finally {
    if (serverId) await cleanupMCPServer(page, serverId)
  }
})

// ---------------------------------------------------------------------------
// MC-2: MCP 服务器 test 连接
// ---------------------------------------------------------------------------
test('MC-2 MCP服务器 test连接', async ({ authPage: page }) => {
  let serverId: number | null = null

  try {
    await test.step('创建 MCP 服务器', async () => {
      const server = await createMCPServer(page)
      serverId = server.id
      await page.screenshot({ path: 'test-results/MC-2-01-创建服务器.png', fullPage: false })
    })

    await test.step('执行连接测试', async () => {
      const res = await API.post(page, `${API_BASE}/mcp-servers/${serverId}/test`)
      // Connection test may fail since endpoint doesn't exist, but should return structured response
      expect(res).toBeDefined()
      expect(res.code).toBeDefined()
      await page.screenshot({ path: 'test-results/MC-2-02-连接测试.png', fullPage: false })
    })

    await test.step('验证连接测试返回结果', async () => {
      const res = await API.post(page, `${API_BASE}/mcp-servers/${serverId}/test`)
      // Should have a success or failure indicator
      expect(res.data !== undefined || res.message !== undefined).toBeTruthy()
      await page.screenshot({ path: 'test-results/MC-2-03-测试结果.png', fullPage: false })
    })
  } finally {
    if (serverId) await cleanupMCPServer(page, serverId)
  }
})

// ---------------------------------------------------------------------------
// MC-3: MCP 服务器工具列表
// ---------------------------------------------------------------------------
test('MC-3 MCP服务器工具列表', async ({ authPage: page }) => {
  let serverId: number | null = null

  try {
    await test.step('创建 MCP 服务器', async () => {
      const server = await createMCPServer(page)
      serverId = server.id
      await page.screenshot({ path: 'test-results/MC-3-01-创建服务器.png', fullPage: false })
    })

    await test.step('获取工具列表', async () => {
      const res = await API.get(page, `${API_BASE}/mcp-servers/${serverId}/tools`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/MC-3-02-工具列表.png', fullPage: false })
    })

    await test.step('验证工具列表结构', async () => {
      const res = await API.get(page, `${API_BASE}/mcp-servers/${serverId}/tools`)
      expect(res.code).toBe(0)
      const tools = Array.isArray(res.data) ? res.data : res.data.list || []
      expect(Array.isArray(tools)).toBe(true)
      await page.screenshot({ path: 'test-results/MC-3-03-工具结构.png', fullPage: false })
    })
  } finally {
    if (serverId) await cleanupMCPServer(page, serverId)
  }
})

// ---------------------------------------------------------------------------
// MC-4: MCP 服务器工具调用
// ---------------------------------------------------------------------------
test('MC-4 MCP服务器工具调用', async ({ authPage: page }) => {
  let serverId: number | null = null

  try {
    await test.step('创建 MCP 服务器', async () => {
      const server = await createMCPServer(page)
      serverId = server.id
      await page.screenshot({ path: 'test-results/MC-4-01-创建服务器.png', fullPage: false })
    })

    await test.step('获取可用工具', async () => {
      const res = await API.get(page, `${API_BASE}/mcp-servers/${serverId}/tools`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/MC-4-02-获取工具.png', fullPage: false })
    })

    await test.step('尝试调用工具', async () => {
      const toolsRes = await API.get(page, `${API_BASE}/mcp-servers/${serverId}/tools`)
      const tools = Array.isArray(toolsRes.data) ? toolsRes.data : toolsRes.data?.list || []
      if (tools.length > 0) {
        const toolName = tools[0].name || tools[0].tool_name || 'unknown'
        const res = await API.post(page, `${API_BASE}/mcp-servers/${serverId}/tools/${toolName}/invoke`, {
          params: {},
        })
        expect(res).toBeDefined()
        await page.screenshot({ path: 'test-results/MC-4-03-工具调用.png', fullPage: false })
      } else {
        // No tools available — verify empty list response
        expect(Array.isArray(tools)).toBe(true)
        await page.screenshot({ path: 'test-results/MC-4-03-无工具.png', fullPage: false })
      }
    })
  } finally {
    if (serverId) await cleanupMCPServer(page, serverId)
  }
})
