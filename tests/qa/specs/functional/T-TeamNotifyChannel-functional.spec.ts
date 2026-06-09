import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: get the first available team ID from the current user's teams */
async function getTeamId(page: any): Promise<number> {
  const res = await API.get(page, `${API_BASE}/me/teams`)
  if (res.code === 0 && Array.isArray(res.data) && res.data.length > 0) {
    return res.data[0].id ?? res.data[0]
  }
  // Fallback: try teams list
  const res2 = await API.get(page, `${API_BASE}/teams?page=1&page_size=10`)
  if (res2.code === 0) {
    const list = res2.data?.list || res2.data || []
    if (list.length > 0) return list[0].id
  }
  return 1 // default fallback
}

/** Helper: create a notify media for testing */
async function createNotifyMedia(page: any): Promise<number> {
  const tag = uid()
  const res = await API.post(page, `${API_BASE}/notify-media`, {
    name: `media-${tag}`,
    type: 'webhook',
    webhook_url: `https://example.com/hook-${tag}`,
  })
  if (res.code === 0 && res.data?.id) return res.data.id
  return 0
}

/** Helper: delete a notify media by ID */
async function cleanupNotifyMedia(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/notify-media/${id}`)
  } catch { /* ignore */ }
}

/** Helper: create a team notify channel and return the created object */
async function createChannel(page: any, teamId: number, mediaId: number, overrides: Record<string, unknown> = {}) {
  const payload = {
    team_id: teamId,
    media_id: mediaId,
    is_default: false,
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/team-notify-channels`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return res.data
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
  let mediaId: number | null = null
  let teamId: number

  try {
    await test.step('获取团队 ID', async () => {
      teamId = await getTeamId(page)
    })

    await test.step('创建通知媒体', async () => {
      mediaId = await createNotifyMedia(page)
    })

    // ---- 1. 创建通知渠道 ----
    await test.step('创建通知渠道', async () => {
      const channel = await createChannel(page, teamId, mediaId!)
      channelId = channel.id
      expect(channel.team_id).toBe(teamId)
      expect(channel.media_id).toBe(mediaId)
      await page.screenshot({ path: 'test-results/TNC-1-01-创建成功.png', fullPage: false })
    })

    // ---- 2. 列表验证 ----
    await test.step('列表验证渠道已保存', async () => {
      const res = await API.get(page, `${API_BASE}/team-notify-channels/${teamId}`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      const found = list.find((c: any) => c.id === channelId)
      expect(found).toBeTruthy()
      expect(found.team_id).toBe(teamId)
      await page.screenshot({ path: 'test-results/TNC-1-02-列表验证.png', fullPage: false })
    })

    // ---- 3. 更新渠道 ----
    await test.step('更新渠道', async () => {
      const res = await API.put(page, `${API_BASE}/team-notify-channels/${channelId}`, {
        team_id: teamId,
        media_id: mediaId,
        is_default: true,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/TNC-1-03-更新成功.png', fullPage: false })
    })

    // ---- 4. 验证更新 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/team-notify-channels/${teamId}`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      const found = list.find((c: any) => c.id === channelId)
      expect(found).toBeTruthy()
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
      const res = await API.get(page, `${API_BASE}/team-notify-channels/${teamId}`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      const found = list.find((c: any) => c.id === channelId)
      expect(found).toBeFalsy()
      await page.screenshot({ path: 'test-results/TNC-1-06-删除验证.png', fullPage: false })
    })

    channelId = null
  } finally {
    if (channelId) await cleanupChannel(page, channelId)
    if (mediaId) await cleanupNotifyMedia(page, mediaId)
  }
})

// ---------------------------------------------------------------------------
// TNC-2 团队通知渠道默认设置
// ---------------------------------------------------------------------------
test('TNC-2 团队通知渠道默认设置', async ({ authPage: page }) => {
  let channelId: number | null = null
  let mediaId: number | null = null
  let teamId: number

  try {
    await test.step('获取团队 ID', async () => {
      teamId = await getTeamId(page)
    })

    await test.step('创建通知媒体', async () => {
      mediaId = await createNotifyMedia(page)
    })

    // ---- 1. 创建渠道 ----
    await test.step('创建渠道', async () => {
      const channel = await createChannel(page, teamId, mediaId!)
      channelId = channel.id
      await page.screenshot({ path: 'test-results/TNC-2-01-创建渠道.png', fullPage: false })
    })

    // ---- 2. 设置为默认渠道 ----
    await test.step('设置为默认渠道', async () => {
      const res = await API.post(page, `${API_BASE}/team-notify-channels/${channelId}/default`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/TNC-2-02-设置默认.png', fullPage: false })
    })

    // ---- 3. 验证默认设置 ----
    await test.step('验证默认设置', async () => {
      const res = await API.get(page, `${API_BASE}/team-notify-channels/${teamId}`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      const found = list.find((c: any) => c.id === channelId)
      expect(found).toBeTruthy()
      expect(found.is_default).toBe(true)
      await page.screenshot({ path: 'test-results/TNC-2-03-默认验证.png', fullPage: false })
    })
  } finally {
    if (channelId) await cleanupChannel(page, channelId)
    if (mediaId) await cleanupNotifyMedia(page, mediaId)
  }
})

// ---------------------------------------------------------------------------
// TNC-3 团队通知渠道列表
// ---------------------------------------------------------------------------
test('TNC-3 团队通知渠道列表', async ({ authPage: page }) => {
  let teamId: number

  try {
    await test.step('获取团队 ID', async () => {
      teamId = await getTeamId(page)
    })

    await test.step('获取团队通知渠道列表', async () => {
      const res = await API.get(page, `${API_BASE}/team-notify-channels/${teamId}`)
      expect(res.code).toBe(0)
      expect(Array.isArray(res.data)).toBe(true)
      await page.screenshot({ path: 'test-results/TNC-3-01-渠道列表.png', fullPage: false })
    })

    await test.step('验证列表结构', async () => {
      const res = await API.get(page, `${API_BASE}/team-notify-channels/${teamId}`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      // Each item should have id, team_id, media_id
      for (const ch of list) {
        expect(ch.id).toBeDefined()
        expect(ch.team_id).toBeDefined()
      }
      await page.screenshot({ path: 'test-results/TNC-3-02-列表结构.png', fullPage: false })
    })
  } catch (e) {
    await page.screenshot({ path: 'test-results/TNC-3-ERROR.png', fullPage: false })
    throw e
  }
})
