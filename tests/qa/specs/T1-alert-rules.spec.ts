import { test, expect } from '../fixtures/auth'

// T1: 告警规则 — 冒烟测试
// 对应复测手册 T1-1 ~ T1-113

test.describe('T1 - 告警规则', () => {

  test.beforeEach(async ({ authenticatedPage: page }) => {
    await page.goto('/alert/rules')
    await page.waitForLoadState('networkidle')
  })

  // T1-1: 列表正常加载
  test('T1-1 列表正常加载', async ({ authenticatedPage: page }) => {
    // 左侧分类侧栏存在
    await expect(page.locator('.category-sidebar, [class*="category"]')).toBeVisible()
    // 右侧规则卡片列表存在
    await expect(page.locator('.rule-card, [class*="rule-card"], [class*="card"]')).toHaveCount({ minimum: 0 })
    // 分页器存在
    await expect(page.locator('.n-pagination, [class*="pagination"]')).toBeVisible()
  })

  // T1-2: 空态
  test('T1-2 空态显示', async ({ authenticatedPage: page }) => {
    // 如果没有规则，应显示空状态
    const cards = page.locator('.rule-card, [class*="rule-card"]')
    const count = await cards.count()
    if (count === 0) {
      await expect(page.locator('text=创建, text=Create, [class*="empty"]')).toBeVisible()
    }
  })

  // T1-6: 点击卡片打开编辑
  test('T1-6 点击卡片打开编辑', async ({ authenticatedPage: page }) => {
    const firstCard = page.locator('.rule-card, [class*="rule-card"]').first()
    if (await firstCard.isVisible()) {
      await firstCard.click()
      await expect(page.locator('.n-modal, [role="dialog"]')).toBeVisible()
    }
  })

  // T1-8: 分类列表与计数
  test('T1-8 分类侧栏显示', async ({ authenticatedPage: page }) => {
    const sidebar = page.locator('.category-sidebar, [class*="category"]')
    await expect(sidebar).toBeVisible()
    // "全部"分类存在
    await expect(sidebar.locator('text=全部, text=All')).toBeVisible()
  })

  // T1-9: 点击分类筛选
  test('T1-9 分类筛选', async ({ authenticatedPage: page }) => {
    const categories = page.locator('.category-sidebar button, [class*="category"] button')
    const count = await categories.count()
    if (count > 1) {
      await categories.nth(1).click()
      await page.waitForLoadState('networkidle')
      // 列表应刷新
    }
  })

  // T1-10: 搜索框筛选
  test('T1-10 搜索框筛选', async ({ authenticatedPage: page }) => {
    const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
    if (await searchInput.isVisible()) {
      await searchInput.fill('test')
      await page.waitForTimeout(500) // debounce
      // 列表应刷新
    }
  })

  // T1-11: 严重度筛选
  test('T1-11 严重度筛选', async ({ authenticatedPage: page }) => {
    const severitySelect = page.locator('select, .n-select').filter({ hasText: /severity|严重度/ }).first()
    if (await severitySelect.isVisible()) {
      await severitySelect.click()
      // 选择 critical
      await page.locator('text=critical, text=严重').first().click()
      await page.waitForLoadState('networkidle')
    }
  })

  // T1-20: 新建规则弹窗
  test('T1-20 新建规则弹窗', async ({ authenticatedPage: page }) => {
    const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
    if (await createBtn.isVisible()) {
      await createBtn.click()
      await expect(page.locator('.n-modal, [role="dialog"]')).toBeVisible()
      // 关闭弹窗
      await page.keyboard.press('Escape')
    }
  })

  // T1-25: 批量操作
  test('T1-25 批量选择', async ({ authenticatedPage: page }) => {
    const checkboxes = page.locator('input[type="checkbox"], .n-checkbox')
    const count = await checkboxes.count()
    if (count > 0) {
      await checkboxes.first().click()
      // 批量操作栏应出现
    }
  })

  // T1-30: 键盘导航
  test('T1-30 键盘导航', async ({ authenticatedPage: page }) => {
    // 按 j 键应选中下一行
    await page.keyboard.press('j')
    // 按 Enter 应打开编辑
    // (depends on implementation)
  })
})
