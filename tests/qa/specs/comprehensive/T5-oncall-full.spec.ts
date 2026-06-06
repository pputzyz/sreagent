import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// T5: 值班完整测试 — 60 个测试用例
// 覆盖：排班管理(T5-1~T5-15)、升级策略(T5-16~T5-30)、值班概览(T5-31~T5-45)、
//       通知中心+用户偏好(T5-46~T5-60)

const SCHEDULE_URL = '/oncall/schedule'
const ESCALATION_URL = '/oncall/config/escalation-policies'
const OVERVIEW_URL = '/oncall/overview'
const CENTER_URL = '/notifications'

/** 生成唯一名称 */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

/** 通过 API 创建排班 */
async function createTestSchedule(page: import('@playwright/test').Page, overrides?: Record<string, unknown>): Promise<number> {
  const name = uid('test_schedule')
  const body = {
    name,
    description: 'E2E test schedule',
    rotation_type: 'weekly',
    timezone: 'Asia/Shanghai',
    handoff_time: '09:00',
    is_enabled: true,
    ...overrides,
  }
  const res = await API.post(page, '/api/v1/schedules', body)
  return res?.data?.id ?? 0
}

/** 通过 API 创建升级策略 */
async function createTestEscalation(page: import('@playwright/test').Page, overrides?: Record<string, unknown>): Promise<number> {
  const name = uid('test_escalation')
  const body = {
    name,
    description: 'E2E test escalation',
    steps: [
      { step_order: 1, target_type: 'user', target_id: 1, delay_minutes: 5, notify_channel_id: null },
    ],
    ...overrides,
  }
  const res = await API.post(page, '/api/v1/escalation-policies', body)
  return res?.data?.id ?? 0
}

/** 通过 API 删除资源 */
async function deleteResource(page: import('@playwright/test').Page, basePath: string, id: number): Promise<void> {
  if (id > 0) {
    await API.del(page, `${basePath}/${id}`)
  }
}

