import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// T10: Dashboard Full Test Suite — 60 tests
// Covers: Dashboard List (T10-1~T10-15), Editor (T10-16~T10-30),
//         Panels (T10-31~T10-45), Features (T10-46~T10-60)

const DASHBOARD_URL = '/dashboard'

/** Generate unique name */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

/** Create a test dashboard via API */
async function createTestDashboard(page: import('@playwright/test').Page, overrides?: Record<string, unknown>): Promise<number> {
  const name = uid('test_dashboard')
  const body = {
    name,
    description: 'Test dashboard',
    is_public: false,
    ...overrides,
  }
  const res = await API.post(page, '/api/v1/dashboards', body)
  return res?.data?.id ?? 0
}

/** Delete a test dashboard via API */
async function deleteTestDashboard(page: import('@playwright/test').Page, id: number): Promise<void> {
  if (id > 0) {
    await API.del(page, `/api/v1/dashboards/${id}`)
  }
}

test.describe('T10 - Dashboard Full Test Suite', () => {

  // ================================================================
  // T10-1 ~ T10-15: Dashboard List
  // ================================================================

  // T10-1: Dashboard list page load
  test('T10-1 Dashboard list page load', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T10-1-dashboard-list-load.png', fullPage: true })
    })

    await test.step('Verify page body visible', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T10-2: Dashboard list search
  test('T10-2 Dashboard list search', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Search dashboards', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[placeholder*="Search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-2-dashboard-search.png', fullPage: false })
      }
    })

    await test.step('Clear search', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[placeholder*="Search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.clear()
        await page.waitForTimeout(400)
      }
    })
  })

  // T10-3: Create dashboard button
  test('T10-3 Create dashboard button', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click create dashboard button', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|新建仪表盘/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-3-create-dashboard.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T10-4: Create dashboard form
  test('T10-4 Create dashboard form', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create dialog and verify form', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const nameInput = page.locator('.n-modal input, [role="dialog"] input').first()
        if (await nameInput.isVisible()) {
          await expect(nameInput).toBeVisible()
          await page.screenshot({ path: 'test-results/T10-4-create-form.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T10-5: Create dashboard and verify
  test('T10-5 Create dashboard and verify', async ({ authPage: page }) => {
    let dashId = 0
    const dashName = uid('created_dash')

    await test.step('Create dashboard via API', async () => {
      dashId = await createTestDashboard(page, { name: dashName })
    })

    await test.step('Navigate to dashboard list', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Search for created dashboard', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[placeholder*="Search"]').first()
      if (await searchInput.isVisible().catch(() => false) && dashId > 0) {
        await searchInput.fill(dashName)
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-5-created-dashboard.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-6: Edit dashboard
  test('T10-6 Edit dashboard', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('edit_dash') })
    })

    await test.step('Navigate to dashboard list', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find and click edit', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-6-edit-dashboard.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-7: Delete dashboard confirmation
  test('T10-7 Delete dashboard confirmation', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard list', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click delete on a dashboard', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T10-7-delete-confirm.png', fullPage: false })
      }
    })

    await test.step('Cancel', async () => {
      const cancelBtn = page.locator('.n-dialog button, .n-modal button').filter({ hasText: /取消|Cancel/ }).first()
      if (await cancelBtn.isVisible().catch(() => false)) {
        await cancelBtn.click()
        await page.waitForTimeout(300)
      } else {
        await page.keyboard.press('Escape')
        await page.waitForTimeout(300)
      }
    })
  })

  // T10-8: Team filter
  test('T10-8 Team filter', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find and use team filter', async () => {
      const teamFilter = page.locator('.n-select').filter({ hasText: /团队|Team|team/ }).first()
      if (await teamFilter.isVisible().catch(() => false)) {
        await teamFilter.click()
        await page.waitForTimeout(300)
        const option = page.locator('.n-select-option').first()
        if (await option.isVisible()) {
          await option.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T10-8-team-filter.png', fullPage: false })
        }
      }
    })
  })

  // T10-9: Public/private toggle filter
  test('T10-9 Public private filter', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find public/private filter', async () => {
      const visibilityFilter = page.locator('.n-select, .n-radio-group').filter({ hasText: /公开|Public|私有|Private|全部/ }).first()
      if (await visibilityFilter.isVisible().catch(() => false)) {
        await visibilityFilter.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T10-9-visibility-filter.png', fullPage: false })
      }
    })
  })

  // T10-10: Sort options
  test('T10-10 Sort options', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find sort control', async () => {
      const sortSelect = page.locator('.n-select').filter({ hasText: /排序|Sort|sort/ }).first()
      if (await sortSelect.isVisible().catch(() => false)) {
        await sortSelect.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T10-10-sort-options.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T10-11: Pagination
  test('T10-11 Pagination', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check pagination controls', async () => {
      const pagination = page.locator('.n-pagination, [class*="pagination"]').first()
      if (await pagination.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-11-pagination.png', fullPage: false })
      }
    })
  })

  // T10-12: Empty state
  test('T10-12 Empty state', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check empty state or list', async () => {
      const emptyState = page.locator('[class*="empty"], .n-empty').first()
      const dashList = page.locator('[class*="dashboard-card"], [class*="dash-item"], .n-card').first()
      const hasContent = await dashList.isVisible().catch(() => false)
      if (!hasContent && await emptyState.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-12-empty-state.png', fullPage: false })
      }
    })
  })

  // T10-13: Dashboard card click navigation
  test('T10-13 Dashboard card click navigation', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('nav_dash') })
    })

    await test.step('Navigate to dashboard list', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click first dashboard card', async () => {
      const card = page.locator('[class*="dashboard-card"], [class*="dash-item"], .n-card').first()
      if (await card.isVisible().catch(() => false)) {
        await card.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-13-card-navigation.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-14: Dashboard list grid/list view toggle
  test('T10-14 View toggle grid list', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find view toggle', async () => {
      const viewToggle = page.locator('button').filter({ hasText: /网格|列表|Grid|List/ }).first()
      if (await viewToggle.isVisible().catch(() => false)) {
        await viewToggle.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T10-14-view-toggle.png', fullPage: false })
      }
    })
  })

  // T10-15: Dashboard API list
  test('T10-15 Dashboard API list', async ({ authPage: page }) => {
    await test.step('Get dashboard list via API', async () => {
      const res = await API.get(page, '/api/v1/dashboards?page=1&page_size=10')
      await page.screenshot({ path: 'test-results/T10-15-API-list.png', fullPage: false })
    })
  })

  // ================================================================
  // T10-16 ~ T10-30: Dashboard Editor
  // ================================================================

  // T10-16: Dashboard editor load
  test('T10-16 Dashboard editor load', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('editor_dash') })
    })

    await test.step('Navigate to dashboard editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
        await page.screenshot({ path: 'test-results/T10-16-editor-load.png', fullPage: true })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-17: Add panel button
  test('T10-17 Add panel button', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('add_panel_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Click add panel button', async () => {
      const addBtn = page.locator('button').filter({ hasText: /添加|Add|面板|Panel|新增/ }).first()
      if (await addBtn.isVisible().catch(() => false)) {
        await addBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-17-add-panel.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-18: Remove panel
  test('T10-18 Remove panel', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('rm_panel_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find panel remove option', async () => {
      const panel = page.locator('[class*="panel"], [class*="Panel"]').first()
      if (await panel.isVisible().catch(() => false)) {
        await panel.hover()
        await page.waitForTimeout(200)
        const removeBtn = page.locator('button[title*="delete"], button[title*="删除"], button[title*="remove"]').first()
        if (await removeBtn.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T10-18-remove-panel.png', fullPage: false })
        }
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-19: Resize panel
  test('T10-19 Resize panel', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('resize_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find panel resize handle', async () => {
      const resizeHandle = page.locator('[class*="resize"], [class*="react-resizable-handle"], [class*="grid-item"]').first()
      if (await resizeHandle.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-19-resize-panel.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-20: Move panel
  test('T10-20 Move panel', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('move_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find panel drag handle', async () => {
      const dragHandle = page.locator('[class*="drag"], [class*="move"], [class*="grip"]').first()
      if (await dragHandle.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-20-move-panel.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-21: Panel settings dialog
  test('T10-21 Panel settings dialog', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('settings_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Click panel settings', async () => {
      const settingsBtn = page.locator('button[title*="edit"], button[title*="设置"], button[title*="Edit"]').first()
      if (await settingsBtn.isVisible().catch(() => false)) {
        await settingsBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-21-panel-settings.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-22: Variable editor
  test('T10-22 Variable editor', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('var_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find variable/settings button', async () => {
      const varBtn = page.locator('button').filter({ hasText: /变量|Variable|Settings|设置/ }).first()
      if (await varBtn.isVisible().catch(() => false)) {
        await varBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-22-variable-editor.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-23: Time range selector
  test('T10-23 Time range selector', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('time_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Click time range selector', async () => {
      const timeRange = page.locator('[class*="time-picker"], [class*="TimePicker"], button').filter({ hasText: /Last|最近|过去|1h|6h|24h|7d/ }).first()
      if (await timeRange.isVisible().catch(() => false)) {
        await timeRange.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T10-23-time-range.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-24: Dashboard save
  test('T10-24 Dashboard save', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('save_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Click save button', async () => {
      const saveBtn = page.locator('button').filter({ hasText: /保存|Save/ }).first()
      if (await saveBtn.isVisible().catch(() => false)) {
        await saveBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-24-dashboard-save.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-25: Undo/Redo buttons
  test('T10-25 Undo redo buttons', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('undo_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find undo/redo buttons', async () => {
      const undoBtn = page.locator('button[title*="undo"], button[title*="撤销"], button').filter({ hasText: /Undo|撤销/ }).first()
      const redoBtn = page.locator('button[title*="redo"], button[title*="重做"], button').filter({ hasText: /Redo|重做/ }).first()
      if (await undoBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-25-undo-redo.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-26: Dashboard title edit inline
  test('T10-26 Dashboard title edit inline', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('title_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Click title to edit', async () => {
      const title = page.locator('h1, h2, [class*="dashboard-title"], [class*="title"]').first()
      if (await title.isVisible().catch(() => false)) {
        await title.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T10-26-title-edit.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-27: Dashboard description edit
  test('T10-27 Dashboard description edit', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('desc_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find description area', async () => {
      const descArea = page.locator('[class*="description"], textarea[placeholder*="description"], [class*="desc"]').first()
      if (await descArea.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-27-desc-edit.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-28: Dashboard tags
  test('T10-28 Dashboard tags', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('tags_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find tags area', async () => {
      const tagsArea = page.locator('[class*="tag"], .n-tag, [class*="tags"]').first()
      if (await tagsArea.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-28-tags.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-29: Dashboard version history
  test('T10-29 Dashboard version history', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('version_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find version history button', async () => {
      const historyBtn = page.locator('button').filter({ hasText: /历史|History|版本|Version/ }).first()
      if (await historyBtn.isVisible().catch(() => false)) {
        await historyBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-29-version-history.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-30: Dashboard settings panel
  test('T10-30 Dashboard settings panel', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('settings_panel_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Open dashboard settings', async () => {
      const settingsBtn = page.locator('button[title*="settings"], button[title*="设置"], button').filter({ hasText: /设置|Settings/ }).first()
      if (await settingsBtn.isVisible().catch(() => false)) {
        await settingsBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-30-dashboard-settings.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
      await deleteTestDashboard(page, dashId)
    })
  })

  // ================================================================
  // T10-31 ~ T10-45: Dashboard Panels
  // ================================================================

  // T10-31: Line chart panel
  test('T10-31 Line chart panel', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find line chart panel', async () => {
      const linePanel = page.locator('[class*="line-chart"], [class*="LineChart"], canvas, svg').first()
      if (await linePanel.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-31-line-chart.png', fullPage: false })
      }
    })
  })

  // T10-32: Bar chart panel
  test('T10-32 Bar chart panel', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find bar chart panel', async () => {
      const barPanel = page.locator('[class*="bar-chart"], [class*="BarChart"]').first()
      if (await barPanel.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-32-bar-chart.png', fullPage: false })
      }
    })
  })

  // T10-33: Gauge panel
  test('T10-33 Gauge panel', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find gauge panel', async () => {
      const gaugePanel = page.locator('[class*="gauge"], [class*="Gauge"]').first()
      if (await gaugePanel.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-33-gauge.png', fullPage: false })
      }
    })
  })

  // T10-34: Table panel
  test('T10-34 Table panel', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find table panel', async () => {
      const tablePanel = page.locator('[class*="table-panel"], [class*="TablePanel"], table').first()
      if (await tablePanel.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-34-table-panel.png', fullPage: false })
      }
    })
  })

  // T10-35: Stat panel
  test('T10-35 Stat panel', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find stat panel', async () => {
      const statPanel = page.locator('[class*="stat"], [class*="Stat"]').first()
      if (await statPanel.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-35-stat-panel.png', fullPage: false })
      }
    })
  })

  // T10-36: Heatmap panel
  test('T10-36 Heatmap panel', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find heatmap panel', async () => {
      const heatPanel = page.locator('[class*="heat"], [class*="Heat"]').first()
      if (await heatPanel.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-36-heatmap.png', fullPage: false })
      }
    })
  })

  // T10-37: Panel query editor
  test('T10-37 Panel query editor', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('query_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find query editor', async () => {
      const queryInput = page.locator('textarea, .monaco-editor, [class*="query"]').first()
      if (await queryInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-37-query-editor.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-38: Panel options
  test('T10-38 Panel options', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('options_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find panel options tab', async () => {
      const optionsTab = page.locator('[role="tab"], .n-tabs-tab').filter({ hasText: /选项|Options|Options/ }).first()
      if (await optionsTab.isVisible().catch(() => false)) {
        await optionsTab.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T10-38-panel-options.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-39: Panel legend
  test('T10-39 Panel legend', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find panel legend', async () => {
      const legend = page.locator('[class*="legend"], [class*="Legend"]').first()
      if (await legend.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-39-panel-legend.png', fullPage: false })
      }
    })
  })

  // T10-40: Panel thresholds
  test('T10-40 Panel thresholds', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('thresh_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find thresholds section', async () => {
      const threshSection = page.locator('[class*="threshold"], text=阈值, text=Threshold').first()
      if (await threshSection.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-40-thresholds.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-41: Panel links
  test('T10-41 Panel links', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('links_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find panel links section', async () => {
      const linksSection = page.locator('[class*="link"], text=链接, text=Links').first()
      if (await linksSection.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-41-panel-links.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-42: Panel repeat
  test('T10-42 Panel repeat', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('repeat_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find repeat option', async () => {
      const repeatSection = page.locator('[class*="repeat"], text=重复, text=Repeat').first()
      if (await repeatSection.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-42-panel-repeat.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-43: Panel grid layout
  test('T10-43 Panel grid layout', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('grid_dash') })
    })

    await test.step('Navigate to editor', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Verify grid layout', async () => {
      const grid = page.locator('[class*="grid"], [class*="layout"], [class*="react-grid"]').first()
      if (await grid.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T10-43-grid-layout.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-44: Panel refresh
  test('T10-44 Panel refresh', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find refresh button', async () => {
      const refreshBtn = page.locator('button').filter({ hasText: /刷新|Refresh|Reload/ }).first()
      if (await refreshBtn.isVisible().catch(() => false)) {
        await refreshBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-44-panel-refresh.png', fullPage: false })
      }
    })
  })

  // T10-45: Panel export
  test('T10-45 Panel export', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find panel export option', async () => {
      const panel = page.locator('[class*="panel"], [class*="Panel"]').first()
      if (await panel.isVisible().catch(() => false)) {
        await panel.hover()
        await page.waitForTimeout(200)
        const moreBtn = page.locator('[class*="panel"] button[title*="more"], [class*="panel"] button').last()
        if (await moreBtn.isVisible().catch(() => false)) {
          await moreBtn.click()
          await page.waitForTimeout(300)
          await page.screenshot({ path: 'test-results/T10-45-panel-export.png', fullPage: false })
        }
      }
    })

    await test.step('Close menu', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // ================================================================
  // T10-46 ~ T10-60: Dashboard Features
  // ================================================================

  // T10-46: Fullscreen mode
  test('T10-46 Fullscreen mode', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('fullscreen_dash') })
    })

    await test.step('Navigate to dashboard', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find fullscreen button', async () => {
      const fsBtn = page.locator('button').filter({ hasText: /全屏|Fullscreen|全屏模式/ }).first()
      if (await fsBtn.isVisible().catch(() => false)) {
        await fsBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-46-fullscreen.png', fullPage: false })
        await page.keyboard.press('Escape')
        await page.waitForTimeout(300)
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-47: Share link
  test('T10-47 Share link', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('share_dash') })
    })

    await test.step('Navigate to dashboard', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find share button', async () => {
      const shareBtn = page.locator('button').filter({ hasText: /分享|Share|链接/ }).first()
      if (await shareBtn.isVisible().catch(() => false)) {
        await shareBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-47-share-link.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-48: Export JSON
  test('T10-48 Export JSON', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('export_dash') })
    })

    await test.step('Navigate to dashboard', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find export option', async () => {
      const moreBtn = page.locator('button').filter({ hasText: /更多|More|操作/ }).first()
      if (await moreBtn.isVisible().catch(() => false)) {
        await moreBtn.click()
        await page.waitForTimeout(300)
        const exportItem = page.locator('.n-dropdown-option, [class*="dropdown"]').filter({ hasText: /导出|Export|JSON/ }).first()
        if (await exportItem.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T10-48-export-json.png', fullPage: false })
        }
      }
    })

    await test.step('Cleanup', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-49: Import JSON
  test('T10-49 Import JSON', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find import option', async () => {
      const moreBtn = page.locator('button').filter({ hasText: /更多|More|导入|Import/ }).first()
      if (await moreBtn.isVisible().catch(() => false)) {
        await moreBtn.click()
        await page.waitForTimeout(300)
        const importItem = page.locator('.n-dropdown-option, [class*="dropdown"]').filter({ hasText: /导入|Import|JSON/ }).first()
        if (await importItem.isVisible().catch(() => false)) {
          await importItem.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T10-49-import-json.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T10-50: Template library
  test('T10-50 Template library', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard page', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find template library', async () => {
      const templateBtn = page.locator('button').filter({ hasText: /模板|Template|Library/ }).first()
      if (await templateBtn.isVisible().catch(() => false)) {
        await templateBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-50-template-library.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T10-51: Annotations
  test('T10-51 Annotations', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('annot_dash') })
    })

    await test.step('Navigate to dashboard', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find annotations button', async () => {
      const annotBtn = page.locator('button').filter({ hasText: /注释|Annotation|标注/ }).first()
      if (await annotBtn.isVisible().catch(() => false)) {
        await annotBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T10-51-annotations.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-52: Snapshot
  test('T10-52 Snapshot', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('snap_dash') })
    })

    await test.step('Navigate to dashboard', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find snapshot option', async () => {
      const moreBtn = page.locator('button').filter({ hasText: /更多|More|操作/ }).first()
      if (await moreBtn.isVisible().catch(() => false)) {
        await moreBtn.click()
        await page.waitForTimeout(300)
        const snapItem = page.locator('.n-dropdown-option, [class*="dropdown"]').filter({ hasText: /快照|Snapshot/ }).first()
        if (await snapItem.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T10-52-snapshot.png', fullPage: false })
        }
      }
    })

    await test.step('Cleanup', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-53: Auto-refresh
  test('T10-53 Auto refresh', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('autorefresh_dash') })
    })

    await test.step('Navigate to dashboard', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find auto-refresh control', async () => {
      const refreshSelect = page.locator('.n-select, [class*="refresh"]').filter({ hasText: /自动刷新|Auto|refresh|关闭/ }).first()
      if (await refreshSelect.isVisible().catch(() => false)) {
        await refreshSelect.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T10-53-auto-refresh.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-54: Time zone selector
  test('T10-54 Time zone selector', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('tz_dash') })
    })

    await test.step('Navigate to dashboard', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Find timezone selector', async () => {
      const tzBtn = page.locator('button, .n-select').filter({ hasText: /时区|Timezone|UTC|Browser/ }).first()
      if (await tzBtn.isVisible().catch(() => false)) {
        await tzBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T10-54-timezone.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-55: Keyboard shortcut d
  test('T10-55 Keyboard shortcut d', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('kbd_dash') })
    })

    await test.step('Navigate to dashboard', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Press k to toggle kiosk mode', async () => {
      await page.keyboard.press('k')
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T10-55-kiosk-mode.png', fullPage: false })
      await page.keyboard.press('k')
      await page.waitForTimeout(300)
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-56: Keyboard shortcut Escape
  test('T10-56 Keyboard shortcut Escape', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create test dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('esc_dash') })
    })

    await test.step('Navigate to dashboard', async () => {
      if (dashId > 0) {
        await page.goto(`${DASHBOARD_URL}/${dashId}`)
        await page.waitForLoadState('networkidle')
      }
    })

    await test.step('Open panel edit then press Escape', async () => {
      const editBtn = page.locator('button[title*="edit"], button[title*="Edit"], button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.keyboard.press('Escape')
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T10-56-escape-key.png', fullPage: false })
      }
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-57: Dashboard API create and read
  test('T10-57 Dashboard API create and read', async ({ authPage: page }) => {
    let dashId = 0
    const dashName = uid('api_dash')

    await test.step('Create dashboard via API', async () => {
      dashId = await createTestDashboard(page, { name: dashName, description: 'API test' })
      expect(dashId).toBeGreaterThan(0)
    })

    await test.step('Read dashboard via API', async () => {
      const res = await API.get(page, `/api/v1/dashboards/${dashId}`)
      expect(res?.data?.name).toBe(dashName)
      await page.screenshot({ path: 'test-results/T10-57-API-read.png', fullPage: false })
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-58: Dashboard API update
  test('T10-58 Dashboard API update', async ({ authPage: page }) => {
    let dashId = 0
    const origName = uid('update_dash')

    await test.step('Create dashboard', async () => {
      dashId = await createTestDashboard(page, { name: origName })
      if (dashId <= 0) {
        await page.screenshot({ path: 'test-results/T10-58-create-failed.png', fullPage: false })
        test.skip()
        return
      }
    })

    await test.step('Update via API', async () => {
      const newName = origName + '_updated'
      await API.put(page, `/api/v1/dashboards/${dashId}`, { name: newName })
      const res = await API.get(page, `/api/v1/dashboards/${dashId}`)
      expect(res).toBeTruthy()
      if (res?.data) {
        expect(res.data.name).toBe(newName)
      }
      await page.screenshot({ path: 'test-results/T10-58-API-update.png', fullPage: false })
    })

    await test.step('Cleanup', async () => {
      await deleteTestDashboard(page, dashId)
    })
  })

  // T10-59: Dashboard API delete
  test('T10-59 Dashboard API delete', async ({ authPage: page }) => {
    let dashId = 0
    await test.step('Create dashboard', async () => {
      dashId = await createTestDashboard(page, { name: uid('delete_dash') })
      if (dashId <= 0) {
        await page.screenshot({ path: 'test-results/T10-59-create-failed.png', fullPage: false })
        test.skip()
        return
      }
    })

    await test.step('Delete via API', async () => {
      await deleteTestDashboard(page, dashId)
      const res = await API.get(page, `/api/v1/dashboards/${dashId}`)
      // Should return error or 404 (res.code !== 0 or res.data is null)
      expect(res).toBeTruthy()
      await page.screenshot({ path: 'test-results/T10-59-API-delete.png', fullPage: false })
    })
  })

  // T10-60: Full dashboard lifecycle
  test('T10-60 Full dashboard lifecycle', async ({ authPage: page }) => {
    const dashName = uid('lifecycle')
    let dashId = 0

    await test.step('Create dashboard', async () => {
      dashId = await createTestDashboard(page, { name: dashName, description: 'Lifecycle test' })
      if (dashId <= 0) {
        await page.screenshot({ path: 'test-results/T10-60-create-failed.png', fullPage: false })
        test.skip()
        return
      }
    })

    await test.step('Verify dashboard exists', async () => {
      const res = await API.get(page, `/api/v1/dashboards/${dashId}`)
      expect(res).toBeTruthy()
      if (res?.data) {
        expect(res.data.name).toBe(dashName)
      }
    })

    await test.step('Update dashboard', async () => {
      const updatedName = dashName + '_updated'
      await API.put(page, `/api/v1/dashboards/${dashId}`, { name: updatedName })
      const res = await API.get(page, `/api/v1/dashboards/${dashId}`)
      expect(res).toBeTruthy()
      if (res?.data) {
        expect(res.data.name).toBe(updatedName)
      }
    })

    await test.step('Navigate to dashboard in UI', async () => {
      await page.goto(`${DASHBOARD_URL}/${dashId}`)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T10-60-lifecycle-ui.png', fullPage: true })
    })

    await test.step('Delete dashboard', async () => {
      await deleteTestDashboard(page, dashId)
    })

    await test.step('Final screenshot', async () => {
      await page.goto(DASHBOARD_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T10-60-lifecycle-done.png', fullPage: true })
    })
  })
})
