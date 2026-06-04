import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: get the first available user ID for participant/shift assignments */
async function getFirstUserId(page: any): Promise<number> {
  const res = await API.get(page, `${API_BASE}/users?page=1&page_size=10`)
  expect(res.code).toBe(0)
  const users = res.data.list || res.data
  expect(Array.isArray(users)).toBe(true)
  expect(users.length).toBeGreaterThanOrEqual(1)
  return users[0].id
}

/** Helper: get two user IDs for rotation tests */
async function getTwoUserIds(page: any): Promise<[number, number]> {
  const res = await API.get(page, `${API_BASE}/users?page=1&page_size=10`)
  expect(res.code).toBe(0)
  const users = res.data.list || res.data
  expect(Array.isArray(users)).toBe(true)
  expect(users.length).toBeGreaterThanOrEqual(2)
  return [users[0].id, users[1].id]
}

/** Helper: create a schedule via API and return the created object */
async function createSchedule(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `test-schedule-${tag}`,
    description: `Functional test schedule ${tag}`,
    rotation_type: 'daily',
    timezone: 'Asia/Shanghai',
    handoff_time: '09:00',
    is_enabled: true,
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/schedules`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag }
}

/** Helper: create an escalation policy via API and return the created object */
async function createEscalationPolicy(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `test-escalation-${tag}`,
    description: `Functional test escalation policy ${tag}`,
    is_enabled: true,
    steps: [],
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/escalation-policies`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag }
}

