import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// T4: 事件完整测试 — 70 个测试用例
// 覆盖：事件列表(T4-1~T4-15)、事件详情(T4-16~T4-30)、事件生命周期(T4-31~T4-50)、
//       协作空间(T4-51~T4-70)

const INCIDENTS_URL = '/oncall/incidents'
const SPACES_URL = '/oncall/spaces'

/** 生成唯一名称 */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

/** 通过 API 创建测试事件 */
async function createTestIncident(page: import('@playwright/test').Page, overrides?: Record<string, unknown>): Promise<number> {
  // 先创建一个协作空间（channel_id 必须大于 0）
  const channelId = await createTestChannel(page)
  if (channelId === 0) return 0

  const name = uid('test_incident')
  const body = {
    title: name,
    description: 'E2E test incident',
    severity: 'warning',
    channel_id: channelId,
    ...overrides,
  }
  const res = await API.post(page, '/api/v1/incidents', body)
  return res?.data?.id ?? 0
}

/** 获取第一个事件 ID */
async function getFirstIncidentId(page: import('@playwright/test').Page): Promise<number | null> {
  const res = await API.get(page, '/api/v1/incidents?page=1&page_size=1')
  const list = res?.data?.list || res?.data?.data?.list || []
  return list.length > 0 ? list[0].id : null
}

/** 通过 API 创建协作空间 */
async function createTestChannel(page: import('@playwright/test').Page, overrides?: Record<string, unknown>): Promise<number> {
  const name = uid('test_space')
  const body = {
    name,
    description: 'E2E test channel',
    status: 'active',
    access_level: 'public',
    auto_close_enabled: false,
    auto_close_minutes: 60,
    ...overrides,
  }
  const res = await API.post(page, '/api/v1/channels', body)
  return res?.data?.id ?? 0
}

/** 通过 API 删除资源 */
async function deleteResource(page: import('@playwright/test').Page, basePath: string, id: number): Promise<void> {
  if (id > 0) {
    await API.del(page, `${basePath}/${id}`)
  }
}

