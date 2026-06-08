import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a channel via API and return its ID */
async function createChannel(page: any, name?: string): Promise<number> {
  const tag = uid()
  const res = await API.post(page, `${API_BASE}/channels`, {
    name: name || `test-ch-${tag}`,
    description: 'Functional test channel',
    access_level: 'public',
  })
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return res.data.id
}

/** Helper: create an incident via API and return the created object */
async function createIncident(page: any, channelId: number, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    title: `func-test-incident-${tag}`,
    description: `Functional test incident ${tag}`,
    severity: 'warning',
    channel_id: channelId,
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/incidents`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a channel by ID, ignoring errors (for cleanup) */
async function cleanupChannel(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/channels/${id}`)
  } catch { /* ignore */ }
}

/** Helper: close an incident by ID, ignoring errors (for cleanup) */
async function closeIncident(page: any, id: number) {
  try {
    await API.post(page, `${API_BASE}/incidents/${id}/close`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// INC-1: Create incident via API -> verify all fields -> verify timeline -> close -> verify status
// ---------------------------------------------------------------------------
test('INC-1 创建故障并验证字段和时间线', async ({ authPage: page }) => {
  let channelId: number | null = null
  let incidentId: number | null = null

  try {
    // ---- 1. Create a channel (required for incident) ----
    await test.step('创建协作空间', async () => {
      channelId = await createChannel(page)
      await page.screenshot({ path: 'test-results/INC-1-01-创建空间.png', fullPage: false })
    })

    // ---- 2. Create incident via API ----
    await test.step('创建故障', async () => {
      const inc = await createIncident(page, channelId!, {
        severity: 'critical',
        description: 'INC-1 full field verification test',
      })
      incidentId = inc.id

      // Verify returned fields
      expect(inc.title).toContain('func-test-incident-')
      expect(inc.severity).toBe('critical')
      expect(inc.status).toBe('triggered')
      expect(inc.channel_id).toBe(channelId)
      expect(inc.description).toBe('INC-1 full field verification test')
      expect(inc.triggered_at).toBeTruthy()
      await page.screenshot({ path: 'test-results/INC-1-02-创建成功.png', fullPage: false })
    })

    // ---- 3. GET to verify all fields persisted ----
    await test.step('GET 验证所有字段已保存', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}`)
      expect(res.code).toBe(0)
      const inc = res.data
      expect(inc.id).toBe(incidentId)
      expect(inc.title).toContain('func-test-incident-')
      expect(inc.severity).toBe('critical')
      expect(inc.status).toBe('triggered')
      expect(inc.channel_id).toBe(channelId)
      expect(inc.description).toBe('INC-1 full field verification test')
      expect(inc.triggered_at).toBeTruthy()
      expect(inc.created_at).toBeTruthy()
      await page.screenshot({ path: 'test-results/INC-1-03-GET验证.png', fullPage: false })
    })

    // ---- 4. Verify timeline has "triggered" entry ----
    await test.step('验证时间线有 triggered 条目', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/timeline`)
      expect(res.code).toBe(0)
      const timeline = res.data
      expect(Array.isArray(timeline)).toBe(true)
      expect(timeline.length).toBeGreaterThanOrEqual(1)

      const triggeredEntry = timeline.find((t: any) => t.action === 'triggered')
      expect(triggeredEntry).toBeTruthy()
      expect(triggeredEntry.incident_id).toBe(incidentId)
      await page.screenshot({ path: 'test-results/INC-1-04-时间线验证.png', fullPage: false })
    })

    // ---- 5. Close the incident ----
    await test.step('关闭故障', async () => {
      const res = await API.post(page, `${API_BASE}/incidents/${incidentId}/close`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/INC-1-05-关闭成功.png', fullPage: false })
    })

    // ---- 6. Verify status is closed ----
    await test.step('验证状态为 closed', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}`)
      expect(res.code).toBe(0)
      expect(res.data.status).toBe('closed')
      expect(res.data.closed_at).toBeTruthy()
      await page.screenshot({ path: 'test-results/INC-1-06-状态验证.png', fullPage: false })
    })

    // Mark as cleaned up
    incidentId = null
  } finally {
    if (incidentId) await closeIncident(page, incidentId)
    if (channelId) await cleanupChannel(page, channelId)
  }
})

// ---------------------------------------------------------------------------
// INC-2: Create -> acknowledge -> verify processing -> snooze -> verify silenced_until -> close -> verify closed
// ---------------------------------------------------------------------------
test('INC-2 故障确认-暂缓-关闭生命周期', async ({ authPage: page }) => {
  let channelId: number | null = null
  let incidentId: number | null = null

  try {
    // ---- 1. Create channel and incident ----
    await test.step('创建协作空间和故障', async () => {
      channelId = await createChannel(page)
      const inc = await createIncident(page, channelId)
      incidentId = inc.id
      expect(inc.status).toBe('triggered')
      await page.screenshot({ path: 'test-results/INC-2-01-创建成功.png', fullPage: false })
    })

    // ---- 2. Acknowledge the incident ----
    await test.step('确认故障', async () => {
      const res = await API.post(page, `${API_BASE}/incidents/${incidentId}/acknowledge`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/INC-2-02-确认成功.png', fullPage: false })
    })

    // ---- 3. Verify status is processing ----
    await test.step('验证状态为 processing', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}`)
      expect(res.code).toBe(0)
      expect(res.data.status).toBe('processing')
      expect(res.data.acknowledged_at).toBeTruthy()
      await page.screenshot({ path: 'test-results/INC-2-03-状态processing.png', fullPage: false })
    })

    // ---- 4. Snooze the incident for 2 hours ----
    await test.step('暂缓故障 2 小时', async () => {
      const until = new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString()
      const res = await API.post(page, `${API_BASE}/incidents/${incidentId}/snooze`, { until })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/INC-2-04-暂缓成功.png', fullPage: false })
    })

    // ---- 5. Verify snoozed_until is set ----
    await test.step('验证 snoozed_until 已设置', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}`)
      expect(res.code).toBe(0)
      expect(res.data.snoozed_until).toBeTruthy()
      // snoozed_until should be a future time
      const snoozedUntil = new Date(res.data.snoozed_until).getTime()
      expect(snoozedUntil).toBeGreaterThan(Date.now())
      await page.screenshot({ path: 'test-results/INC-2-05-暂缓验证.png', fullPage: false })
    })

    // ---- 6. Close the incident ----
    await test.step('关闭故障', async () => {
      const res = await API.post(page, `${API_BASE}/incidents/${incidentId}/close`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/INC-2-06-关闭成功.png', fullPage: false })
    })

    // ---- 7. Verify status is closed ----
    await test.step('验证状态为 closed', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}`)
      expect(res.code).toBe(0)
      expect(res.data.status).toBe('closed')
      expect(res.data.closed_at).toBeTruthy()
      await page.screenshot({ path: 'test-results/INC-2-07-状态closed.png', fullPage: false })
    })

    // Mark as cleaned up
    incidentId = null
  } finally {
    if (incidentId) await closeIncident(page, incidentId)
    if (channelId) await cleanupChannel(page, channelId)
  }
})