/** Helper: delete a schedule by ID, ignoring errors */
async function cleanupSchedule(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/schedules/${id}`)
  } catch { /* ignore */ }
}

/** Helper: delete an escalation policy by ID, ignoring errors */
async function cleanupEscalation(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/escalation-policies/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// ONCALL-1: Create schedule -> verify -> add participants -> verify -> generate shifts -> verify
// ---------------------------------------------------------------------------
test('ONCALL-1 排班创建-参与者-班次生成', async ({ authPage: page }) => {
  let scheduleId: number | null = null

  try {
    // ---- 1. Create schedule ----
    await test.step('创建排班', async () => {
      const schedule = await createSchedule(page, {
        rotation_type: 'daily',
        handoff_time: '10:00',
        timezone: 'Asia/Shanghai',
      })
      scheduleId = schedule.id

      expect(schedule.name).toContain('test-schedule-')
      expect(schedule.rotation_type).toBe('daily')
      expect(schedule.handoff_time).toBe('10:00')
      expect(schedule.timezone).toBe('Asia/Shanghai')
      expect(schedule.is_enabled).toBe(true)
      await page.screenshot({ path: 'test-results/ONCALL-1-01-创建排班.png', fullPage: false })
    })

    // ---- 2. GET to verify schedule persisted ----
    await test.step('GET 验证排班已保存', async () => {
      const res = await API.get(page, `${API_BASE}/schedules/${scheduleId}`)
      expect(res.code).toBe(0)
      const s = res.data
      expect(s.id).toBe(scheduleId)
      expect(s.name).toContain('test-schedule-')
      expect(s.rotation_type).toBe('daily')
      expect(s.handoff_time).toBe('10:00')
      expect(s.timezone).toBe('Asia/Shanghai')
      expect(s.is_enabled).toBe(true)
      await page.screenshot({ path: 'test-results/ONCALL-1-02-GET验证.png', fullPage: false })
    })

    // ---- 3. Get user IDs for participants ----
    let userIds: [number, number]
    await test.step('获取用户列表', async () => {
      userIds = await getTwoUserIds(page)
      await page.screenshot({ path: 'test-results/ONCALL-1-03-获取用户.png', fullPage: false })
    })

    // ---- 4. Set participants ----
    await test.step('设置参与者', async () => {
      const res = await API.put(page, `${API_BASE}/schedules/${scheduleId}/participants`, {
        user_ids: userIds,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ONCALL-1-04-设置参与者.png', fullPage: false })
    })

    // ---- 5. Verify participants ----
    await test.step('验证参与者已保存', async () => {
      const res = await API.get(page, `${API_BASE}/schedules/${scheduleId}/participants`)
      expect(res.code).toBe(0)
      const participants = res.data
      expect(Array.isArray(participants)).toBe(true)
      expect(participants.length).toBe(2)

      // Verify user IDs match
      const participantUserIds = participants.map((p: any) => p.user_id).sort()
      expect(participantUserIds).toEqual([...userIds].sort())

      // Verify positions are set
      for (const p of participants) {
        expect(p.schedule_id).toBe(scheduleId)
        expect(p.position).toBeGreaterThanOrEqual(0)
      }
      await page.screenshot({ path: 'test-results/ONCALL-1-05-参与者验证.png', fullPage: false })
    })

    // ---- 6. Generate shifts ----
    await test.step('生成班次 (2 周)', async () => {
      const res = await API.post(page, `${API_BASE}/schedules/${scheduleId}/generate-shifts`, {
        weeks: 2,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ONCALL-1-06-生成班次.png', fullPage: false })
    })

    // ---- 7. Verify shifts were created ----
    await test.step('验证班次已生成', async () => {
      // List shifts for the next 2 weeks
      const now = new Date()
      const twoWeeksLater = new Date(now.getTime() + 14 * 24 * 60 * 60 * 1000)
      const start = now.toISOString()
      const end = twoWeeksLater.toISOString()

      const res = await API.get(
        page,
        `${API_BASE}/schedules/${scheduleId}/shifts?start=${encodeURIComponent(start)}&end=${encodeURIComponent(end)}`
      )
      expect(res.code).toBe(0)
      const shifts = res.data
      expect(Array.isArray(shifts)).toBe(true)
      // With 2 participants and daily rotation over 2 weeks, we expect ~14 shifts
      expect(shifts.length).toBeGreaterThanOrEqual(7)

      // Each shift should have valid fields
      for (const shift of shifts) {
        expect(shift.schedule_id).toBe(scheduleId)
        expect(shift.user_id).toBeGreaterThan(0)
        expect(shift.start_time).toBeTruthy()
        expect(shift.end_time).toBeTruthy()
        expect(new Date(shift.end_time).getTime()).toBeGreaterThan(new Date(shift.start_time).getTime())
      }
      await page.screenshot({ path: 'test-results/ONCALL-1-07-班次验证.png', fullPage: false })
    })
  } finally {
    if (scheduleId) await cleanupSchedule(page, scheduleId)
  }
})

// ---------------------------------------------------------------------------
// ONCALL-2: Create escalation policy -> add steps -> verify -> update -> verify changes
// ---------------------------------------------------------------------------
test('ONCALL-2 升级策略创建-步骤-更新', async ({ authPage: page }) => {
  let policyId: number | null = null

  try {
    // ---- 1. Get user ID for escalation steps ----
    let userId: number
    await test.step('获取用户 ID', async () => {
      userId = await getFirstUserId(page)
      await page.screenshot({ path: 'test-results/ONCALL-2-01-获取用户.png', fullPage: false })
    })

    // ---- 2. Create escalation policy with steps ----
    await test.step('创建升级策略 (含步骤)', async () => {
      const policy = await createEscalationPolicy(page, {
        name: `escalation-with-steps-${uid()}`,
        description: 'Test escalation with initial steps',
        steps: [
          {
            step_order: 1,
            target_type: 'user',
            target_id: userId,
            delay_minutes: 5,
          },
          {
            step_order: 2,
            target_type: 'user',
            target_id: userId,
            delay_minutes: 15,
          },
        ],
      })
      policyId = policy.id

      expect(policy.name).toContain('escalation-with-steps-')
      expect(policy.is_enabled).toBe(true)
      await page.screenshot({ path: 'test-results/ONCALL-2-02-创建成功.png', fullPage: false })
    })

    // ---- 3. Verify policy and steps are saved ----
    await test.step('GET 验证策略和步骤', async () => {
      const res = await API.get(page, `${API_BASE}/escalation-policies/${policyId}`)
      expect(res.code).toBe(0)
      const policy = res.data
      expect(policy.id).toBe(policyId)
      expect(policy.name).toContain('escalation-with-steps-')
      expect(policy.is_enabled).toBe(true)

      // Verify steps
      const steps = policy.steps
      expect(Array.isArray(steps)).toBe(true)
      expect(steps.length).toBe(2)

      // Sort by step_order for consistent assertions
      const sorted = [...steps].sort((a: any, b: any) => a.step_order - b.step_order)
      expect(sorted[0].step_order).toBe(1)
      expect(sorted[0].target_type).toBe('user')
      expect(sorted[0].target_id).toBe(userId)
      expect(sorted[0].delay_minutes).toBe(5)

      expect(sorted[1].step_order).toBe(2)
      expect(sorted[1].target_type).toBe('user')
      expect(sorted[1].target_id).toBe(userId)
      expect(sorted[1].delay_minutes).toBe(15)

      await page.screenshot({ path: 'test-results/ONCALL-2-03-步骤验证.png', fullPage: false })
    })

    // ---- 4. Update the escalation policy (add a 3rd step, change description) ----
    await test.step('更新升级策略 (添加第 3 步)', async () => {
      const res = await API.put(page, `${API_BASE}/escalation-policies/${policyId}`, {
        name: `escalation-updated-${uid()}`,
        description: 'Updated: added step 3 with 30min delay',
        is_enabled: true,
        steps: [
          {
            step_order: 1,
            target_type: 'user',
            target_id: userId,
            delay_minutes: 5,
          },
          {
            step_order: 2,
            target_type: 'user',
            target_id: userId,
            delay_minutes: 15,
          },
          {
            step_order: 3,
            target_type: 'user',
            target_id: userId,
            delay_minutes: 30,
          },
        ],
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ONCALL-2-04-更新成功.png', fullPage: false })
    })

    // ---- 5. Verify updates took effect ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/escalation-policies/${policyId}`)
      expect(res.code).toBe(0)
      const policy = res.data
      expect(policy.name).toContain('escalation-updated-')
      expect(policy.description).toBe('Updated: added step 3 with 30min delay')

      // Verify 3 steps now
      const steps = policy.steps
      expect(Array.isArray(steps)).toBe(true)
      expect(steps.length).toBe(3)

      const sorted = [...steps].sort((a: any, b: any) => a.step_order - b.step_order)
      expect(sorted[0].step_order).toBe(1)
      expect(sorted[0].delay_minutes).toBe(5)
      expect(sorted[1].step_order).toBe(2)
      expect(sorted[1].delay_minutes).toBe(15)
      expect(sorted[2].step_order).toBe(3)
      expect(sorted[2].delay_minutes).toBe(30)

      await page.screenshot({ path: 'test-results/ONCALL-2-05-更新验证.png', fullPage: false })
    })

    // ---- 6. Update to remove a step (back to 2 steps) ----
    await test.step('删除第 3 步', async () => {
      const res = await API.put(page, `${API_BASE}/escalation-policies/${policyId}`, {
        name: policyId ? `escalation-final-${policyId}` : 'escalation-final',
        description: 'Final state: 2 steps',
        is_enabled: true,
        steps: [
          {
            step_order: 1,
            target_type: 'user',
            target_id: userId,
            delay_minutes: 5,
          },
          {
            step_order: 2,
            target_type: 'user',
            target_id: userId,
            delay_minutes: 15,
          },
        ],
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ONCALL-2-06-删除步骤.png', fullPage: false })
    })

    // ---- 7. Verify step removal ----
    await test.step('验证步骤已删除', async () => {
      const res = await API.get(page, `${API_BASE}/escalation-policies/${policyId}`)
      expect(res.code).toBe(0)
      expect(res.data.steps.length).toBe(2)
      await page.screenshot({ path: 'test-results/ONCALL-2-07-删除验证.png', fullPage: false })
    })
  } finally {
    if (policyId) await cleanupEscalation(page, policyId)
  }
})

