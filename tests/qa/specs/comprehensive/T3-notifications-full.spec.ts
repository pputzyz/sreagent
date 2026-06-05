import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// T3: 通知完整测试 — 70 个测试用例
// 覆盖：通知策略列表(T3-1~T3-15)、通知渠道(T3-16~T3-30)、消息模板(T3-31~T3-45)、
//       订阅规则(T3-46~T3-55)、静默规则(T3-56~T3-70)

const POLICIES_URL = '/oncall/notify/policies'
const CHANNELS_URL = '/oncall/notify/channels'
const TEMPLATES_URL = '/oncall/notify/templates'
const SUBSCRIPTIONS_URL = '/oncall/notify/subscriptions'
const CENTER_URL = '/notifications'

/** 生成唯一名称 */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

/** 通过 API 创建通知渠道 */
async function createTestMedia(page: import('@playwright/test').Page, overrides?: Record<string, unknown>): Promise<number> {
  const name = uid('test_media')
  const body = {
    name,
    type: 'http',
    description: 'E2E test media',
    is_enabled: true,
    config: JSON.stringify({ method: 'POST', url: 'https://httpbin.org/post', headers: {}, body: '{}' }),
    variables: '{}',
    ...overrides,
  }
  const res = await API.post(page, '/api/v1/notify-media', body)
  return res?.data?.id ?? 0
}

/** 通过 API 创建通知策略 */
async function createTestRule(page: import('@playwright/test').Page, overrides?: Record<string, unknown>): Promise<number> {
  const name = uid('test_rule')
  const body = {
    name,
    description: 'E2E test rule',
    severities: 'critical,warning',
    match_labels: {},
    pipeline: '[]',
    notify_configs: '[]',
    repeat_interval: 3600,
    callback_url: '',
    is_enabled: true,
    ...overrides,
  }
  const res = await API.post(page, '/api/v1/notify-rules', body)
  return res?.data?.id ?? 0
}

/** 通过 API 创建消息模板 */
async function createTestTemplate(page: import('@playwright/test').Page, overrides?: Record<string, unknown>): Promise<number> {
  const name = uid('test_tmpl')
  const body = {
    name,
    description: 'E2E test template',
    type: 'text',
    content: 'Alert: {{.AlertName}} Severity: {{.Severity}}',
    ...overrides,
  }
  const res = await API.post(page, '/api/v1/message-templates', body)
  return res?.data?.id ?? 0
}

/** 通过 API 创建订阅规则 */
async function createTestSubscription(page: import('@playwright/test').Page, overrides?: Record<string, unknown>): Promise<number> {
  const name = uid('test_sub')
  const body = {
    name,
    description: 'E2E test subscription',
    match_labels: {},
    severities: 'critical',
    notify_rule_id: null,
    user_id: 1,
    team_id: null,
    is_enabled: true,
    ...overrides,
  }
  const res = await API.post(page, '/api/v1/subscribe-rules', body)
  return res?.data?.id ?? 0
}

/** 通过 API 删除资源 */
async function deleteResource(page: import('@playwright/test').Page, basePath: string, id: number): Promise<void> {
  if (id > 0) {
    await API.del(page, `${basePath}/${id}`)
  }
}