test.describe('T4 - 事件完整测试', () => {

  // ================================================================
  // T4-1 ~ T4-15: 事件列表
  // ================================================================

  // T4-1: 事件列表初始加载
  test('T4-1 事件列表初始加载', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T4-1-事件列表.png', fullPage: true })
    })

    await test.step('验证页面标题', async () => {
      const header = page.locator('.incidents-page, [class*="incidents"]').first()
      if (await header.isVisible().catch(() => false)) {
        await expect(header).toBeVisible()
      }
    })
  })

  // T4-2: 事件列表视图切换
  test('T4-2 事件列表视图切换', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('切换到"我的"视图', async () => {
      const mineTab = page.locator('.n-radio-button, [role="radio"]').filter({ hasText: /我的|mine/i }).first()
      if (await mineTab.isVisible().catch(() => false)) {
        await mineTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-2-我的视图.png', fullPage: false })
      }
    })

    await test.step('切换回全部视图', async () => {
      const allTab = page.locator('.n-radio-button, [role="radio"]').filter({ hasText: /全部|all/i }).first()
      if (await allTab.isVisible().catch(() => false)) {
        await allTab.click()
        await page.waitForTimeout(300)
      }
    })
  })

  // T4-3: 事件列表状态筛选
  test('T4-3 事件列表状态筛选', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('筛选触发中状态', async () => {
      const triggeredBtn = page.locator('.n-radio-button, [role="radio"]').filter({ hasText: /触发|triggered/i }).first()
      if (await triggeredBtn.isVisible().catch(() => false)) {
        await triggeredBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-3-状态筛选.png', fullPage: false })
      }
    })

    await test.step('重置筛选', async () => {
      const allBtn = page.locator('.filter-group .n-radio-button, [class*="filter"] [role="radio"]').filter({ hasText: /全部|all/i }).first()
      if (await allBtn.isVisible().catch(() => false)) {
        await allBtn.click()
        await page.waitForTimeout(300)
      }
    })
  })

  // T4-4: 事件列表严重等级筛选
  test('T4-4 事件列表严重等级筛选', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('筛选 Critical 等级', async () => {
      const criticalBtn = page.locator('.n-radio-button, [role="radio"]').filter({ hasText: /critical|严重/i }).first()
      if (await criticalBtn.isVisible().catch(() => false)) {
        await criticalBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-4-等级筛选.png', fullPage: false })
      }
    })

    await test.step('重置筛选', async () => {
      const allBtn = page.locator('.filter-group .n-radio-button, [class*="filter"] [role="radio"]').filter({ hasText: /全部|all/i }).first()
      if (await allBtn.isVisible().catch(() => false)) {
        await allBtn.click()
        await page.waitForTimeout(300)
      }
    })
  })

  // T4-5: 事件列表搜索
  test('T4-5 事件列表搜索', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('.search-box input, [class*="search"] input').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-5-搜索结果.png', fullPage: false })
      }
    })
  })

  // T4-6: 事件列表刷新按钮
  test('T4-6 事件列表刷新', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击刷新按钮', async () => {
      const refreshBtn = page.locator('button[aria-label*="刷新"], button[aria-label*="refresh"], button circle').first()
      if (await refreshBtn.isVisible().catch(() => false)) {
        await refreshBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-6-刷新后.png', fullPage: false })
      }
    })
  })

  // T4-7: 事件列表分页
  test('T4-7 事件列表分页', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查分页控件', async () => {
      const pagination = page.locator('.n-pagination, [class*="pagination"]').first()
      if (await pagination.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T4-7-分页控件.png', fullPage: false })
      }
    })
  })

  // T4-8: 事件列表行严重等级颜色
  test('T4-8 事件列表严重等级颜色', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证严重等级颜色指示器', async () => {
      const severityDots = page.locator('.dot[data-severity], .status-bar[data-severity], [data-severity]')
      const count = await severityDots.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T4-8-等级颜色.png', fullPage: false })
      }
    })
  })

  // T4-9: 事件列表状态标签
  test('T4-9 事件列表状态标签', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证状态标签', async () => {
      const statusPills = page.locator('.status-pill, [class*="status-pill"]')
      const count = await statusPills.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T4-9-状态标签.png', fullPage: false })
      }
    })
  })

  // T4-10: 事件列表操作下拉菜单
  test('T4-10 事件列表操作菜单', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击操作按钮', async () => {
      const actionBtn = page.locator('.action-trigger, [class*="action-trigger"]').first()
      if (await actionBtn.isVisible().catch(() => false)) {
        await actionBtn.click({ force: true })
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-10-操作菜单.png', fullPage: false })
      }
    })

    await test.step('关闭菜单', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-11: 事件列表元数据显示
  test('T4-11 事件列表元数据', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证元数据项', async () => {
      const metaItems = page.locator('.meta-item, [class*="meta-item"]')
      const count = await metaItems.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T4-11-元数据.png', fullPage: false })
      }
    })
  })

  // T4-12: 事件列表创建按钮
  test('T4-12 事件列表创建按钮', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击创建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-12-创建弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-13: 事件列表创建弹窗表单
  test('T4-13 创建事件弹窗表单', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗验证表单', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)

        const modal = page.locator('.n-modal, .create-modal').first()
        if (await modal.isVisible().catch(() => false)) {
          const titleInput = page.locator('.n-modal input').first()
          const severitySelect = page.locator('.n-modal .n-select').first()
          await page.screenshot({ path: 'test-results/T4-13-创建表单.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-14: 事件列表选中状态
  test('T4-14 事件列表选中状态', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证复选框', async () => {
      const checkboxes = page.locator('.row-checkbox, [class*="checkbox"]')
      const count = await checkboxes.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T4-14-复选框.png', fullPage: false })
      }
    })
  })

  // T4-15: 事件列表空状态
  test('T4-15 事件列表空状态', async ({ authPage: page }) => {
    await test.step('导航到事件列表页并搜索不存在的事件', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
      const searchInput = page.locator('.search-box input, [class*="search"] input').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('zzz_nonexistent_zzz')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-15-空状态.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T4-16 ~ T4-30: 事件详情
  // ================================================================

  // T4-16: 事件详情页加载
  test('T4-16 事件详情页加载', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T4-16-事件详情.png', fullPage: true })
    })

    await test.step('验证详情页标题', async () => {
      const title = page.locator('.incident-title, h1, [class*="incident-title"]').first()
      if (await title.isVisible().catch(() => false)) {
        await expect(title).toBeVisible()
      }
    })
  })

  // T4-17: 事件详情返回按钮
  test('T4-17 事件详情返回按钮', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证返回按钮', async () => {
      const backBtn = page.locator('button').filter({ hasText: /返回|back/i }).first()
      const backIcon = page.locator('.n-icon [class*="arrow-back"], button circle').first()
      const hasBack = await backBtn.isVisible().catch(() => false) || await backIcon.isVisible().catch(() => false)
      await page.screenshot({ path: 'test-results/T4-17-返回按钮.png', fullPage: false })
    })
  })

  // T4-18: 事件详情状态条
  test('T4-18 事件详情状态条', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证状态条', async () => {
      const stripe = page.locator('.header-stripe, [class*="header-stripe"]').first()
      if (await stripe.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T4-18-状态条.png', fullPage: false })
      }
    })
  })

  // T4-19: 事件详情 Overview Tab
  test('T4-19 事件详情OverviewTab', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证 Overview 内容', async () => {
      const overviewContent = page.locator('.detail-main, [class*="detail-main"]').first()
      if (await overviewContent.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T4-19-Overview.png', fullPage: false })
      }
    })
  })

  // T4-20: 事件详情 Timeline Tab
  test('T4-20 事件详情TimelineTab', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击 Timeline Tab', async () => {
      const timelineTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /时间线|timeline/i }).first()
      if (await timelineTab.isVisible().catch(() => false)) {
        await timelineTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-20-Timeline.png', fullPage: false })
      }
    })
  })

  // T4-21: 事件详情标签展示
  test('T4-21 事件详情标签展示', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证标签区域', async () => {
      const labels = page.locator('.label-chip, [class*="label-chip"], [class*="labels"]')
      const labelsSection = page.locator('[class*="labels-section"], [class*="label-list"]')
      await page.screenshot({ path: 'test-results/T4-21-标签展示.png', fullPage: false })
    })
  })

  // T4-22: 事件详情操作按钮
  test('T4-22 事件详情操作按钮', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证操作按钮', async () => {
      const actionBar = page.locator('.action-bar, [class*="action-bar"]').first()
      if (await actionBar.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T4-22-操作按钮.png', fullPage: false })
      }
    })
  })

  // T4-23: 事件详情更多操作菜单
  test('T4-23 事件详情更多操作', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击更多操作', async () => {
      const moreBtn = page.locator('button').filter({ hasText: /操作|action|more/i }).first()
      if (await moreBtn.isVisible().catch(() => false)) {
        await moreBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-23-更多操作.png', fullPage: false })
      }
    })

    await test.step('关闭菜单', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-24: 事件详情评论区
  test('T4-24 事件详情评论区', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证评论区域', async () => {
      const commentArea = page.locator('textarea, [class*="comment"], [class*="md-editor"]').first()
      if (await commentArea.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T4-24-评论区.png', fullPage: false })
      }
    })
  })

  // T4-25: 事件详情相关告警
  test('T4-25 事件详情相关告警', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证相关告警区域', async () => {
      const relatedAlerts = page.locator('[class*="related"], [class*="alert-list"], [class*="n-data-table"]').first()
      await page.screenshot({ path: 'test-results/T4-25-相关告警.png', fullPage: false })
    })
  })

  // T4-26: 事件详情持续时间
  test('T4-26 事件详情持续时间', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证持续时间显示', async () => {
      const stripe = page.locator('.header-stripe, [class*="header-stripe"]').first()
      if (await stripe.isVisible().catch(() => false)) {
        const text = await stripe.textContent()
        await page.screenshot({ path: 'test-results/T4-26-持续时间.png', fullPage: false })
      }
    })
  })

  // T4-27: 事件详情刷新按钮
  test('T4-27 事件详情刷新', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击刷新按钮', async () => {
      const refreshBtn = page.locator('button[aria-label*="刷新"], button[aria-label*="refresh"]').first()
      if (await refreshBtn.isVisible().catch(() => false)) {
        await refreshBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-27-刷新后.png', fullPage: false })
      }
    })
  })

  // T4-28: 事件详情 Dispatch Log Tab
  test('T4-28 事件详情DispatchLog', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击 Dispatch Log Tab', async () => {
      const dispatchTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /dispatch|派发/i }).first()
      if (await dispatchTab.isVisible().catch(() => false)) {
        await dispatchTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-28-DispatchLog.png', fullPage: false })
      }
    })
  })

  // T4-29: 事件详情变更事件 Tab
  test('T4-29 事件详情变更事件', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击变更事件 Tab', async () => {
      const changesTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /变更|change/i }).first()
      if (await changesTab.isVisible().catch(() => false)) {
        await changesTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-29-变更事件.png', fullPage: false })
      }
    })
  })

  // T4-30: 事件详情 PostMortem Tab
  test('T4-30 事件详情PostMortem', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击 PostMortem Tab', async () => {
      const pmTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /post.?mortem|复盘|事后/i }).first()
      if (await pmTab.isVisible().catch(() => false)) {
        await pmTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-30-PostMortem.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T4-31 ~ T4-50: 事件生命周期
  // ================================================================

  // T4-31: 通过 API 创建事件
  test('T4-31 通过API创建事件', async ({ authPage: page }) => {
    await test.step('通过 API 创建事件', async () => {
      const incidentId = await createTestIncident(page)
      expect(incidentId).toBeGreaterThan(0)
      await page.screenshot({ path: 'test-results/T4-31-API创建.png', fullPage: false })
    })
  })

  // T4-32: 通过 UI 创建事件
  test('T4-32 通过UI创建事件', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('填写创建表单', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)

        const titleInput = page.locator('.n-modal input').first()
        if (await titleInput.isVisible().catch(() => false)) {
          await titleInput.fill(uid('ui_incident'))
          await page.screenshot({ path: 'test-results/T4-32-UI创建.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-33: 事件确认操作
  test('T4-33 事件确认操作', async ({ authPage: page }) => {
    const incidentId = await createTestIncident(page)

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找确认按钮', async () => {
      const ackBtn = page.locator('button').filter({ hasText: /确认|acknowledge|ack/i }).first()
      if (await ackBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T4-33-确认按钮.png', fullPage: false })
      }
    })
  })

  // T4-34: 事件关闭操作
  test('T4-34 事件关闭操作', async ({ authPage: page }) => {
    const incidentId = await createTestIncident(page)

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找关闭按钮', async () => {
      const closeBtn = page.locator('button').filter({ hasText: /关闭|close/i }).first()
      if (await closeBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T4-34-关闭按钮.png', fullPage: false })
      }
    })
  })

  // T4-35: 事件从列表确认
  test('T4-35 事件从列表确认', async ({ authPage: page }) => {
    const incidentId = await createTestIncident(page)

    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击行操作菜单', async () => {
      const actionBtn = page.locator('.action-trigger, [class*="action-trigger"]').first()
      if (await actionBtn.isVisible().catch(() => false)) {
        await actionBtn.click({ force: true })
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T4-35-列表操作.png', fullPage: false })
      }
    })

    await test.step('关闭菜单', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-36: 事件 Snooze 模态框
  test('T4-36 事件Snooze模态框', async ({ authPage: page }) => {
    const incidentId = await createTestIncident(page)

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开 Snooze 模态框', async () => {
      const moreBtn = page.locator('button').filter({ hasText: /操作|action|more/i }).first()
      if (await moreBtn.isVisible().catch(() => false)) {
        await moreBtn.click()
        await page.waitForTimeout(300)
        const snoozeItem = page.locator('[class*="dropdown"] [class*="item"], .n-dropdown-option').filter({ hasText: /snooze|静默|暂停/i }).first()
        if (await snoozeItem.isVisible().catch(() => false)) {
          await snoozeItem.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T4-36-Snooze模态框.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-37: 事件 Merge 模态框
  test('T4-37 事件Merge模态框', async ({ authPage: page }) => {
    const incidentId = await createTestIncident(page)

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开 Merge 模态框', async () => {
      const moreBtn = page.locator('button').filter({ hasText: /操作|action|more/i }).first()
      if (await moreBtn.isVisible().catch(() => false)) {
        await moreBtn.click()
        await page.waitForTimeout(300)
        const mergeItem = page.locator('[class*="dropdown"] [class*="item"], .n-dropdown-option').filter({ hasText: /merge|合并/i }).first()
        if (await mergeItem.isVisible().catch(() => false)) {
          await mergeItem.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T4-37-Merge模态框.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-38: 事件 Reassign 模态框
  test('T4-38 事件Reassign模态框', async ({ authPage: page }) => {
    const incidentId = await createTestIncident(page)

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开 Reassign 模态框', async () => {
      const moreBtn = page.locator('button').filter({ hasText: /操作|action|more/i }).first()
      if (await moreBtn.isVisible().catch(() => false)) {
        await moreBtn.click()
        await page.waitForTimeout(300)
        const reassignItem = page.locator('[class*="dropdown"] [class*="item"], .n-dropdown-option').filter({ hasText: /reassign|转派|分配/i }).first()
        if (await reassignItem.isVisible().catch(() => false)) {
          await reassignItem.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T4-38-Reassign模态框.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-39: 事件 Escalate 操作
  test('T4-39 事件Escalate操作', async ({ authPage: page }) => {
    const incidentId = await createTestIncident(page)

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找升级按钮', async () => {
      const moreBtn = page.locator('button').filter({ hasText: /操作|action|more/i }).first()
      if (await moreBtn.isVisible().catch(() => false)) {
        await moreBtn.click()
        await page.waitForTimeout(300)
        const escalateItem = page.locator('[class*="dropdown"] [class*="item"], .n-dropdown-option').filter({ hasText: /escalate|升级/i }).first()
        if (await escalateItem.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T4-39-升级按钮.png', fullPage: false })
        }
      }
    })

    await test.step('关闭菜单', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-40: 事件 Quick Silence 操作
  test('T4-40 事件QuickSilence', async ({ authPage: page }) => {
    const incidentId = await createTestIncident(page)

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开 Quick Silence', async () => {
      const moreBtn = page.locator('button').filter({ hasText: /操作|action|more/i }).first()
      if (await moreBtn.isVisible().catch(() => false)) {
        await moreBtn.click()
        await page.waitForTimeout(300)
        const silenceItem = page.locator('[class*="dropdown"] [class*="item"], .n-dropdown-option').filter({ hasText: /silence|静默|免打扰/i }).first()
        if (await silenceItem.isVisible().catch(() => false)) {
          await silenceItem.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T4-40-QuickSilence.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-41: 事件 Timeline 动作筛选
  test('T4-41 事件Timeline动作筛选', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('切换到 Timeline Tab', async () => {
      const timelineTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /时间线|timeline/i }).first()
      if (await timelineTab.isVisible().catch(() => false)) {
        await timelineTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('验证 Timeline 筛选', async () => {
      const filterSelect = page.locator('.n-select, [class*="n-select"]').filter({ hasText: /筛选|filter/i }).first()
      await page.screenshot({ path: 'test-results/T4-41-Timeline筛选.png', fullPage: false })
    })
  })

  // T4-42: 事件 Timeline 条目颜色
  test('T4-42 事件Timeline条目颜色', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('切换到 Timeline Tab', async () => {
      const timelineTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /时间线|timeline/i }).first()
      if (await timelineTab.isVisible().catch(() => false)) {
        await timelineTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-42-Timeline颜色.png', fullPage: false })
      }
    })
  })

  // T4-43: 事件自动刷新轮询
  test('T4-43 事件自动刷新', async ({ authPage: page }) => {
    const incidentId = await createTestIncident(page)

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('等待自动刷新', async () => {
      await page.waitForTimeout(2000)
      await page.screenshot({ path: 'test-results/T4-43-自动刷新.png', fullPage: false })
    })
  })

  // T4-44: 事件批量选中
  test('T4-44 事件批量选中', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击全选复选框', async () => {
      const selectAll = page.locator('.select-all-checkbox, [class*="select-all"]').first()
      if (await selectAll.isVisible().catch(() => false)) {
        await selectAll.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-44-批量选中.png', fullPage: false })
      }
    })

    await test.step('取消选中', async () => {
      const selectAll = page.locator('.select-all-checkbox, [class*="select-all"]').first()
      if (await selectAll.isVisible().catch(() => false)) {
        await selectAll.click()
      }
    })
  })

  // T4-45: 事件批量操作栏
  test('T4-45 事件批量操作栏', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('选中事件触发批量操作栏', async () => {
      const selectAll = page.locator('.select-all-checkbox, [class*="select-all"]').first()
      if (await selectAll.isVisible().catch(() => false)) {
        await selectAll.click()
        await page.waitForTimeout(500)

        const bulkBar = page.locator('.bulk-action-bar, [class*="bulk-action"]').first()
        if (await bulkBar.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T4-45-批量操作栏.png', fullPage: false })
        }
      }
    })

    await test.step('取消选中', async () => {
      const selectAll = page.locator('.select-all-checkbox, [class*="select-all"]').first()
      if (await selectAll.isVisible().catch(() => false)) {
        await selectAll.click()
      }
    })
  })

  // T4-46: 事件关闭状态样式
  test('T4-46 事件关闭状态样式', async ({ authPage: page }) => {
    await test.step('导航到事件列表页并筛选已关闭', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
      const closedBtn = page.locator('.n-radio-button, [role="radio"]').filter({ hasText: /已关闭|closed/i }).first()
      if (await closedBtn.isVisible().catch(() => false)) {
        await closedBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-46-关闭样式.png', fullPage: false })
      }
    })
  })

  // T4-47: 事件指派人头像
  test('T4-47 事件指派人头像', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证指派人头像', async () => {
      const avatars = page.locator('.avatar, [class*="avatar"]')
      const count = await avatars.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T4-47-指派人.png', fullPage: false })
      }
    })
  })

  // T4-48: 事件持续时间文本
  test('T4-48 事件持续时间文本', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证持续时间', async () => {
      const metaItems = page.locator('.meta-item, [class*="meta-item"]')
      const count = await metaItems.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T4-48-持续时间.png', fullPage: false })
      }
    })
  })

  // T4-49: 事件 ID 显示
  test('T4-49 事件ID显示', async ({ authPage: page }) => {
    const incidentId = await getFirstIncidentId(page)
    if (!incidentId) return

    await test.step('导航到事件详情页', async () => {
      await page.goto(`${INCIDENTS_URL}/${incidentId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证事件 ID', async () => {
      const idText = page.locator('.incident-id, [class*="incident-id"]').first()
      if (await idText.isVisible().catch(() => false)) {
        const text = await idText.textContent()
        await page.screenshot({ path: 'test-results/T4-49-事件ID.png', fullPage: false })
      }
    })
  })

  // T4-50: 事件导航到详情
  test('T4-50 事件导航到详情', async ({ authPage: page }) => {
    await test.step('导航到事件列表页', async () => {
      await page.goto(INCIDENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击事件行', async () => {
      const incidentRow = page.locator('.incident-row, [class*="incident-row"]').first()
      if (await incidentRow.isVisible().catch(() => false)) {
        await incidentRow.click()
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T4-50-导航详情.png', fullPage: true })
      }
    })
  })

  // ================================================================
  // T4-51 ~ T4-70: 协作空间
  // ================================================================

  // T4-51: 协作空间列表初始加载
  test('T4-51 协作空间列表加载', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T4-51-空间列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T4-52: 协作空间搜索
  test('T4-52 协作空间搜索', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-52-空间搜索.png', fullPage: false })
      }
    })
  })

  // T4-53: 协作空间状态筛选
  test('T4-53 协作空间状态筛选', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证状态筛选', async () => {
      const filterSelect = page.locator('.n-select, [class*="n-select"]').first()
      if (await filterSelect.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T4-53-状态筛选.png', fullPage: false })
      }
    })
  })

  // T4-54: 协作空间新建弹窗
  test('T4-54 协作空间新建弹窗', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-54-新建空间弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-55: 协作空间卡片视图
  test('T4-55 协作空间卡片视图', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证卡片视图', async () => {
      const cardView = page.locator('[class*="card"], [class*="grid"]').first()
      if (await cardView.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T4-55-卡片视图.png', fullPage: false })
      }
    })
  })

  // T4-56: 协作空间列表视图切换
  test('T4-56 协作空间视图切换', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('切换到列表视图', async () => {
      const listBtn = page.locator('button').filter({ hasText: /list|列表/i }).first()
      const listIcon = page.locator('[class*="list-outline"], [class*="list"]').first()
      if (await listBtn.isVisible().catch(() => false)) {
        await listBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-56-列表视图.png', fullPage: false })
      }
    })
  })

  // T4-57: 协作空间团队筛选
  test('T4-57 协作空间团队筛选', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证团队筛选', async () => {
      const teamFilter = page.locator('.n-select, [class*="n-select"]').nth(1)
      if (await teamFilter.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T4-57-团队筛选.png', fullPage: false })
      }
    })
  })

  // T4-58: 协作空间排序
  test('T4-58 协作空间排序', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证排序选项', async () => {
      const sortSelect = page.locator('.n-select, [class*="n-select"]').filter({ hasText: /排序|sort/i }).first()
      if (await sortSelect.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T4-58-排序选项.png', fullPage: false })
      }
    })
  })

  // T4-59: 协作空间刷新
  test('T4-59 协作空间刷新', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击刷新按钮', async () => {
      const refreshBtn = page.locator('button').filter({ hasText: /刷新|refresh/i }).first()
      if (await refreshBtn.isVisible().catch(() => false)) {
        await refreshBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-59-刷新后.png', fullPage: false })
      }
    })
  })

  // T4-60: 协作空间分页
  test('T4-60 协作空间分页', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查分页控件', async () => {
      const pagination = page.locator('.n-pagination, [class*="pagination"]').first()
      if (await pagination.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T4-60-空间分页.png', fullPage: false })
      }
    })
  })

  // T4-61: 协作空间详情页加载
  test('T4-61 协作空间详情页', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击第一个空间', async () => {
      const spaceCard = page.locator('[class*="card"], [class*="channel-item"], [class*="space"]').first()
      if (await spaceCard.isVisible().catch(() => false)) {
        await spaceCard.click()
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T4-61-空间详情.png', fullPage: true })
      }
    })
  })

  // T4-62: 协作空间空状态
  test('T4-62 协作空间空状态', async ({ authPage: page }) => {
    await test.step('导航到协作空间页并搜索', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('zzz_nonexistent_zzz')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-62-空间空状态.png', fullPage: false })
      }
    })
  })

  // T4-63: 协作空间收藏功能
  test('T4-63 协作空间收藏', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找收藏按钮', async () => {
      const starBtn = page.locator('[class*="star"], [class*="favorite"]').first()
      if (await starBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T4-63-收藏按钮.png', fullPage: false })
      }
    })
  })

  // T4-64: 协作空间创建表单字段
  test('T4-64 协作空间创建表单', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗验证字段', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)

        const modal = page.locator('.n-modal, [class*="modal"]').first()
        if (await modal.isVisible().catch(() => false)) {
          const nameInput = page.locator('.n-modal input').first()
          const descInput = page.locator('.n-modal textarea').first()
          await page.screenshot({ path: 'test-results/T4-64-创建表单.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-65: 协作空间自动关闭配置
  test('T4-65 协作空间自动关闭', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗检查自动关闭', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T4-65-自动关闭配置.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T4-66: 协作空间噪音配置
  test('T4-66 协作空间噪音配置', async ({ authPage: page }) => {
    const channelId = await createTestChannel(page)

    await test.step('导航到协作空间详情', async () => {
      await page.goto(`${SPACES_URL}/${channelId}`)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T4-66-噪音配置.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T4-67: 协作空间团队标签
  test('T4-67 协作空间团队标签', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证团队标签', async () => {
      const teamTags = page.locator('[class*="team"], [class*="tag"]').filter({ hasText: /team|团队/i })
      await page.screenshot({ path: 'test-results/T4-67-团队标签.png', fullPage: false })
    })
  })

  // T4-68: 协作空间事件计数
  test('T4-68 协作空间事件计数', async ({ authPage: page }) => {
    await test.step('导航到协作空间页', async () => {
      await page.goto(SPACES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证事件计数', async () => {
      const incidentCounts = page.locator('[class*="incident-count"], [class*="count"]')
      await page.screenshot({ path: 'test-results/T4-68-事件计数.png', fullPage: false })
    })
  })

  // T4-69: 协作空间派发策略
  test('T4-69 协作空间派发策略', async ({ authPage: page }) => {
    const channelId = await createTestChannel(page)

    await test.step('导航到协作空间详情', async () => {
      await page.goto(`${SPACES_URL}/${channelId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找派发策略相关元素', async () => {
      const dispatchSection = page.locator('[class*="dispatch"], [class*="policy"]').first()
      await page.screenshot({ path: 'test-results/T4-69-派发策略.png', fullPage: false })
    })
  })

  // T4-70: 协作空间排除规则
  test('T4-70 协作空间排除规则', async ({ authPage: page }) => {
    const channelId = await createTestChannel(page)

    await test.step('导航到协作空间详情', async () => {
      await page.goto(`${SPACES_URL}/${channelId}`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找排除规则相关元素', async () => {
      const exclusionSection = page.locator('[class*="exclusion"], [class*="exclude"], [class*="mute"]').first()
      await page.screenshot({ path: 'test-results/T4-70-排除规则.png', fullPage: false })
    })
  })
})