// ---------------------------------------------------------------------------
// ONCALL-3: Create schedule -> create override -> verify -> delete override -> verify deleted
// ---------------------------------------------------------------------------
test('ONCALL-3 排班替班创建-验证-删除', async ({ authPage: page }) => {
  let scheduleId: number | null = null
  let overrideId: number | null = null

  try {
    // ---- 1. Create schedule ----
    await test.step('创建排班', async () => {
      const schedule = await createSchedule(page, {
        rotation_type: 'weekly',
        handoff_time: '09:00',
      })
      scheduleId = schedule.id
      expect(schedule.rotation_type).toBe('weekly')
      await page.screenshot({ path: 'test-results/ONCALL-3-01-创建排班.png', fullPage: false })
    })

    // ---- 2. Get a user ID for the override ----
    let userId: number
    await test.step('获取用户 ID', async () => {
      userId = await getFirstUserId(page)
      await page.screenshot({ path: 'test-results/ONCALL-3-02-获取用户.png', fullPage: false })
    })

    // ---- 3. Create an override ----
    const startTime = new Date(Date.now() + 60 * 60 * 1000).toISOString() // 1 hour from now
    const endTime = new Date(Date.now() + 25 * 60 * 60 * 1000).toISOString() // 25 hours from now
    const overrideReason = `Test override ${uid()}`

    await test.step('创建替班', async () => {
      const res = await API.post(page, `${API_BASE}/schedules/${scheduleId}/overrides`, {
        user_id: userId,
        start_time: startTime,
        end_time: endTime,
        reason: overrideReason,
      })
      expect(res.code).toBe(0)
      const override = res.data
      expect(override).toBeTruthy()
      expect(override.id).toBeGreaterThan(0)
      overrideId = override.id

      expect(override.schedule_id).toBe(scheduleId)
      expect(override.user_id).toBe(userId)
      expect(override.reason).toBe(overrideReason)
      await page.screenshot({ path: 'test-results/ONCALL-3-03-替班创建.png', fullPage: false })
    })

    // ---- 4. Verify override exists in list ----
    await test.step('验证替班在列表中', async () => {
      const res = await API.get(page, `${API_BASE}/schedules/${scheduleId}/overrides`)
      expect(res.code).toBe(0)
      const overrides = res.data
      expect(Array.isArray(overrides)).toBe(true)
      expect(overrides.length).toBeGreaterThanOrEqual(1)

      const found = overrides.find((o: any) => o.id === overrideId)
      expect(found).toBeTruthy()
      expect(found.schedule_id).toBe(scheduleId)
      expect(found.user_id).toBe(userId)
      expect(found.reason).toBe(overrideReason)
      await page.screenshot({ path: 'test-results/ONCALL-3-04-替班验证.png', fullPage: false })
    })

    // ---- 5. Delete the override ----
    await test.step('删除替班', async () => {
      const res = await API.del(
        page,
        `${API_BASE}/schedules/${scheduleId}/overrides/${overrideId}`
      )
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/ONCALL-3-05-替班删除.png', fullPage: false })
    })

    // ---- 6. Verify override is gone ----
    await test.step('验证替班已删除', async () => {
      const res = await API.get(page, `${API_BASE}/schedules/${scheduleId}/overrides`)
      expect(res.code).toBe(0)
      const overrides = res.data
      expect(Array.isArray(overrides)).toBe(true)

      const found = overrides.find((o: any) => o.id === overrideId)
      expect(found).toBeFalsy()
      await page.screenshot({ path: 'test-results/ONCALL-3-06-删除验证.png', fullPage: false })
    })

    // Mark override as cleaned up
    overrideId = null
  } finally {
    if (overrideId && scheduleId) {
      try {
        await API.del(page, `${API_BASE}/schedules/${scheduleId}/overrides/${overrideId}`)
      } catch { /* ignore */ }
    }
    if (scheduleId) await cleanupSchedule(page, scheduleId)
  }
})