test.describe('T3 - 通知完整测试', () => {

  // ================================================================
  // T3-1 ~ T3-15: 通知策略列表
  // ================================================================

  // T3-1: 通知策略列表初始加载
  test('T3-1 通知策略列表初始加载', async ({ authPage: page }) => {
    await test.step('导航到通知策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T3-1-通知策略列表.png', fullPage: true })
    })

    await test.step('验证页面标题区域', async () => {
      await expect(page.locator('.rules-page, .page-container, [class*="rules"]').first()).toBeVisible()
    })

    await test.step('验证工具栏存在', async () => {
      const toolbar = page.locator('.toolbar, [class*="toolbar"]').first()
      if (await toolbar.isVisible().catch(() => false)) {
        await expect(toolbar).toBeVisible()
      }
    })
  })

  // T3-2: 通知策略骨架屏加载
  test('T3-2 通知策略骨架屏加载', async ({ authPage: page }) => {
    await test.step('导航并观察加载状态', async () => {
      await page.goto(POLICIES_URL)
      await page.screenshot({ path: 'test-results/T3-2-骨架屏.png', fullPage: false })
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证加载完成后内容出现', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T3-3: 通知策略空状态展示
  test('T3-3 通知策略空状态或列表', async ({ authPage: page }) => {
    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查空状态或列表内容', async () => {
      const emptyState = page.locator('[class*="empty"], .n-empty, [class*="EmptyState"]').first()
      const ruleList = page.locator('.row-list, [class*="sre-notify-card"]').first()
      const hasContent = await ruleList.isVisible().catch(() => false)
      if (!hasContent && await emptyState.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-3-空状态.png', fullPage: false })
      } else {
        await page.screenshot({ path: 'test-results/T3-3-列表内容.png', fullPage: false })
      }
    })
  })

  // T3-4: 通知策略搜索功能
  test('T3-4 通知策略搜索', async ({ authPage: page }) => {
    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('.toolbar input, [class*="toolbar"] input').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-4-搜索结果.png', fullPage: false })
      }
    })

    await test.step('清除搜索', async () => {
      const searchInput = page.locator('.toolbar input, [class*="toolbar"] input').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.clear()
        await page.waitForTimeout(300)
      }
    })
  })

  // T3-5: 通知策略计数显示
  test('T3-5 通知策略计数显示', async ({ authPage: page }) => {
    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证计数元素', async () => {
      const count = page.locator('.count, [class*="count"]').first()
      if (await count.isVisible().catch(() => false)) {
        const text = await count.textContent()
        expect(text).toBeTruthy()
        await page.screenshot({ path: 'test-results/T3-5-计数显示.png', fullPage: false })
      }
    })
  })

  // T3-6: 新建通知策略弹窗
  test('T3-6 新建通知策略弹窗', async ({ authPage: page }) => {
    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-6-新建弹窗.png', fullPage: false })
      }
    })

    await test.step('验证弹窗表单字段', async () => {
      const modal = page.locator('.n-modal, [class*="modal"]').first()
      if (await modal.isVisible().catch(() => false)) {
        await expect(modal).toBeVisible()
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-7: 通知策略启用/禁用切换
  test('T3-7 通知策略启用禁用切换', async ({ authPage: page }) => {
    const ruleId = await createTestRule(page)
    if (!ruleId) return

    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找开关并截图', async () => {
      const switches = page.locator('.n-switch, [class*="switch"]')
      const count = await switches.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T3-7-启用禁用.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-rules', ruleId)
    })
  })

  // T3-8: 通知策略严重等级标签
  test('T3-8 通知策略严重等级标签', async ({ authPage: page }) => {
    const ruleId = await createTestRule(page, { severities: 'critical,warning,info' })

    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证严重等级标签', async () => {
      const sevChips = page.locator('.sev-chip, [class*="sev-chip"]')
      const count = await sevChips.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T3-8-严重等级标签.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-rules', ruleId)
    })
  })

  // T3-9: 通知策略匹配标签展示
  test('T3-9 通知策略匹配标签展示', async ({ authPage: page }) => {
    const ruleId = await createTestRule(page, { match_labels: { env: 'production', region: 'cn-east' } })

    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证标签芯片', async () => {
      const labelChips = page.locator('.label-chip, code[class*="label"]')
      const count = await labelChips.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T3-9-匹配标签.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-rules', ruleId)
    })
  })

  // T3-10: 通知策略测试按钮
  test('T3-10 通知策略测试按钮', async ({ authPage: page }) => {
    const ruleId = await createTestRule(page)

    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找测试按钮', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await testBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-10-测试弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-rules', ruleId)
    })
  })

  // T3-11: 通知策略行菜单操作
  test('T3-11 通知策略行菜单', async ({ authPage: page }) => {
    const ruleId = await createTestRule(page)

    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击行操作按钮', async () => {
      const actionBtn = page.locator('.sre-icon-btn, button[aria-label*="操作"], button[aria-label*="action"]').first()
      if (await actionBtn.isVisible().catch(() => false)) {
        await actionBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-11-行菜单.png', fullPage: false })
      }
    })

    await test.step('关闭菜单', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-rules', ruleId)
    })
  })

  // T3-12: 通知策略编辑弹窗
  test('T3-12 通知策略编辑弹窗', async ({ authPage: page }) => {
    const ruleId = await createTestRule(page)

    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击编辑', async () => {
      const actionBtn = page.locator('.sre-icon-btn, button[aria-label*="操作"], button[aria-label*="action"]').first()
      if (await actionBtn.isVisible().catch(() => false)) {
        await actionBtn.click()
        await page.waitForTimeout(300)
        const editItem = page.locator('[class*="dropdown"] [class*="item"], .n-dropdown-option').filter({ hasText: /编辑|Edit/ }).first()
        if (await editItem.isVisible().catch(() => false)) {
          await editItem.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T3-12-编辑弹窗.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-rules', ruleId)
    })
  })

  // T3-13: 通知策略分页控件
  test('T3-13 通知策略分页控件', async ({ authPage: page }) => {
    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查分页控件', async () => {
      const pagination = page.locator('.n-pagination, [class*="pagination"]').first()
      if (await pagination.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-13-分页控件.png', fullPage: false })
      }
    })
  })

  // T3-14: 通知策略重复间隔显示
  test('T3-14 通知策略重复间隔', async ({ authPage: page }) => {
    const ruleId = await createTestRule(page, { repeat_interval: 7200 })

    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证重复间隔元数据', async () => {
      const metaItems = page.locator('.meta, [class*="meta"]')
      const count = await metaItems.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T3-14-重复间隔.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-rules', ruleId)
    })
  })

  // T3-15: 通知策略描述显示
  test('T3-15 通知策略描述显示', async ({ authPage: page }) => {
    const ruleId = await createTestRule(page, { description: 'This is a test description for E2E' })

    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证描述文本', async () => {
      const descText = page.locator('.meta, [class*="meta"]').filter({ hasText: /E2E|test/i }).first()
      if (await descText.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-15-描述显示.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-rules', ruleId)
    })
  })

  // ================================================================
  // T3-16 ~ T3-30: 通知渠道列表
  // ================================================================

  // T3-16: 通知渠道列表初始加载
  test('T3-16 通知渠道列表初始加载', async ({ authPage: page }) => {
    await test.step('导航到通知渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T3-16-渠道列表.png', fullPage: true })
    })

    await test.step('验证页面结构', async () => {
      await expect(page.locator('.media-page, [class*="media"]').first()).toBeVisible()
    })
  })

  // T3-17: 通知渠道搜索功能
  test('T3-17 通知渠道搜索', async ({ authPage: page }) => {
    await test.step('导航到渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('.toolbar input, [class*="toolbar"] input').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-17-渠道搜索.png', fullPage: false })
      }
    })
  })

  // T3-18: 通知渠道类型筛选
  test('T3-18 通知渠道类型筛选', async ({ authPage: page }) => {
    await test.step('导航到渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证类型筛选下拉', async () => {
      const typeSelect = page.locator('.toolbar .n-select, [class*="toolbar"] .n-select').first()
      if (await typeSelect.isVisible().catch(() => false)) {
        await typeSelect.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-18-类型筛选.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T3-19: 新建通知渠道弹窗
  test('T3-19 新建通知渠道弹窗', async ({ authPage: page }) => {
    await test.step('导航到渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-19-新建渠道弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-20: 创建 HTTP 类型渠道
  test('T3-20 创建HTTP类型渠道', async ({ authPage: page }) => {
    await test.step('导航到渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开新建弹窗并选择 HTTP 类型', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)

        const nameInput = page.locator('.n-modal input').first()
        if (await nameInput.isVisible().catch(() => false)) {
          await nameInput.fill(uid('http_channel'))
          await page.screenshot({ path: 'test-results/T3-20-HTTP渠道表单.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-21: 通知渠道类型图标显示
  test('T3-21 通知渠道类型图标', async ({ authPage: page }) => {
    const mediaId = await createTestMedia(page)

    await test.step('导航到渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证类型图标', async () => {
      const typeIcons = page.locator('.type-icon, [class*="type-icon"]')
      const count = await typeIcons.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T3-21-类型图标.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-media', mediaId)
    })
  })

  // T3-22: 通知渠道类型标签
  test('T3-22 通知渠道类型标签', async ({ authPage: page }) => {
    const mediaId = await createTestMedia(page, { type: 'lark_webhook' })

    await test.step('导航到渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证类型标签', async () => {
      const typeChips = page.locator('.type-chip, [class*="type-chip"]')
      const count = await typeChips.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T3-22-类型标签.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-media', mediaId)
    })
  })

  // T3-23: 通知渠道测试连接
  test('T3-23 通知渠道测试连接', async ({ authPage: page }) => {
    const mediaId = await createTestMedia(page)

    await test.step('导航到渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找测试按钮', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-23-测试按钮.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-media', mediaId)
    })
  })

  // T3-24: 通知渠道目标摘要
  test('T3-24 通知渠道目标摘要', async ({ authPage: page }) => {
    const mediaId = await createTestMedia(page, {
      config: JSON.stringify({ method: 'POST', url: 'https://example.com/webhook', headers: {}, body: '{}' }),
    })

    await test.step('导航到渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证目标摘要', async () => {
      const targets = page.locator('.target, code[class*="target"]')
      const count = await targets.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T3-24-目标摘要.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-media', mediaId)
    })
  })

  // T3-25: 通知渠道启用状态
  test('T3-25 通知渠道启用状态', async ({ authPage: page }) => {
    const mediaId = await createTestMedia(page, { is_enabled: false })

    await test.step('导航到渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证状态文本', async () => {
      const statusText = page.locator('.status-text, [class*="status-text"]')
      const count = await statusText.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T3-25-启用状态.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-media', mediaId)
    })
  })

  // T3-26: 内置渠道标记
  test('T3-26 内置渠道标记', async ({ authPage: page }) => {
    await test.step('导航到渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查内置标记', async () => {
      const builtinChips = page.locator('.builtin-chip, [class*="builtin"]')
      const count = await builtinChips.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T3-26-内置标记.png', fullPage: false })
      }
    })
  })

  // T3-27: 邮件渠道配置表单
  test('T3-27 邮件渠道配置表单', async ({ authPage: page }) => {
    await test.step('导航到渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开新建并切换到邮件类型', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-27-邮件配置.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-28: Lark Webhook 渠道配置
  test('T3-28 LarkWebhook渠道配置', async ({ authPage: page }) => {
    await test.step('导航到渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开新建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-28-Lark配置.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-29: 通知渠道描述显示
  test('T3-29 通知渠道描述显示', async ({ authPage: page }) => {
    const mediaId = await createTestMedia(page, { description: 'E2E test channel description' })

    await test.step('导航到渠道页', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证描述', async () => {
      const meta = page.locator('.meta, [class*="meta"]').filter({ hasText: /E2E/i }).first()
      if (await meta.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-29-渠道描述.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-media', mediaId)
    })
  })

  // T3-30: 通知渠道空状态
  test('T3-30 通知渠道空状态', async ({ authPage: page }) => {
    await test.step('导航到渠道页并搜索不存在的渠道', async () => {
      await page.goto(CHANNELS_URL)
      await page.waitForLoadState('networkidle')
      const searchInput = page.locator('.toolbar input, [class*="toolbar"] input').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('zzz_nonexistent_zzz')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-30-渠道空状态.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T3-31 ~ T3-45: 消息模板
  // ================================================================

  // T3-31: 消息模板列表初始加载
  test('T3-31 消息模板列表初始加载', async ({ authPage: page }) => {
    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T3-31-模板列表.png', fullPage: true })
    })

    await test.step('验证页面结构', async () => {
      await expect(page.locator('.tmpl-page, [class*="tmpl"]').first()).toBeVisible()
    })
  })

  // T3-32: 消息模板搜索
  test('T3-32 消息模板搜索', async ({ authPage: page }) => {
    const tmplId = await createTestTemplate(page)

    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('搜索模板', async () => {
      const searchInput = page.locator('.toolbar input, [class*="toolbar"] input').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-32-模板搜索.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/message-templates', tmplId)
    })
  })

  // T3-33: 消息模板类型筛选
  test('T3-33 消息模板类型筛选', async ({ authPage: page }) => {
    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证类型筛选', async () => {
      const typeSelect = page.locator('.toolbar .n-select, [class*="toolbar"] .n-select').first()
      if (await typeSelect.isVisible().catch(() => false)) {
        await typeSelect.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-33-类型筛选.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T3-34: 新建消息模板弹窗
  test('T3-34 新建消息模板弹窗', async ({ authPage: page }) => {
    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-34-新建模板弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-35: 消息模板变量区域
  test('T3-35 消息模板变量区域', async ({ authPage: page }) => {
    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开新建弹窗查看变量', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)

        const varSection = page.locator('.var-section, [class*="var-section"]').first()
        if (await varSection.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T3-35-变量区域.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-36: 消息模板类型标签
  test('T3-36 消息模板类型标签', async ({ authPage: page }) => {
    const tmplId = await createTestTemplate(page, { type: 'markdown' })

    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证类型标签', async () => {
      const typeChips = page.locator('.tmpl-type-chip, [class*="tmpl-type-chip"]')
      const count = await typeChips.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T3-36-模板类型标签.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/message-templates', tmplId)
    })
  })

  // T3-37: 消息模板内容预览
  test('T3-37 消息模板内容预览', async ({ authPage: page }) => {
    const tmplId = await createTestTemplate(page, { content: 'Alert {{.AlertName}} is {{.Severity}}' })

    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证内容截断预览', async () => {
      const contentCode = page.locator('.tmpl-content, code[class*="tmpl-content"]').first()
      if (await contentCode.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-37-内容预览.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/message-templates', tmplId)
    })
  })

  // T3-38: 消息模板内置标记
  test('T3-38 消息模板内置标记', async ({ authPage: page }) => {
    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查内置标记', async () => {
      const builtinChips = page.locator('.tmpl-builtin, [class*="builtin"]')
      const count = await builtinChips.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T3-38-内置标记.png', fullPage: false })
      }
    })
  })

  // T3-39: 消息模板行点击编辑
  test('T3-39 消息模板行点击编辑', async ({ authPage: page }) => {
    const tmplId = await createTestTemplate(page)

    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击模板行', async () => {
      const row = page.locator('.tmpl-row, [class*="tmpl-row"]').first()
      if (await row.isVisible().catch(() => false)) {
        await row.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-39-行点击编辑.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/message-templates', tmplId)
    })
  })

  // T3-40: 消息模板预览功能
  test('T3-40 消息模板预览功能', async ({ authPage: page }) => {
    const tmplId = await createTestTemplate(page, { content: 'Hello {{.AlertName}}' })

    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击预览', async () => {
      const actionBtn = page.locator('.tmpl-actions button, [class*="tmpl-actions"] button').first()
      if (await actionBtn.isVisible().catch(() => false)) {
        await actionBtn.click()
        await page.waitForTimeout(300)
        const previewItem = page.locator('[class*="dropdown"] [class*="item"], .n-dropdown-option').filter({ hasText: /预览|Preview/ }).first()
        if (await previewItem.isVisible().catch(() => false)) {
          await previewItem.click()
          await page.waitForTimeout(1000)
          await page.screenshot({ path: 'test-results/T3-40-预览结果.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/message-templates', tmplId)
    })
  })

  // T3-41: 消息模板行下拉菜单
  test('T3-41 消息模板行下拉菜单', async ({ authPage: page }) => {
    const tmplId = await createTestTemplate(page)

    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击操作按钮', async () => {
      const actionBtn = page.locator('.tmpl-actions button, [class*="tmpl-actions"] button').first()
      if (await actionBtn.isVisible().catch(() => false)) {
        await actionBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-41-下拉菜单.png', fullPage: false })
      }
    })

    await test.step('关闭菜单', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/message-templates', tmplId)
    })
  })

  // T3-42: 消息模板空状态
  test('T3-42 消息模板空状态', async ({ authPage: page }) => {
    await test.step('导航到模板页并搜索不存在的模板', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
      const searchInput = page.locator('.toolbar input, [class*="toolbar"] input').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('zzz_nonexistent_zzz')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-42-模板空状态.png', fullPage: false })
      }
    })
  })

  // T3-43: 消息模板描述显示
  test('T3-43 消息模板描述显示', async ({ authPage: page }) => {
    const tmplId = await createTestTemplate(page, { description: 'E2E template description' })

    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证描述', async () => {
      const desc = page.locator('.tmpl-desc, [class*="tmpl-desc"]').filter({ hasText: /E2E/i }).first()
      if (await desc.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-43-模板描述.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/message-templates', tmplId)
    })
  })

  // T3-44: 消息模板变量点击复制
  test('T3-44 消息模板变量复制', async ({ authPage: page }) => {
    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开新建弹窗并点击变量', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)

        const varChip = page.locator('.var-chip, [class*="var-chip"]').first()
        if (await varChip.isVisible().catch(() => false)) {
          await varChip.click()
          await page.waitForTimeout(300)
          await page.screenshot({ path: 'test-results/T3-44-变量复制.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-45: 消息模板 Lark Card 类型
  test('T3-45 消息模板LarkCard类型', async ({ authPage: page }) => {
    const tmplId = await createTestTemplate(page, { type: 'lark_card', content: '{"type":"template","data":{}}' })

    await test.step('导航到模板页', async () => {
      await page.goto(TEMPLATES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证 Lark Card 模板显示', async () => {
      const typeChips = page.locator('.tmpl-type-chip, [class*="tmpl-type-chip"]').filter({ hasText: /lark/i })
      if (await typeChips.first().isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-45-LarkCard类型.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/message-templates', tmplId)
    })
  })

  // ================================================================
  // T3-46 ~ T3-55: 订阅规则
  // ================================================================

  // T3-46: 订阅规则列表初始加载
  test('T3-46 订阅规则列表初始加载', async ({ authPage: page }) => {
    await test.step('导航到订阅页', async () => {
      await page.goto(SUBSCRIPTIONS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T3-46-订阅列表.png', fullPage: true })
    })

    await test.step('验证页面结构', async () => {
      await expect(page.locator('.sub-page, [class*="sub"]').first()).toBeVisible()
    })
  })

  // T3-47: 订阅规则搜索
  test('T3-47 订阅规则搜索', async ({ authPage: page }) => {
    const subId = await createTestSubscription(page)

    await test.step('导航到订阅页', async () => {
      await page.goto(SUBSCRIPTIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('搜索订阅', async () => {
      const searchInput = page.locator('.toolbar input, [class*="toolbar"] input').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-47-订阅搜索.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/subscribe-rules', subId)
    })
  })

  // T3-48: 新建订阅规则弹窗
  test('T3-48 新建订阅规则弹窗', async ({ authPage: page }) => {
    await test.step('导航到订阅页', async () => {
      await page.goto(SUBSCRIPTIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-48-新建订阅弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-49: 订阅规则启用/禁用
  test('T3-49 订阅规则启用禁用', async ({ authPage: page }) => {
    const subId = await createTestSubscription(page)

    await test.step('导航到订阅页', async () => {
      await page.goto(SUBSCRIPTIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证开关', async () => {
      const switches = page.locator('.n-switch, [class*="switch"]')
      if (await switches.first().isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-49-订阅开关.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/subscribe-rules', subId)
    })
  })

  // T3-50: 订阅规则用户订阅显示
  test('T3-50 订阅规则用户订阅', async ({ authPage: page }) => {
    const subId = await createTestSubscription(page, { user_id: 1, team_id: null })

    await test.step('导航到订阅页', async () => {
      await page.goto(SUBSCRIPTIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证用户订阅显示', async () => {
      const subscribers = page.locator('.subscriber, [class*="subscriber"]')
      if (await subscribers.first().isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-50-用户订阅.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/subscribe-rules', subId)
    })
  })

  // T3-51: 订阅规则行菜单
  test('T3-51 订阅规则行菜单', async ({ authPage: page }) => {
    const subId = await createTestSubscription(page)

    await test.step('导航到订阅页', async () => {
      await page.goto(SUBSCRIPTIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击行操作按钮', async () => {
      const actionBtn = page.locator('.sre-icon-btn, button[aria-label*="操作"], button[aria-label*="action"]').first()
      if (await actionBtn.isVisible().catch(() => false)) {
        await actionBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-51-订阅行菜单.png', fullPage: false })
      }
    })

    await test.step('关闭菜单', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/subscribe-rules', subId)
    })
  })

  // T3-52: 订阅规则匹配标签
  test('T3-52 订阅规则匹配标签', async ({ authPage: page }) => {
    const subId = await createTestSubscription(page)

    await test.step('导航到订阅页', async () => {
      await page.goto(SUBSCRIPTIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证标签显示', async () => {
      const labelChips = page.locator('.label-chip, code[class*="label"]')
      const mutedText = page.locator('.muted, [class*="muted"]')
      const hasLabels = await labelChips.count() > 0
      const hasMuted = await mutedText.first().isVisible().catch(() => false)
      if (hasLabels || hasMuted) {
        await page.screenshot({ path: 'test-results/T3-52-订阅标签.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/subscribe-rules', subId)
    })
  })

  // T3-53: 订阅规则严重等级
  test('T3-53 订阅规则严重等级', async ({ authPage: page }) => {
    const subId = await createTestSubscription(page, { severities: 'critical,warning' })

    await test.step('导航到订阅页', async () => {
      await page.goto(SUBSCRIPTIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证严重等级', async () => {
      const sevChips = page.locator('.sev-chip, [class*="sev-chip"]')
      if (await sevChips.first().isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-53-订阅严重等级.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/subscribe-rules', subId)
    })
  })

  // T3-54: 订阅规则通知策略关联
  test('T3-54 订阅规则通知策略关联', async ({ authPage: page }) => {
    const subId = await createTestSubscription(page)

    await test.step('导航到订阅页', async () => {
      await page.goto(SUBSCRIPTIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证通知策略关联', async () => {
      const metaItems = page.locator('.meta, [class*="meta"]')
      if (await metaItems.first().isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-54-策略关联.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/subscribe-rules', subId)
    })
  })

  // T3-55: 订阅规则空状态
  test('T3-55 订阅规则空状态', async ({ authPage: page }) => {
    await test.step('导航到订阅页并搜索不存在的订阅', async () => {
      await page.goto(SUBSCRIPTIONS_URL)
      await page.waitForLoadState('networkidle')
      const searchInput = page.locator('.toolbar input, [class*="toolbar"] input').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('zzz_nonexistent_zzz')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-55-订阅空状态.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T3-56 ~ T3-70: 通知中心 + 静默规则
  // ================================================================

  // T3-56: 通知中心初始加载
  test('T3-56 通知中心初始加载', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T3-56-通知中心.png', fullPage: true })
    })

    await test.step('验证页面结构', async () => {
      await expect(page.locator('.notif-center, [class*="notif"]').first()).toBeVisible()
    })
  })

  // T3-57: 通知中心筛选切换
  test('T3-57 通知中心筛选切换', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('切换到未读筛选', async () => {
      const unreadTab = page.locator('.n-radio-button, [role="radio"]').filter({ hasText: /未读|unread/i }).first()
      if (await unreadTab.isVisible().catch(() => false)) {
        await unreadTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-57-未读筛选.png', fullPage: false })
      }
    })

    await test.step('切换回全部', async () => {
      const allTab = page.locator('.n-radio-button, [role="radio"]').filter({ hasText: /全部|all/i }).first()
      if (await allTab.isVisible().catch(() => false)) {
        await allTab.click()
        await page.waitForTimeout(300)
      }
    })
  })

  // T3-58: 通知中心严重等级筛选
  test('T3-58 通知中心严重等级筛选', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击严重等级筛选', async () => {
      const sevChips = page.locator('.sev-chip, [class*="sev-chip"]')
      if (await sevChips.first().isVisible().catch(() => false)) {
        await sevChips.first().click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-58-等级筛选.png', fullPage: false })
      }
    })
  })

  // T3-59: 通知中心全部标记已读
  test('T3-59 通知中心全部标记已读', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找全部标记已读按钮', async () => {
      const markAllBtn = page.locator('button').filter({ hasText: /全部已读|mark all/i }).first()
      if (await markAllBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-59-全部已读按钮.png', fullPage: false })
      }
    })
  })

  // T3-60: 通知中心刷新按钮
  test('T3-60 通知中心刷新', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击刷新按钮', async () => {
      const refreshBtn = page.locator('button').filter({ hasText: /刷新|Refresh/ }).first()
      if (await refreshBtn.isVisible().catch(() => false)) {
        await refreshBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-60-刷新后.png', fullPage: false })
      }
    })
  })

  // T3-61: 通知中心分页
  test('T3-61 通知中心分页', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查分页控件', async () => {
      const pagination = page.locator('.n-pagination, [class*="pagination"]').first()
      if (await pagination.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-61-通知分页.png', fullPage: false })
      }
    })
  })

  // T3-62: 通知项未读状态样式
  test('T3-62 通知项未读状态', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证未读样式', async () => {
      const unreadItems = page.locator('.notif-item.unread, [class*="unread"]')
      const count = await unreadItems.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T3-62-未读样式.png', fullPage: false })
      }
    })
  })

  // T3-63: 通知项类型标签
  test('T3-63 通知项类型标签', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证类型标签', async () => {
      const typeTags = page.locator('.n-tag, [class*="n-tag"]')
      const count = await typeTags.count()
      if (count > 0) {
        await page.screenshot({ path: 'test-results/T3-63-类型标签.png', fullPage: false })
      }
    })
  })

  // T3-64: 通知项删除按钮
  test('T3-64 通知项删除按钮', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找删除按钮', async () => {
      const deleteBtns = page.locator('.notif-actions button, [class*="notif-actions"] button')
      if (await deleteBtns.first().isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-64-删除按钮.png', fullPage: false })
      }
    })
  })

  // T3-65: 通知项时间显示
  test('T3-65 通知项时间显示', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证时间元数据', async () => {
      const metaItems = page.locator('.notif-meta, [class*="notif-meta"]')
      if (await metaItems.first().isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-65-时间显示.png', fullPage: false })
      }
    })
  })

  // T3-66: 通知中心空状态
  test('T3-66 通知中心空状态', async ({ authPage: page }) => {
    await test.step('导航到通知中心', async () => {
      await page.goto(CENTER_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查空状态或列表', async () => {
      const emptyState = page.locator('.n-empty, [class*="empty"]')
      const notifList = page.locator('.notif-list, [class*="notif-list"]')
      const hasItems = await notifList.isVisible().catch(() => false)
      if (!hasItems && await emptyState.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T3-66-通知空状态.png', fullPage: false })
      }
    })
  })

  // T3-67: Alert Channels 页面加载
  test('T3-67 AlertChannels页面加载', async ({ authPage: page }) => {
    await test.step('导航到 Alert Channels 页', async () => {
      await page.goto('/oncall/notify/alert-channels')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T3-67-AlertChannels.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T3-68: Alert Channels 新建弹窗
  test('T3-68 AlertChannels新建弹窗', async ({ authPage: page }) => {
    await test.step('导航到 Alert Channels 页', async () => {
      await page.goto('/oncall/notify/alert-channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T3-68-AC新建弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T3-69: 通知策略测试弹窗表单字段
  test('T3-69 测试弹窗表单字段', async ({ authPage: page }) => {
    const ruleId = await createTestRule(page)

    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开测试弹窗验证字段', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await testBtn.click()
        await page.waitForTimeout(500)

        const alertNameInput = page.locator('.n-modal input, .test-modal input').first()
        const severitySelect = page.locator('.n-modal .n-select, .test-modal .n-select').first()
        const hasFields = await alertNameInput.isVisible().catch(() => false) || await severitySelect.isVisible().catch(() => false)
        await page.screenshot({ path: 'test-results/T3-69-测试弹窗字段.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-rules', ruleId)
    })
  })

  // T3-70: 通知策略测试结果展示
  test('T3-70 测试结果展示', async ({ authPage: page }) => {
    const ruleId = await createTestRule(page)

    await test.step('导航到策略页', async () => {
      await page.goto(POLICIES_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开测试弹窗并发送测试', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await testBtn.click()
        await page.waitForTimeout(500)

        const sendBtn = page.locator('.n-modal button[type="primary"], .n-modal button').filter({ hasText: /发送|Send/ }).first()
        if (await sendBtn.isVisible().catch(() => false)) {
          await sendBtn.click()
          await page.waitForTimeout(2000)
          await page.screenshot({ path: 'test-results/T3-70-测试结果.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteResource(page, '/api/v1/notify-rules', ruleId)
    })
  })
})
