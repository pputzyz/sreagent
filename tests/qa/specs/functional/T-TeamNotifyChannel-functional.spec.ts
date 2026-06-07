import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a team notify channel and return the created object */
async function createChannel(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `channel-${tag}`,
    type: 'webhook',
    webhook_url: `https://example.com/hook-${tag}`,
    enabled: true,
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/team-notify-channels`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a channel by ID, ignoring errors (for cleanup) */
async function cleanupChannel(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/team-notify-channels/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// TNC-1 团队通知渠道 CRUD
// ---------------------------------------------------------------------------
test('TNC-1 团队通知渠道CRUD', async ({ authPage: page }) => {
  let channelId: number | null = null

  try {
    // ---- 1. 创建通知渠道 ----
    await test.step('创建通知渠道', async () => {
      const channel = await createChannel(page, {
        description: 'CRUD 测试渠道',
      })
      channelId = channel.id
      expect(channel.name).toContain('channel-')
      expect(channel.type).toBe('webhook')
      expect(channel.enabled).toBe(true)
      await page.screenshot({ path: 'test-results/TNC-1-01-创建成功.png', fullPage: false })
    })

    // ---- 2. GET 验证 ----
    await test.step('GET 验证渠道已保存', async () => {
      const res = await API.get(page, `${API_BASE}/team-notify-channels/${channelId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(channelId)
      expect(res.data.name).toContain('channel-')
      expect(res.data.type).toBe('webhook')
      await page.screenshot({ path: 'test-results/TNC-1-02-GET验证.png', fullPage: false })
    })

    // ---- 3. 更新渠道 ----
    await test.step('更新渠道名称', async () => {
      const res = await API.put(page, `${API_BASE}/team-notify-channels/${channelId}`, {
        name: `updated-channel-${uid()}`,
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/TNC-1-03-更新成功.png', fullPage: false })
    })

    // ---- 4. 验证更新 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/team-notify-channels/${channelId}`)
      expect(res.code).toBe(0)
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/TNC-1-04-更新验证.png', fullPage: false })
    })

    // ---- 5. 删除渠道 ----
    await test.step('删除渠道', async () => {
      const res = await API.del(page, `${API_BASE}/team-notify-channels/${channelId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/TNC-1-05-删除成功.png', fullPage: false })
    })

    // ---- 6. 验证删除 ----
    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/team-notify-channels/${channelId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/TNC-1-06-删除验证.png', fullPage: false })
    })

    channelId = null
  } finally {
    if (channelId) await cleanupChannel(page, channelId)
  }
})

// ---------------------------------------------------------------------------
// TNC-2 团队通知渠道默认设置
// ---------------------------------------------------------------------------
test('TNC-2 团队通知渠道默认设置', async ({ authPage: page }) => {
  let channelId: number | null = null

  try {
    // ---- 1. 创建渠道 ----
    await test.step('创建渠道', async () => {
      const channel = await createChannel(page)
      channelId = channel.id
      await page.screenshot({ path: 'test-results/TNC-2-01-创建渠道.png', fullPage: false })
    })

    // ---- 2. 设置为默认渠道 ----
    await test.step('设置为默认渠道', async () => {
      const res = await API.put(page, `${API_BASE}/team-notify-channels/${channelId}`, {
        is_default: true,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/TNC-2-02-设置默认.png', fullPage: false })
    })

    // ---- 3. 验证默认设置 ----
    await test.step('验证默认设置', async () => {
      const res = await API.get(page, `${API_BASE}/team-notify-channels/${channelId}`)
      expect(res.code).toBe(0)
      expect(res.data.is_default).toBe(true)
      await page.screenshot({ path: 'test-results/TNC-2-03-默认验证.png', fullPage: false })
    })

    // ---- 4. 取消默认 ----
    await test.step('取消默认', async () => {
      const res = await API.put(page, `${API_BASE}/team-notify-channels/${channelId}`, {
        is_default: false,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/TNC-2-04-取消默认.png', fullPage: false })
    })

    // ---- 5. 验证取消默认 ----
    await test.step('验证取消默认', async () => {
      const res = await API.get(page, `${API_BASE}/team-notify-channels/${channelId}`)
      expect(res.code).toBe(0)
      expect(res.data.is_default).toBe(false)
      await page.screenshot({ path: 'test-results/TNC-2-05-取消验证.png', fullPage: false })
    })
  } finally {
    if (channelId) await cleanupChannel(page, channelId)
  }
})

// ---------------------------------------------------------------------------
// TNC-3 团队通知渠道测试发送
// ---------------------------------------------------------------------------
test('TNC-3 团队通知渠道测试发送', async ({ authPage: page }) => {
  let channelId: number | null = null

  try {
    // ---- 1. 创建渠道 ----
    await test.step('创建渠道', async () => {
      const channel = await createChannel(page, {
        type: 'webhook',
        webhook_url: 'https://httpbin.org/post',
      })
      channelId = channel.id
      await page.screenshot({ path: 'test-results/TNC-3-01-创建渠道.png', fullPage: false })
    })

    // ---- 2. 发送测试通知 ----
    await test.step('发送测试通知', async () => {
      const res = await API.post(page, `${API_BASE}/team-notify-channels/${channelId}/test`)
      // 测试发送可能成功或失败（取决于外部服务），但 API 应正常响应
      expect(res).toBeDefined()
      expect(res).toHaveProperty('code')
      await page.screenshot({ path: 'test-results/TNC-3-02-测试发送.png', fullPage: false })
    })
  } finally {
    if (channelId) await cleanupChannel(page, channelId)
  }
})

// ---------------------------------------------------------------------------
// TNC-4 团队通知渠道启用禁用
// ---------------------------------------------------------------------------
test('TNC-4 团队通知渠道启用禁用', async ({ authPage: page }) => {
  let channelId: number | null = null

  try {
    // ---- 1. 创建启用的渠道 ----
    await test.step('创建启用的渠道', async () => {
      const channel = await createChannel(page, { enabled: true })
      channelId = channel.id
      expect(channel.enabled).toBe(true)
      await page.screenshot({ path: 'test-results/TNC-4-01-创建渠道.png', fullPage: false })
    })

    // ---- 2. 禁用渠道 ----
    await test.step('禁用渠道', async () => {
      const res = await API.put(page, `${API_BASE}/team-notify-channels/${channelId}`, {
        enabled: false,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/TNC-4-02-禁用渠道.png', fullPage: false })
    })

    // ---- 3. 验证已禁用 ----
    await test.step('验证已禁用', async () => {
      const res = await API.get(page, `${API_BASE}/team-notify-channels/${channelId}`)
      expect(res.code).toBe(0)
      expect(res.data.enabled).toBe(false)
      await page.screenshot({ path: 'test-results/TNC-4-03-禁用验证.png', fullPage: false })
    })

    // ---- 4. 重新启用 ----
    await test.step('重新启用', async () => {
      const res = await API.put(page, `${API_BASE}/team-notify-channels/${channelId}`, {
        enabled: true,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/TNC-4-04-重新启用.png', fullPage: false })
    })

    // ---- 5. 验证已启用 ----
    await test.step('验证已启用', async () => {
      const res = await API.get(page, `${API_BASE}/team-notify-channels/${channelId}`)
      expect(res.code).toBe(0)
      expect(res.data.enabled).toBe(true)
      await page.screenshot({ path: 'test-results/TNC-4-05-启用验证.png', fullPage: false })
    })
  } finally {
    if (channelId) await cleanupChannel(page, channelId)
  }
})