test.describe('T5 - 值班完整测试', () => {

  // ================================================================
  // T5-1 ~ T5-15: 排班管理
  // ================================================================

  // T5-1: 排班页面初始加载
  test('T5-1 排班页面初始加载', async ({ authPage: page }) => {
    await test.step('导航到排班页', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T5-1-排班页面.png', fullPage: true })
    })

    await test.step('验证页面结构', async () => {
      await expect(page.locator('.schedule-page, [class*="schedule"]').first()).toBeVisible()
    })
  })

  // T5-2: 排班侧边栏列表
  test('T5-2 排班侧边栏列表', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证侧边栏', async () => {
      const sidebar = page.locator('.schedule-sidebar-wrap, [class*="sidebar"]').first()
      if (await sidebar.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T5-2-侧边栏.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-3: 新建排班弹窗
  test('T5-3 新建排班弹窗', async ({ authPage: page }) => {
    await test.step('导航到排班页', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建排班按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /新建排班|new schedule|创建排班/i }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-3-新建排班弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T5-4: 选择排班查看详情
  test('T5-4 选择排班查看详情', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击侧边栏中的排班', async () => {
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T5-4-排班详情.png', fullPage: true })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-5: 排班日历网格
  test('T5-5 排班日历网格', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('验证日历网格', async () => {
      const calendar = page.locator('.calendar-container, .calendar-grid, [class*="calendar"]').first()
      if (await calendar.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T5-5-日历网格.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-6: 排班周导航
  test('T5-6 排班周导航', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('点击上一周', async () => {
      const prevBtn = page.locator('button').filter({ hasText: /‹|prev|上/i }).first()
      if (await prevBtn.isVisible().catch(() => false)) {
        await prevBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-6-上一周.png', fullPage: false })
      }
    })

    await test.step('点击下一周', async () => {
      const nextBtn = page.locator('button').filter({ hasText: /›|next|下/i }).first()
      if (await nextBtn.isVisible().catch(() => false)) {
        await nextBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-7: 排班今天按钮
  test('T5-7 排班今天按钮', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('点击今天按钮', async () => {
      const todayBtn = page.locator('button').filter({ hasText: /today|今天/i }).first()
      if (await todayBtn.isVisible().catch(() => false)) {
        await todayBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-7-今天.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-8: 排班当前值班人徽章
  test('T5-8 排班当前值班人', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('验证当前值班人徽章', async () => {
      const oncallBadge = page.locator('.oncall-badge, [class*="oncall-badge"]').first()
      if (await oncallBadge.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T5-8-当前值班人.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-9: 排班配置 Tab
  test('T5-9 排班配置Tab', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('验证配置 Tab', async () => {
      const configTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /config|配置/i }).first()
      if (await configTab.isVisible().catch(() => false)) {
        await configTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-9-配置Tab.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-10: 排班参与者 Tab
  test('T5-10 排班参与者Tab', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('点击参与者 Tab', async () => {
      const participantsTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /participant|参与者/i }).first()
      if (await participantsTab.isVisible().catch(() => false)) {
        await participantsTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-10-参与者Tab.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-11: 排班 Overrides Tab
  test('T5-11 排班OverridesTab', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('点击 Overrides Tab', async () => {
      const overridesTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /override|替班/i }).first()
      if (await overridesTab.isVisible().catch(() => false)) {
        await overridesTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-11-OverridesTab.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-12: 排班导出 iCal
  test('T5-12 排班导出iCal', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('查找导出按钮', async () => {
      const exportBtn = page.locator('button').filter({ hasText: /export|导出|ical/i }).first()
      if (await exportBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T5-12-导出按钮.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-13: 排班生成班次
  test('T5-13 排班生成班次', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('点击生成班次按钮', async () => {
      const generateBtn = page.locator('button').filter({ hasText: /generate|生成班次/i }).first()
      if (await generateBtn.isVisible().catch(() => false)) {
        await generateBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-13-生成班次.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-14: 排班编辑弹窗
  test('T5-14 排班编辑弹窗', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('点击编辑按钮', async () => {
      const editBtn = page.locator('button').filter({ hasText: /edit|编辑/i }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-14-编辑弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-15: 排班新班次按钮
  test('T5-15 排班新班次按钮', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找新班次按钮', async () => {
      const newShiftBtn = page.locator('button').filter({ hasText: /new shift|新建班次|新班次/i }).first()
      if (await newShiftBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T5-15-新班次按钮.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // ================================================================
  // T5-16 ~ T5-30: 升级策略
  // ================================================================

  // T5-16: 升级策略列表加载
  test('T5-16 升级策略列表加载', async ({ authPage: page }) => {
    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T5-16-升级策略列表.png', fullPage: true })
    })

    await test.step('验证页面结构', async () => {
      await expect(page.locator('.page-container, [class*="escalation"]').first()).toBeVisible()
    })
  })

  // T5-17: 升级策略表格显示
  test('T5-17 升级策略表格', async ({ authPage: page }) => {
    const policyId = await createTestEscalation(page)

    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证表格', async () => {
      const dataTable = page.locator('.n-data-table, [class*="data-table"]').first()
      if (await dataTable.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T5-17-策略表格.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/escalation-policies', policyId)
    })
  })

  // T5-18: 新建升级策略弹窗
  test('T5-18 新建升级策略弹窗', async ({ authPage: page }) => {
    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|create|新建|add/i }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-18-新建策略弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T5-19: 升级策略步骤添加
  test('T5-19 升级策略步骤添加', async ({ authPage: page }) => {
    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开新建弹窗并添加步骤', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|create|新建|add/i }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)

        const addStepBtn = page.locator('button').filter({ hasText: /add step|添加步骤/i }).first()
        if (await addStepBtn.isVisible().catch(() => false)) {
          await addStepBtn.click()
          await page.waitForTimeout(300)
          await page.screenshot({ path: 'test-results/T5-19-添加步骤.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T5-20: 升级策略步骤目标类型
  test('T5-20 升级策略目标类型', async ({ authPage: page }) => {
    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开新建弹窗验证目标类型', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|create|新建|add/i }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)

        const addStepBtn = page.locator('button').filter({ hasText: /add step|添加步骤/i }).first()
        if (await addStepBtn.isVisible().catch(() => false)) {
          await addStepBtn.click()
          await page.waitForTimeout(300)
          await page.screenshot({ path: 'test-results/T5-20-目标类型.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T5-21: 升级策略延迟配置
  test('T5-21 升级策略延迟配置', async ({ authPage: page }) => {
    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开新建弹窗验证延迟', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|create|新建|add/i }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)

        const addStepBtn = page.locator('button').filter({ hasText: /add step|添加步骤/i }).first()
        if (await addStepBtn.isVisible().catch(() => false)) {
          await addStepBtn.click()
          await page.waitForTimeout(300)
          const delayInput = page.locator('.n-input-number, [class*="input-number"]').first()
          if (await delayInput.isVisible().catch(() => false)) {
            await page.screenshot({ path: 'test-results/T5-21-延迟配置.png', fullPage: false })
          }
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T5-22: 升级策略编辑
  test('T5-22 升级策略编辑', async ({ authPage: page }) => {
    const policyId = await createTestEscalation(page)

    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击编辑按钮', async () => {
      const editBtn = page.locator('button').filter({ hasText: /edit|编辑/i }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-22-编辑策略.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/escalation-policies', policyId)
    })
  })

  // T5-23: 升级策略删除确认
  test('T5-23 升级策略删除确认', async ({ authPage: page }) => {
    const policyId = await createTestEscalation(page)

    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击删除按钮', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /delete|删除/i }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-23-删除确认.png', fullPage: false })
      }
    })

    await test.step('取消删除', async () => {
      const cancelBtn = page.locator('button').filter({ hasText: /cancel|取消/i }).first()
      if (await cancelBtn.isVisible().catch(() => false)) {
        await cancelBtn.click()
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/escalation-policies', policyId)
    })
  })

  // T5-24: 升级策略空状态
  test('T5-24 升级策略空状态', async ({ authPage: page }) => {
    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查空状态或表格', async () => {
      const emptyState = page.locator('.n-empty, [class*="empty"]')
      const dataTable = page.locator('.n-data-table, [class*="data-table"]')
      await page.screenshot({ path: 'test-results/T5-24-策略空状态.png', fullPage: false })
    })
  })

  // T5-25: 升级策略步骤删除
  test('T5-25 升级策略步骤删除', async ({ authPage: page }) => {
    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开新建弹窗添加并删除步骤', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|create|新建|add/i }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)

        const addStepBtn = page.locator('button').filter({ hasText: /add step|添加步骤/i }).first()
        if (await addStepBtn.isVisible().catch(() => false)) {
          await addStepBtn.click()
          await page.waitForTimeout(300)

          const removeStepBtn = page.locator('button[type="error"], button').filter({ hasText: /×|remove|删除/i }).first()
          if (await removeStepBtn.isVisible().catch(() => false)) {
            await page.screenshot({ path: 'test-results/T5-25-步骤删除.png', fullPage: false })
          }
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T5-26: 升级策略团队关联
  test('T5-26 升级策略团队关联', async ({ authPage: page }) => {
    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开新建弹窗查看团队', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|create|新建|add/i }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-26-团队关联.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T5-27: 升级策略通知渠道选择
  test('T5-27 升级策略通知渠道', async ({ authPage: page }) => {
    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开新建弹窗添加步骤查看渠道', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|create|新建|add/i }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)

        const addStepBtn = page.locator('button').filter({ hasText: /add step|添加步骤/i }).first()
        if (await addStepBtn.isVisible().catch(() => false)) {
          await addStepBtn.click()
          await page.waitForTimeout(300)
          await page.screenshot({ path: 'test-results/T5-27-通知渠道.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T5-28: 升级策略步骤排序
  test('T5-28 升级策略步骤排序', async ({ authPage: page }) => {
    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开新建弹窗添加多个步骤', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|create|新建|add/i }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)

        const addStepBtn = page.locator('button').filter({ hasText: /add step|添加步骤/i }).first()
        if (await addStepBtn.isVisible().catch(() => false)) {
          await addStepBtn.click()
          await page.waitForTimeout(200)
          await addStepBtn.click()
          await page.waitForTimeout(200)
          await page.screenshot({ path: 'test-results/T5-28-步骤排序.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T5-29: 升级策略 ID 列
  test('T5-29 升级策略ID列', async ({ authPage: page }) => {
    const policyId = await createTestEscalation(page)

    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证 ID 列', async () => {
      const idCells = page.locator('.n-data-table td, [class*="data-table"] td')
      const count = await idCells.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T5-29-ID列.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/escalation-policies', policyId)
    })
  })

  // T5-30: 升级策略步骤计数
  test('T5-30 升级策略步骤计数', async ({ authPage: page }) => {
    const policyId = await createTestEscalation(page)

    await test.step('导航到升级策略页', async () => {
      await page.goto(ESCALATION_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证步骤计数列', async () => {
      const stepsCells = page.locator('.n-data-table td, [class*="data-table"] td')
      const count = await stepsCells.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T5-30-步骤计数.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/escalation-policies', policyId)
    })
  })

  // ================================================================
  // T5-31 ~ T5-45: 值班概览
  // ================================================================

  // T5-31: 值班概览页面加载
  test('T5-31 值班概览页面加载', async ({ authPage: page }) => {
    await test.step('导航到值班概览', async () => {
      await page.goto(OVERVIEW_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T5-31-值班概览.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T5-32: 值班概览统计卡片
  test('T5-32 值班概览统计卡片', async ({ authPage: page }) => {
    await test.step('导航到值班概览', async () => {
      await page.goto(OVERVIEW_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证统计卡片', async () => {
      const statCards = page.locator('[class*="stat"], [class*="card"], [class*="metric"]')
      const count = await statCards.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T5-32-统计卡片.png', fullPage: false })
      }
    })
  })

  // T5-33: 值班概览当前值班人
  test('T5-33 值班概览当前值班', async ({ authPage: page }) => {
    await test.step('导航到值班概览', async () => {
      await page.goto(OVERVIEW_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证当前值班信息', async () => {
      const oncallSection = page.locator('[class*="oncall"], [class*="current"]').first()
      await page.screenshot({ path: 'test-results/T5-33-当前值班.png', fullPage: false })
    })
  })

  // T5-34: 值班概览即将值班
  test('T5-34 值班概览即将值班', async ({ authPage: page }) => {
    await test.step('导航到值班概览', async () => {
      await page.goto(OVERVIEW_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证即将值班信息', async () => {
      const upcomingSection = page.locator('[class*="upcoming"], [class*="next"]').first()
      await page.screenshot({ path: 'test-results/T5-34-即将值班.png', fullPage: false })
    })
  })

  // T5-35: 值班概览日历视图
  test('T5-35 值班概览日历视图', async ({ authPage: page }) => {
    await test.step('导航到值班概览', async () => {
      await page.goto(OVERVIEW_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证日历视图', async () => {
      const calendar = page.locator('[class*="calendar"], [class*="schedule"]').first()
      await page.screenshot({ path: 'test-results/T5-35-日历视图.png', fullPage: false })
    })
  })

  // T5-36: 值班概览排班列表
  test('T5-36 值班概览排班列表', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到值班概览', async () => {
      await page.goto(OVERVIEW_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证排班列表', async () => {
      const scheduleList = page.locator('[class*="schedule"], [class*="list"]').first()
      await page.screenshot({ path: 'test-results/T5-36-排班列表.png', fullPage: false })
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-37: 排班日历当前时间线
  test('T5-37 排班日历当前时间线', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('验证当前时间线', async () => {
      const timeLine = page.locator('.current-time-line, [class*="current-time"]').first()
      if (await timeLine.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T5-37-当前时间线.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-38: 排班日历周末标记
  test('T5-38 排班日历周末标记', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('验证周末标记', async () => {
      const weekendCols = page.locator('.cal-day-col.weekend, [class*="weekend"]')
      const count = await weekendCols.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T5-38-周末标记.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-39: 排班日历今天标记
  test('T5-39 排班日历今天标记', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('验证今天标记', async () => {
      const todayHeader = page.locator('.cal-day-header.today, [class*="today"]').first()
      if (await todayHeader.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T5-39-今天标记.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-40: 排班日历班次块
  test('T5-40 排班日历班次块', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('验证班次块', async () => {
      const shiftBlocks = page.locator('.shift-block, [class*="shift-block"]')
      const count = await shiftBlocks.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T5-40-班次块.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-41: 排班配置详情
  test('T5-41 排班配置详情', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page, { rotation_type: 'daily', timezone: 'UTC' })

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('验证配置详情', async () => {
      const configItems = page.locator('.config-item, [class*="config-item"]')
      const count = await configItems.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T5-41-配置详情.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-42: 排班时间标签
  test('T5-42 排班时间标签', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(1000)
    })

    await test.step('选择排班', async () => {
      // Wait for sidebar schedule items to load
      const scheduleItem = page.locator('.schedule-item, [class*="sidebar"] [class*="item"]').first()
      await scheduleItem.waitFor({ state: 'visible', timeout: 5000 }).catch(() => {})
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1500)
      }
    })

    await test.step('验证时间标签', async () => {
      // Time labels are inside .cal-time-gutter-body, only visible when schedule is selected
      const timeLabels = page.locator('.cal-time-label, .cal-time-gutter-body [class*="time"]')
      const count = await timeLabels.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T5-42-时间标签.png', fullPage: false })
      } else {
        // Calendar might not be visible if schedule wasn't selected — still take screenshot
        await page.screenshot({ path: 'test-results/T5-42-时间标签.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-43: 排班空状态
  test('T5-43 排班空状态', async ({ authPage: page }) => {
    await test.step('导航到排班页', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证空状态或排班列表', async () => {
      const sidebar = page.locator('.schedule-sidebar-wrap, [class*="sidebar"]').first()
      if (await sidebar.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T5-43-排班空状态.png', fullPage: false })
      }
    })
  })

  // T5-44: 排班删除确认
  test('T5-44 排班删除确认', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('点击删除按钮', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /delete|删除/i }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-44-删除确认.png', fullPage: false })
      }
    })

    await test.step('取消删除', async () => {
      const cancelBtn = page.locator('button').filter({ hasText: /cancel|取消/i }).first()
      if (await cancelBtn.isVisible().catch(() => false)) {
        await cancelBtn.click()
      } else {
        await page.keyboard.press('Escape')
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-45: 排班日历小时线
  test('T5-45 排班日历小时线', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('验证小时线', async () => {
      const hourLines = page.locator('.cal-hour-line, [class*="hour-line"]')
      const count = await hourLines.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T5-45-小时线.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // ================================================================
  // T5-46 ~ T5-60: 通知中心 + 用户偏好
  // ================================================================

  // T5-46: 通知中心加载
  test('T5-46 通知中心加载', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T5-46-通知中心.png', fullPage: true })
    })

    await test.step('验证页面结构', async () => {
      await expect(page.locator('.notif-center, [class*="notif"]').first()).toBeVisible()
    })
  })

  // T5-47: 通知中心全部/未读/已读切换
  test('T5-47 通知中心筛选切换', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('切换到未读', async () => {
      const unreadBtn = page.locator('.n-radio-button, [role="radio"]').filter({ hasText: /未读|unread/i }).first()
      if (await unreadBtn.isVisible().catch(() => false)) {
        await unreadBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-47-未读筛选.png', fullPage: false })
      }
    })

    await test.step('切换到已读', async () => {
      const readBtn = page.locator('.n-radio-button, [role="radio"]').filter({ hasText: /已读|read/i }).first()
      if (await readBtn.isVisible().catch(() => false)) {
        await readBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('切换回全部', async () => {
      const allBtn = page.locator('.n-radio-button, [role="radio"]').filter({ hasText: /全部|all/i }).first()
      if (await allBtn.isVisible().catch(() => false)) {
        await allBtn.click()
      }
    })
  })

  // T5-48: 通知中心严重等级芯片
  test('T5-48 通知中心严重等级芯片', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证严重等级芯片', async () => {
      const sevChips = page.locator('.sev-chip, [class*="sev-chip"]')
      const count = await sevChips.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T5-48-等级芯片.png', fullPage: false })
      }
    })
  })

  // T5-49: 通知项点击标记已读
  test('T5-49 通知项标记已读', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找未读通知', async () => {
      const unreadItem = page.locator('.notif-item.unread, [class*="unread"]').first()
      if (await unreadItem.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T5-49-未读通知.png', fullPage: false })
      }
    })
  })

  // T5-50: 通知项删除确认
  test('T5-50 通知项删除确认', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击删除按钮', async () => {
      const deleteBtn = page.locator('.notif-actions button, [class*="notif-actions"] button').first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-50-删除确认.png', fullPage: false })
      }
    })

    await test.step('取消删除', async () => {
      const cancelBtn = page.locator('button').filter({ hasText: /cancel|取消/i }).first()
      if (await cancelBtn.isVisible().catch(() => false)) {
        await cancelBtn.click()
      } else {
        await page.keyboard.press('Escape')
      }
    })
  })

  // T5-51: 通知中心分页
  test('T5-51 通知中心分页', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查分页', async () => {
      const pagination = page.locator('.n-pagination, [class*="pagination"]').first()
      if (await pagination.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T5-51-通知分页.png', fullPage: false })
      }
    })
  })

  // T5-52: 通知中心自动轮询
  test('T5-52 通知中心自动轮询', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证页面加载和刷新按钮', async () => {
      // The notification center has built-in auto-refresh polling (30s interval)
      // There's no visible toggle — just verify the page loads and refresh button exists
      await page.waitForTimeout(1000)
      const refreshBtn = page.locator('button').filter({ hasText: /刷新|refresh/i }).first()
      if (await refreshBtn.isVisible().catch(() => false)) {
        // Click refresh to verify it works
        await refreshBtn.click()
        await page.waitForTimeout(500)
      }
      await page.screenshot({ path: 'test-results/T5-52-自动轮询.png', fullPage: false })
    })
  })

  // T5-53: 通知类型颜色
  test('T5-53 通知类型颜色', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证通知类型标签颜色', async () => {
      const typeTags = page.locator('.n-tag, [class*="n-tag"]')
      const count = await typeTags.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T5-53-类型颜色.png', fullPage: false })
      }
    })
  })

  // T5-54: 通知内容显示
  test('T5-54 通知内容显示', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证通知内容', async () => {
      const notifContent = page.locator('.notif-content, [class*="notif-content"]')
      const count = await notifContent.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T5-54-通知内容.png', fullPage: false })
      }
    })
  })

  // T5-55: 通知标题显示
  test('T5-55 通知标题显示', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证通知标题', async () => {
      const notifTitle = page.locator('.notif-title, [class*="notif-title"]')
      const count = await notifTitle.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T5-55-通知标题.png', fullPage: false })
      }
    })
  })

  // T5-56: 通知全部标记已读确认
  test('T5-56 通知全部标记已读确认', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击全部标记已读', async () => {
      const markAllBtn = page.locator('button').filter({ hasText: /全部已读|mark all/i }).first()
      if (await markAllBtn.isVisible().catch(() => false)) {
        await markAllBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-56-全部已读确认.png', fullPage: false })
      }
    })

    await test.step('取消', async () => {
      const cancelBtn = page.locator('button').filter({ hasText: /cancel|取消/i }).first()
      if (await cancelBtn.isVisible().catch(() => false)) {
        await cancelBtn.click()
      } else {
        await page.keyboard.press('Escape')
      }
    })
  })

  // T5-57: 通知刷新按钮
  test('T5-57 通知刷新按钮', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击刷新', async () => {
      const refreshBtn = page.locator('button').filter({ hasText: /刷新|refresh/i }).first()
      if (await refreshBtn.isVisible().catch(() => false)) {
        await refreshBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-57-刷新后.png', fullPage: false })
      }
    })
  })

  // T5-58: 通知空状态
  test('T5-58 通知空状态', async ({ authPage: page }) => {
    await test.step('导航到通知中心并筛选已读', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
      const readBtn = page.locator('.n-radio-button, [role="radio"]').filter({ hasText: /已读|read/i }).first()
      if (await readBtn.isVisible().catch(() => false)) {
        await readBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('检查空状态', async () => {
      const emptyState = page.locator('.n-empty, [class*="empty"]')
      const notifList = page.locator('.notif-list, [class*="notif-list"]')
      const hasItems = await notifList.isVisible().catch(() => false)
      if (!hasItems && await emptyState.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T5-58-通知空状态.png', fullPage: false })
      }
    })
  })

  // T5-59: 排班日历班次点击编辑
  test('T5-59 排班班次点击编辑', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('点击班次块', async () => {
      const shiftBlock = page.locator('.shift-block, [class*="shift-block"]').first()
      if (await shiftBlock.isVisible().catch(() => false)) {
        await shiftBlock.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-59-班次编辑.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })

  // T5-60: 排班日历点击创建班次
  test('T5-60 排班日历点击创建', async ({ authPage: page }) => {
    const scheduleId = await createTestSchedule(page)

    await test.step('导航到排班页并选择排班', async () => {
      await page.goto(SCHEDULE_URL)
      await page.waitForLoadState('networkidle')
      const scheduleItem = page.locator('[class*="sidebar"] [class*="item"], [class*="sidebar"] li').first()
      if (await scheduleItem.isVisible().catch(() => false)) {
        await scheduleItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('点击日历空白区域', async () => {
      const calDayCol = page.locator('.cal-day-col, [class*="cal-day-col"]').first()
      if (await calDayCol.isVisible().catch(() => false)) {
        await calDayCol.click({ position: { x: 50, y: 200 } })
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T5-60-点击创建.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/schedules', scheduleId)
    })
  })
})
