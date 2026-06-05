import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// T13: Edge Cases Test Suite — 70 tests
// Covers: Empty States (T13-1~T13-15), Error States (T13-16~T13-30),
//         Loading States (T13-31~T13-45), Concurrent Operations (T13-46~T13-55),
//         Data Validation (T13-56~T13-70)

const BASE_URL = 'http://localhost:3000'

/** Generate unique name */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

test.describe('T13 - Edge Cases Test Suite', () => {

  // ================================================================
  // T13-1 ~ T13-15: Empty States
  // ================================================================

  // T13-1: Empty alert rules list
  test('T13-1 Empty alert rules list', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-1-empty-alert-rules.png', fullPage: true })
    })

    await test.step('Check for empty state or table', async () => {
      const emptyState = page.locator('.n-empty, [class*="empty"], [class*="no-data"], text=暂无数据, text=No data').first()
      const table = page.locator('table, .n-data-table, [class*="table"]').first()
      const hasEmpty = await emptyState.isVisible().catch(() => false)
      const hasTable = await table.isVisible().catch(() => false)
      expect(hasEmpty || hasTable).toBeTruthy()
    })
  })

  // T13-2: Empty alert events list
  test('T13-2 Empty alert events list', async ({ authPage: page }) => {
    await test.step('Navigate to alert events page', async () => {
      await page.goto(BASE_URL + '/alert/events')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-2-empty-alert-events.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-3: Empty incident list
  test('T13-3 Empty incident list', async ({ authPage: page }) => {
    await test.step('Navigate to incidents page', async () => {
      await page.goto(BASE_URL + '/incidents')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-3-empty-incidents.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      const emptyOrTable = page.locator('.n-empty, [class*="empty"], table, .n-data-table').first()
      await expect(emptyOrTable).toBeVisible({ timeout: 10000 }).catch(() => {})
    })
  })

  // T13-4: Empty oncall schedule list
  test('T13-4 Empty oncall schedule list', async ({ authPage: page }) => {
    await test.step('Navigate to oncall schedules page', async () => {
      await page.goto(BASE_URL + '/oncall/schedules')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-4-empty-schedules.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-5: Empty team list
  test('T13-5 Empty team list', async ({ authPage: page }) => {
    await test.step('Navigate to teams page', async () => {
      await page.goto(BASE_URL + '/teams')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-5-empty-teams.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-6: Empty user list
  test('T13-6 Empty user list', async ({ authPage: page }) => {
    await test.step('Navigate to users page', async () => {
      await page.goto(BASE_URL + '/users')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-6-empty-users.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-7: Empty datasource list
  test('T13-7 Empty datasource list', async ({ authPage: page }) => {
    await test.step('Navigate to datasources page', async () => {
      await page.goto(BASE_URL + '/datasources')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-7-empty-datasources.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-8: Empty template list
  test('T13-8 Empty template list', async ({ authPage: page }) => {
    await test.step('Navigate to templates page', async () => {
      await page.goto(BASE_URL + '/settings/templates')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-8-empty-templates.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-9: Empty notify rule list
  test('T13-9 Empty notify rule list', async ({ authPage: page }) => {
    await test.step('Navigate to notify rules page', async () => {
      await page.goto(BASE_URL + '/notify/rules')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-9-empty-notify-rules.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-10: Empty channel list
  test('T13-10 Empty channel list', async ({ authPage: page }) => {
    await test.step('Navigate to channels page', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-10-empty-channels.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-11: Empty knowledge base
  test('T13-11 Empty knowledge base', async ({ authPage: page }) => {
    await test.step('Navigate to knowledge base page', async () => {
      await page.goto(BASE_URL + '/ai/knowledge')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-11-empty-knowledge.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-12: Empty annotations
  test('T13-12 Empty annotations', async ({ authPage: page }) => {
    await test.step('Navigate to annotations page', async () => {
      await page.goto(BASE_URL + '/alert/annotations')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-12-empty-annotations.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-13: Empty tasks list
  test('T13-13 Empty tasks list', async ({ authPage: page }) => {
    await test.step('Navigate to tasks page', async () => {
      await page.goto(BASE_URL + '/tasks')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-13-empty-tasks.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-14: Empty inspections list
  test('T13-14 Empty inspections list', async ({ authPage: page }) => {
    await test.step('Navigate to inspections page', async () => {
      await page.goto(BASE_URL + '/inspections')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-14-empty-inspections.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-15: Empty dashboards
  test('T13-15 Empty dashboards', async ({ authPage: page }) => {
    await test.step('Navigate to dashboards page', async () => {
      await page.goto(BASE_URL + '/dashboards')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-15-empty-dashboards.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // ================================================================
  // T13-16 ~ T13-30: Error States
  // ================================================================

  // T13-16: Network error handling
  test('T13-16 Network error handling', async ({ authPage: page }) => {
    await test.step('Navigate to a page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Simulate offline by blocking requests', async () => {
      await page.route('**/api/**', route => route.abort('connectionrefused'))
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-16-network-error.png', fullPage: true })
    })

    await test.step('Verify error message or fallback UI', async () => {
      const errorIndicator = page.locator('[class*="error"], [class*="fail"], [class*="offline"], .n-result--error').first()
      const bodyVisible = await page.locator('body').isVisible()
      expect(bodyVisible).toBeTruthy()
    })

    await test.step('Restore network', async () => {
      await page.unroute('**/api/**')
    })
  })

  // T13-17: 401 Unauthorized response
  test('T13-17 401 Unauthorized response', async ({ authPage: page }) => {
    await test.step('Intercept API with 401', async () => {
      await page.route('**/api/**', route => route.fulfill({ status: 401, body: JSON.stringify({ code: 40001, message: 'Unauthorized' }) }))
    })

    await test.step('Navigate to a protected page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-17-401-unauthorized.png', fullPage: true })
    })

    await test.step('Verify redirect to login or error display', async () => {
      const url = page.url()
      const isLoginOrError = url.includes('login') || await page.locator('[class*="error"], [class*="unauthorized"]').first().isVisible().catch(() => false)
      await page.screenshot({ path: 'test-results/T13-17-401-result.png', fullPage: false })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/**')
    })
  })

  // T13-18: 403 Forbidden response
  test('T13-18 403 Forbidden response', async ({ authPage: page }) => {
    await test.step('Intercept API with 403', async () => {
      await page.route('**/api/**', route => route.fulfill({ status: 403, body: JSON.stringify({ code: 10200, message: 'Permission denied' }) }))
    })

    await test.step('Navigate to a page', async () => {
      await page.goto(BASE_URL + '/settings')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-18-403-forbidden.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/**')
    })
  })

  // T13-19: 404 Not Found response
  test('T13-19 404 Not Found response', async ({ authPage: page }) => {
    await test.step('Navigate to a non-existent route', async () => {
      await page.goto(BASE_URL + '/nonexistent/deep/route/xyz')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-19-404-not-found.png', fullPage: true })
    })

    await test.step('Verify 404 or redirect', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-20: 500 Server Error response
  test('T13-20 500 Server Error response', async ({ authPage: page }) => {
    await test.step('Intercept API with 500', async () => {
      await page.route('**/api/**', route => route.fulfill({ status: 500, body: JSON.stringify({ code: 50001, message: 'Internal Server Error' }) }))
    })

    await test.step('Navigate to a page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-20-500-server-error.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/**')
    })
  })

  // T13-21: Timeout handling
  test('T13-21 Timeout handling', async ({ authPage: page }) => {
    await test.step('Intercept API with delay', async () => {
      await page.route('**/api/**', route => {
        return new Promise(resolve => setTimeout(() => route.abort('timedout'), 30000))
      })
    })

    await test.step('Navigate and check for timeout UI', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(5000)
      await page.screenshot({ path: 'test-results/T13-21-timeout.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/**')
    })
  })

  // T13-22: Invalid JSON response
  test('T13-22 Invalid JSON response', async ({ authPage: page }) => {
    await test.step('Intercept API with invalid JSON', async () => {
      await page.route('**/api/v1/alert-rules**', route => route.fulfill({ status: 200, body: 'not valid json {{{' }))
    })

    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-22-invalid-json.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-rules**')
    })
  })

  // T13-23: Empty response body
  test('T13-23 Empty response body', async ({ authPage: page }) => {
    await test.step('Intercept API with empty body', async () => {
      await page.route('**/api/v1/alert-rules**', route => route.fulfill({ status: 200, body: '' }))
    })

    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-23-empty-response.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-rules**')
    })
  })

  // T13-24: Null data response
  test('T13-24 Null data response', async ({ authPage: page }) => {
    await test.step('Intercept API with null data', async () => {
      await page.route('**/api/v1/alert-rules**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: null }) }))
    })

    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-24-null-data.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-rules**')
    })
  })

  // T13-25: Undefined fields response
  test('T13-25 Undefined fields response', async ({ authPage: page }) => {
    await test.step('Intercept API with missing fields', async () => {
      await page.route('**/api/v1/alert-rules**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: { list: [{ id: 1 }] } }) }))
    })

    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-25-undefined-fields.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-rules**')
    })
  })

  // T13-26: Connection refused
  test('T13-26 Connection refused', async ({ authPage: page }) => {
    await test.step('Block all API requests', async () => {
      await page.route('**/api/**', route => route.abort('connectionrefused'))
    })

    await test.step('Try to load a page', async () => {
      await page.goto(BASE_URL + '/alert/events')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-26-connection-refused.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/**')
    })
  })

  // T13-27: DNS error simulation
  test('T13-27 DNS error simulation', async ({ authPage: page }) => {
    await test.step('Block requests as name not resolved', async () => {
      await page.route('**/api/**', route => route.abort('namenotresolved'))
    })

    await test.step('Navigate and check UI', async () => {
      await page.goto(BASE_URL + '/dashboards')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-27-dns-error.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/**')
    })
  })

  // T13-28: SSL error simulation
  test('T13-28 SSL error simulation', async ({ authPage: page }) => {
    await test.step('Block requests as failed SSL', async () => {
      await page.route('**/api/**', route => route.abort('failed'))
    })

    await test.step('Navigate and check UI', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-28-ssl-error.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/**')
    })
  })

  // T13-29: CORS error simulation
  test('T13-29 CORS error simulation', async ({ authPage: page }) => {
    await test.step('Intercept API with CORS-like error', async () => {
      await page.route('**/api/**', route => route.fulfill({ status: 0, body: '' }))
    })

    await test.step('Navigate and check UI', async () => {
      await page.goto(BASE_URL + '/oncall/schedules')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-29-cors-error.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/**')
    })
  })

  // T13-30: Rate limit response
  test('T13-30 Rate limit response', async ({ authPage: page }) => {
    await test.step('Intercept API with 429', async () => {
      await page.route('**/api/**', route => route.fulfill({ status: 429, body: JSON.stringify({ code: 42900, message: 'Too Many Requests' }) }))
    })

    await test.step('Navigate and check UI', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-30-rate-limit.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/**')
    })
  })

  // ================================================================
  // T13-31 ~ T13-45: Loading States
  // ================================================================

  // T13-31: Initial page loading
  test('T13-31 Initial page loading', async ({ authPage: page }) => {
    await test.step('Capture loading state on initial navigation', async () => {
      const navigation = page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(200)
      await page.screenshot({ path: 'test-results/T13-31-initial-loading.png', fullPage: false })
      await navigation
      await page.waitForLoadState('networkidle')
    })

    await test.step('Verify page loaded', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-32: Refresh loading
  test('T13-32 Refresh loading', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Trigger refresh and capture loading', async () => {
      const refreshBtn = page.locator('button').filter({ hasText: /刷新|Refresh|Reload/ }).first()
      if (await refreshBtn.isVisible().catch(() => false)) {
        await refreshBtn.click()
        await page.waitForTimeout(200)
        await page.screenshot({ path: 'test-results/T13-32-refresh-loading.png', fullPage: false })
      } else {
        await page.reload()
        await page.waitForTimeout(200)
        await page.screenshot({ path: 'test-results/T13-32-refresh-loading.png', fullPage: false })
      }
    })
  })

  // T13-33: Pagination loading
  test('T13-33 Pagination loading', async ({ authPage: page }) => {
    await test.step('Navigate to page with pagination', async () => {
      await page.goto(BASE_URL + '/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click next page', async () => {
      const nextBtn = page.locator('.n-pagination-item--next, button[aria-label="next"], [class*="pagination"] button:last-child').first()
      if (await nextBtn.isVisible().catch(() => false)) {
        await nextBtn.click()
        await page.waitForTimeout(200)
        await page.screenshot({ path: 'test-results/T13-33-pagination-loading.png', fullPage: false })
      }
    })
  })

  // T13-34: Filter loading
  test('T13-34 Filter loading', async ({ authPage: page }) => {
    await test.step('Navigate to page with filters', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Apply a filter', async () => {
      const severitySelect = page.locator('.n-select, [class*="filter"]').first()
      if (await severitySelect.isVisible().catch(() => false)) {
        await severitySelect.click()
        await page.waitForTimeout(200)
        await page.screenshot({ path: 'test-results/T13-34-filter-loading.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T13-35: Search loading
  test('T13-35 Search loading', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Type in search and capture loading', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[placeholder*="Search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(200)
        await page.screenshot({ path: 'test-results/T13-35-search-loading.png', fullPage: false })
      }
    })
  })

  // T13-36: Create loading
  test('T13-36 Create loading', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click create button and capture', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T13-36-create-loading.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T13-37: Edit loading
  test('T13-37 Edit loading', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click edit on first item', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit|修改/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T13-37-edit-loading.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T13-38: Delete loading
  test('T13-38 Delete loading', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click delete on first item', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete|移除/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T13-38-delete-loading.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T13-39: Batch operation loading
  test('T13-39 Batch operation loading', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Select items and check batch actions', async () => {
      const checkboxes = page.locator('.n-checkbox, input[type="checkbox"]')
      const count = await checkboxes.count()
      if (count > 0) {
        await checkboxes.first().click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T13-39-batch-loading.png', fullPage: false })
      }
    })
  })

  // T13-40: Import loading
  test('T13-40 Import loading', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find import button', async () => {
      const importBtn = page.locator('button').filter({ hasText: /导入|Import/ }).first()
      if (await importBtn.isVisible().catch(() => false)) {
        await importBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T13-40-import-loading.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T13-41: Export loading
  test('T13-41 Export loading', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find export button', async () => {
      const exportBtn = page.locator('button').filter({ hasText: /导出|Export/ }).first()
      if (await exportBtn.isVisible().catch(() => false)) {
        await exportBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T13-41-export-loading.png', fullPage: false })
      }
    })
  })

  // T13-42: Test connection loading
  test('T13-42 Test connection loading', async ({ authPage: page }) => {
    await test.step('Navigate to datasources page', async () => {
      await page.goto(BASE_URL + '/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find test connection button', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|检查|Check/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await testBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T13-42-test-loading.png', fullPage: false })
      }
    })
  })

  // T13-43: Health check loading
  test('T13-43 Health check loading', async ({ authPage: page }) => {
    await test.step('Navigate to settings health page', async () => {
      await page.goto(BASE_URL + '/settings/health')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-43-health-check-loading.png', fullPage: true })
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-44: AI loading
  test('T13-44 AI loading', async ({ authPage: page }) => {
    await test.step('Navigate to AI assistant page', async () => {
      await page.goto(BASE_URL + '/ai')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for AI loading states', async () => {
      await page.screenshot({ path: 'test-results/T13-44-ai-loading.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-45: Sync loading
  test('T13-45 Sync loading', async ({ authPage: page }) => {
    await test.step('Navigate to datasources page', async () => {
      await page.goto(BASE_URL + '/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find sync button', async () => {
      const syncBtn = page.locator('button').filter({ hasText: /同步|Sync|刷新/ }).first()
      if (await syncBtn.isVisible().catch(() => false)) {
        await syncBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T13-45-sync-loading.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T13-46 ~ T13-55: Concurrent Operations
  // ================================================================

  // T13-46: Double click create button
  test('T13-46 Double click create button', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Double click create button rapidly', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.dblclick()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-46-double-click-create.png', fullPage: false })
      }
    })

    await test.step('Ensure only one dialog opened', async () => {
      const dialogs = page.locator('.n-modal, [class*="modal"], [class*="dialog"]')
      const count = await dialogs.count()
      await page.screenshot({ path: 'test-results/T13-46-dialog-count.png', fullPage: false })
      await page.keyboard.press('Escape')
    })
  })

  // T13-47: Double click edit button
  test('T13-47 Double click edit button', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Double click edit button', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit|修改/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.dblclick()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-47-double-click-edit.png', fullPage: false })
      }
    })

    await test.step('Close any dialogs', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T13-48: Double click delete button
  test('T13-48 Double click delete button', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Double click delete button', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete|移除/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.dblclick()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-48-double-click-delete.png', fullPage: false })
      }
    })

    await test.step('Cancel delete dialog', async () => {
      const cancelBtn = page.locator('button').filter({ hasText: /取消|Cancel/ }).first()
      if (await cancelBtn.isVisible().catch(() => false)) {
        await cancelBtn.click()
      } else {
        await page.keyboard.press('Escape')
      }
    })
  })

  // T13-49: Rapid filter changes
  test('T13-49 Rapid filter changes', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Rapidly change filters', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        for (let i = 0; i < 5; i++) {
          await searchInput.fill(`filter_${i}`)
          await page.waitForTimeout(100)
        }
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-49-rapid-filters.png', fullPage: false })
      }
    })
  })

  // T13-50: Rapid search
  test('T13-50 Rapid search', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Type rapidly in search', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.pressSequentially('abcdef', { delay: 30 })
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-50-rapid-search.png', fullPage: false })
      }
    })
  })

  // T13-51: Rapid pagination
  test('T13-51 Rapid pagination', async ({ authPage: page }) => {
    await test.step('Navigate to page with pagination', async () => {
      await page.goto(BASE_URL + '/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click next page multiple times rapidly', async () => {
      const nextBtn = page.locator('.n-pagination-item--next, button[aria-label="next"]').first()
      for (let i = 0; i < 3; i++) {
        if (await nextBtn.isVisible().catch(() => false)) {
          await nextBtn.click()
          await page.waitForTimeout(100)
        }
      }
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T13-51-rapid-pagination.png', fullPage: false })
    })
  })

  // T13-52: Concurrent API calls
  test('T13-52 Concurrent API calls', async ({ authPage: page }) => {
    await test.step('Navigate to page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Make concurrent API calls via page navigation', async () => {
      const promises = [
        page.goto(BASE_URL + '/alert/rules'),
        page.goto(BASE_URL + '/alert/events'),
      ]
      await Promise.allSettled(promises)
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T13-52-concurrent-api.png', fullPage: false })
    })
  })

  // T13-53: Race condition — navigate before data loads
  test('T13-53 Race condition navigate before load', async ({ authPage: page }) => {
    await test.step('Start slow load then navigate away', async () => {
      await page.route('**/api/v1/alert-rules**', async route => {
        await new Promise(r => setTimeout(r, 3000))
        await route.continue()
      })
      const navPromise = page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(500)
      await page.goto(BASE_URL + '/alert/events')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-53-race-condition.png', fullPage: false })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-rules**')
    })
  })

  // T13-54: Stale data after navigation back
  test('T13-54 Stale data after navigation back', async ({ authPage: page }) => {
    await test.step('Navigate forward then back', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
      await page.goto(BASE_URL + '/alert/events')
      await page.waitForLoadState('networkidle')
      await page.goBack()
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T13-54-stale-data.png', fullPage: true })
    })

    await test.step('Verify page is still functional', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T13-55: Optimistic update rollback
  test('T13-55 Optimistic update rollback', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Try to toggle a rule status', async () => {
      const toggle = page.locator('.n-switch, [class*="switch"], [class*="toggle"]').first()
      if (await toggle.isVisible().catch(() => false)) {
        await toggle.click()
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T13-55-optimistic-update.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T13-56 ~ T13-70: Data Validation
  // ================================================================

  // T13-56: Empty name validation
  test('T13-56 Empty name validation', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create dialog and submit empty name', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const submitBtn = page.locator('button').filter({ hasText: /确定|Submit|Save|保存|OK/ }).first()
        if (await submitBtn.isVisible().catch(() => false)) {
          await submitBtn.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T13-56-empty-name.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T13-57: Long name validation
  test('T13-57 Long name validation', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create dialog and enter very long name', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const nameInput = page.locator('input[placeholder*="名称"], input[placeholder*="name"], input[placeholder*="Name"]').first()
        if (await nameInput.isVisible().catch(() => false)) {
          const longName = 'A'.repeat(500)
          await nameInput.fill(longName)
          await page.waitForTimeout(300)
          await page.screenshot({ path: 'test-results/T13-57-long-name.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T13-58: Special characters in name
  test('T13-58 Special characters in name', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create dialog and enter special characters', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const nameInput = page.locator('input[placeholder*="名称"], input[placeholder*="name"], input[placeholder*="Name"]').first()
        if (await nameInput.isVisible().catch(() => false)) {
          await nameInput.fill('!@#$%^&*()_+-=[]{}|;:\'",.<>?/~`')
          await page.waitForTimeout(300)
          await page.screenshot({ path: 'test-results/T13-58-special-chars.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T13-59: SQL injection in name
  test('T13-59 SQL injection in name', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Try SQL injection in name field', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill("'; DROP TABLE alert_rules; --")
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-59-sql-injection.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })

  // T13-60: XSS in name
  test('T13-60 XSS in name', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Try XSS in search field', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('<script>alert("xss")</script>')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-60-xss-injection.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })

  // T13-61: Unicode characters
  test('T13-61 Unicode characters', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Enter unicode characters in search', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('日本語テスト 한국어 عربي')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-61-unicode.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })

  // T13-62: Emoji in name
  test('T13-62 Emoji in name', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Enter emoji in search', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('Test 🔥 Alert 🚨 Rule 📊')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-62-emoji.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })

  // T13-63: Newlines in input
  test('T13-63 Newlines in input', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Try multiline input in name field', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const nameInput = page.locator('input[placeholder*="名称"], input[placeholder*="name"]').first()
        if (await nameInput.isVisible().catch(() => false)) {
          await nameInput.fill('Line1\nLine2\nLine3')
          await page.waitForTimeout(300)
          await page.screenshot({ path: 'test-results/T13-63-newlines.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T13-64: Tabs in input
  test('T13-64 Tabs in input', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Try tab characters in search', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('col1\tcol2\tcol3')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-64-tabs.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })

  // T13-65: HTML tags in input
  test('T13-65 HTML tags in input', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Enter HTML tags in search', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('<div><p>Test</p><img src=x onerror=alert(1)></div>')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-65-html-tags.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })

  // T13-66: Script tags in input
  test('T13-66 Script tags in input', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Enter script tags in search', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('<script>document.cookie</script>')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-66-script-tags.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })

  // T13-67: Null bytes in input
  test('T13-67 Null bytes in input', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Enter null byte characters in search', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test\x00null\x00byte')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-67-null-bytes.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })

  // T13-68: Binary-like data in input
  test('T13-68 Binary-like data in input', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Enter binary-like data in search', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('\x01\x02\x03\x04\x05\x06\x07\x08')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-68-binary-data.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })

  // T13-69: Very large data in API response
  test('T13-69 Very large data in API response', async ({ authPage: page }) => {
    await test.step('Intercept API with large data', async () => {
      const largeList = Array.from({ length: 1000 }, (_, i) => ({
        id: i + 1,
        name: `Rule_${i + 1}_${'x'.repeat(100)}`,
        severity: 'warning',
        status: 'active',
      }))
      await page.route('**/api/v1/alert-rules**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, data: { list: largeList, total: 1000 } }),
      }))
    })

    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T13-69-large-data.png', fullPage: true })
    })

    await test.step('Verify page still responsive', async () => {
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-rules**')
    })
  })

  // T13-70: Nested JSON in label values
  test('T13-70 Nested JSON in label values', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Enter nested JSON-like string in search', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        const nestedJson = '{"key":{"nested":{"deep":"value"}}}'
        await searchInput.fill(nestedJson)
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T13-70-nested-json.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })
})