// ---------------------------------------------------------------------------
// INC-3: Create -> add comment -> verify comment in timeline -> reassign -> verify assigned_to changed
// ---------------------------------------------------------------------------
test('INC-3 故障评论和转派', async ({ authPage: page }) => {
  let channelId: number | null = null
  let incidentId: number | null = null

  try {
    // ---- 1. Create channel and incident ----
    await test.step('创建协作空间和故障', async () => {
      channelId = await createChannel(page)
      const inc = await createIncident(page, channelId, {
        severity: 'warning',
      })
      incidentId = inc.id
      expect(inc.status).toBe('triggered')
      expect(inc.assigned_to).toBeFalsy() // no assignee initially
      await page.screenshot({ path: 'test-results/INC-3-01-创建成功.png', fullPage: false })
    })

    // ---- 2. Add a comment ----
    const commentText = `Test comment from functional test ${uid()}`
    await test.step('添加评论', async () => {
      const res = await API.post(page, `${API_BASE}/incidents/${incidentId}/comment`, {
        content: commentText,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/INC-3-02-评论成功.png', fullPage: false })
    })

    // ---- 3. Verify comment appears in timeline ----
    await test.step('验证评论出现在时间线中', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/timeline`)
      expect(res.code).toBe(0)
      const timeline = res.data
      expect(Array.isArray(timeline)).toBe(true)

      const commentEntry = timeline.find(
        (t: any) => t.action === 'commented' && t.content === commentText
      )
      expect(commentEntry).toBeTruthy()
      expect(commentEntry.incident_id).toBe(incidentId)
      expect(commentEntry.content).toBe(commentText)
      await page.screenshot({ path: 'test-results/INC-3-03-评论验证.png', fullPage: false })
    })

    // ---- 4. Get admin user ID (user 1 is admin) for reassignment ----
    let adminUserId: number
    await test.step('获取用户列表用于转派', async () => {
      const res = await API.get(page, `${API_BASE}/users?page=1&page_size=10`)
      expect(res.code).toBe(0)
      const users = res.data.list || res.data
      expect(Array.isArray(users)).toBe(true)
      expect(users.length).toBeGreaterThanOrEqual(1)
      // Pick the first user for reassignment
      adminUserId = users[0].id
      await page.screenshot({ path: 'test-results/INC-3-04-获取用户.png', fullPage: false })
    })

    // ---- 5. Reassign the incident ----
    await test.step('转派故障', async () => {
      const res = await API.post(page, `${API_BASE}/incidents/${incidentId}/reassign`, {
        user_id: adminUserId,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/INC-3-05-转派成功.png', fullPage: false })
    })

    // ---- 6. Verify assigned_to changed ----
    await test.step('验证 assigned_to 已变更', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}`)
      expect(res.code).toBe(0)
      expect(res.data.assigned_to).toBe(adminUserId)
      await page.screenshot({ path: 'test-results/INC-3-06-转派验证.png', fullPage: false })
    })

    // ---- 7. Verify timeline has "reassigned" entry ----
    await test.step('验证时间线有 reassigned 条目', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/timeline`)
      expect(res.code).toBe(0)
      const timeline = res.data
      const reassignedEntry = timeline.find((t: any) => t.action === 'reassigned')
      expect(reassignedEntry).toBeTruthy()
      await page.screenshot({ path: 'test-results/INC-3-07-转派时间线.png', fullPage: false })
    })
  } finally {
    if (incidentId) await closeIncident(page, incidentId)
    if (channelId) await cleanupChannel(page, channelId)
  }
})

// ---------------------------------------------------------------------------
// INC-4: Create -> verify timeline "triggered" -> acknowledge -> verify "acknowledged" entry
// ---------------------------------------------------------------------------
test('INC-4 故障时间线生命周期验证', async ({ authPage: page }) => {
  let channelId: number | null = null
  let incidentId: number | null = null

  try {
    // ---- 1. Create channel and incident ----
    await test.step('创建协作空间和故障', async () => {
      channelId = await createChannel(page)
      const inc = await createIncident(page, channelId, {
        severity: 'info',
        description: 'INC-4 timeline lifecycle test',
      })
      incidentId = inc.id
      await page.screenshot({ path: 'test-results/INC-4-01-创建成功.png', fullPage: false })
    })

    // ---- 2. Verify timeline has "triggered" entry ----
    await test.step('验证时间线有 triggered 条目', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/timeline`)
      expect(res.code).toBe(0)
      const timeline = res.data
      expect(Array.isArray(timeline)).toBe(true)
      expect(timeline.length).toBeGreaterThanOrEqual(1)

      const triggeredEntry = timeline.find((t: any) => t.action === 'triggered')
      expect(triggeredEntry).toBeTruthy()
      expect(triggeredEntry.incident_id).toBe(incidentId)
      // Content should describe the trigger event
      expect(triggeredEntry.content).toBeTruthy()
      await page.screenshot({ path: 'test-results/INC-4-02-triggered时间线.png', fullPage: false })
    })

    // ---- 3. Acknowledge the incident ----
    await test.step('确认故障', async () => {
      const res = await API.post(page, `${API_BASE}/incidents/${incidentId}/acknowledge`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/INC-4-03-确认成功.png', fullPage: false })
    })

    // ---- 4. Verify timeline has "acknowledged" entry ----
    await test.step('验证时间线有 acknowledged 条目', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/timeline`)
      expect(res.code).toBe(0)
      const timeline = res.data
      expect(Array.isArray(timeline)).toBe(true)

      const ackEntry = timeline.find((t: any) => t.action === 'acknowledged')
      expect(ackEntry).toBeTruthy()
      expect(ackEntry.incident_id).toBe(incidentId)
      // Should have an actor (the user who acknowledged)
      expect(ackEntry.actor_id).toBeTruthy()
      await page.screenshot({ path: 'test-results/INC-4-04-acknowledged时间线.png', fullPage: false })
    })

    // ---- 5. Verify timeline order: triggered before acknowledged ----
    await test.step('验证时间线顺序: triggered 在 acknowledged 之前', async () => {
      const res = await API.get(page, `${API_BASE}/incidents/${incidentId}/timeline`)
      expect(res.code).toBe(0)
      const timeline = res.data

      const triggeredIdx = timeline.findIndex((t: any) => t.action === 'triggered')
      const ackIdx = timeline.findIndex((t: any) => t.action === 'acknowledged')
      expect(triggeredIdx).toBeGreaterThanOrEqual(0)
      expect(ackIdx).toBeGreaterThanOrEqual(0)
      expect(triggeredIdx).toBeLessThan(ackIdx)

      // Close the incident and verify "closed" entry
      await API.post(page, `${API_BASE}/incidents/${incidentId}/close`)
      const res2 = await API.get(page, `${API_BASE}/incidents/${incidentId}/timeline`)
      const closedEntry = res2.data.find((t: any) => t.action === 'closed')
      expect(closedEntry).toBeTruthy()

      // Verify full lifecycle order
      const closedIdx = res2.data.findIndex((t: any) => t.action === 'closed')
      expect(closedIdx).toBeGreaterThan(ackIdx)

      await page.screenshot({ path: 'test-results/INC-4-05-完整生命周期.png', fullPage: false })
    })

    // Mark as cleaned up
    incidentId = null
  } finally {
    if (incidentId) await closeIncident(page, incidentId)
    if (channelId) await cleanupChannel(page, channelId)
  }
})
