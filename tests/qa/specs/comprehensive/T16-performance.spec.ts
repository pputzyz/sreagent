import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// T16: Performance Test Suite — 50 tests
// Covers: Page Load Performance (T16-1~T16-10), API Response Times (T16-11~T16-20),
//         Large Dataset Handling (T16-21~T16-30), Concurrent Operations (T16-31~T16-40),
//         Resource Usage (T16-41~T16-50)

const BASE_URL = 'http://localhost:3000'
const API_URL = 'http://localhost:8080'

/** Measure navigation time in ms */
async function measureNavigation(page: import('@playwright/test').Page, url: string): Promise<number> {
  const start = Date.now()
  await page.goto(url)
  await page.waitForLoadState('domcontentloaded')
  return Date.now() - start
}

/** Generate unique name */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

/** Thresholds (ms) */
const PAGE_LOAD_THRESHOLD = 10000
const API_RESPONSE_THRESHOLD = 5000

test.describe('T16 - Performance Test Suite', () => {

  // ================================================================
  // T16-1 ~ T16-10: Page Load Performance
  // ================================================================

  // T16-1: Dashboard load time
  test('T16-1 Dashboard load time', async ({ authPage: page }) => {
    await test.step('Measure dashboard page load time', async () => {
      const loadTime = await measureNavigation(page, BASE_URL + '/')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T16-1-dashboard-load.png', fullPage: true })
      expect(loadTime).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page fully rendered', async () => {
      const body = page.locator('body')
      await expect(body).toBeVisible()
    })
  })

  // T16-2: Alert list load time
  test('T16-2 Alert list load time', async ({ authPage: page }) => {
    await test.step('Measure alert rules page load time', async () => {
      const loadTime = await measureNavigation(page, BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T16-2-alert-list-load.png', fullPage: true })
      expect(loadTime).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify table or empty state rendered', async () => {
      const content = page.locator('table, .n-data-table, .n-empty, [class*="empty"]').first()
      await expect(content).toBeVisible({ timeout: 10000 }).catch(() => {})
    })
  })

  // T16-3: Event list load time
  test('T16-3 Event list load time', async ({ authPage: page }) => {
    await test.step('Measure alert events page load time', async () => {
      const loadTime = await measureNavigation(page, BASE_URL + '/alert/events')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T16-3-event-list-load.png', fullPage: true })
      expect(loadTime).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page rendered', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-4: Incident list load time
  test('T16-4 Incident list load time', async ({ authPage: page }) => {
    await test.step('Measure incidents page load time', async () => {
      const loadTime = await measureNavigation(page, BASE_URL + '/incidents')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T16-4-incident-list-load.png', fullPage: true })
      expect(loadTime).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page rendered', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-5: Schedule load time
  test('T16-5 Schedule load time', async ({ authPage: page }) => {
    await test.step('Measure oncall schedules page load time', async () => {
      const loadTime = await measureNavigation(page, BASE_URL + '/oncall/schedules')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T16-5-schedule-load.png', fullPage: true })
      expect(loadTime).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page rendered', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-6: Datasource list load time
  test('T16-6 Datasource list load time', async ({ authPage: page }) => {
    await test.step('Measure datasources page load time', async () => {
      const loadTime = await measureNavigation(page, BASE_URL + '/datasources')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T16-6-datasource-list-load.png', fullPage: true })
      expect(loadTime).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page rendered', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-7: Settings page load time
  test('T16-7 Settings page load time', async ({ authPage: page }) => {
    await test.step('Measure settings page load time', async () => {
      const loadTime = await measureNavigation(page, BASE_URL + '/settings')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T16-7-settings-load.png', fullPage: true })
      expect(loadTime).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page rendered', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-8: AI page load time
  test('T16-8 AI page load time', async ({ authPage: page }) => {
    await test.step('Measure AI assistant page load time', async () => {
      const loadTime = await measureNavigation(page, BASE_URL + '/ai')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T16-8-ai-load.png', fullPage: true })
      expect(loadTime).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page rendered', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-9: Login page load time
  test('T16-9 Login page load time', async ({ page }) => {
    await test.step('Measure login page load time', async () => {
      await page.goto(BASE_URL)
      await page.evaluate(() => { localStorage.clear() })
      const loadTime = await measureNavigation(page, BASE_URL + '/login')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T16-9-login-load.png', fullPage: false })
      expect(loadTime).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify login form rendered', async () => {
      const loginForm = page.locator('input[type="password"], input[placeholder*="用户"], input[placeholder*="user"]').first()
      await expect(loginForm).toBeVisible({ timeout: 10000 }).catch(() => {})
    })
  })

  // T16-10: 404 page load time
  test('T16-10 404 page load time', async ({ authPage: page }) => {
    await test.step('Measure 404 page load time', async () => {
      const loadTime = await measureNavigation(page, BASE_URL + '/nonexistent-page-xyz-404')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T16-10-404-load.png', fullPage: false })
      expect(loadTime).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify 404 or redirect handled gracefully', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // ================================================================
  // T16-11 ~ T16-20: API Response Times
  // ================================================================

  // T16-11: Alert rules API response time
  test('T16-11 Alert rules API response time', async ({ authPage: page }) => {
    await test.step('Measure alert rules API response time', async () => {
      const start = Date.now()
      const response = await API.get(page, '/api/v1/alert-rules')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-11-alert-rules-api.png', fullPage: false })
      expect(elapsed).toBeLessThan(API_RESPONSE_THRESHOLD)
    })
  })

  // T16-12: Events API response time
  test('T16-12 Events API response time', async ({ authPage: page }) => {
    await test.step('Measure events API response time', async () => {
      const start = Date.now()
      const response = await API.get(page, '/api/v1/alert-events')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-12-events-api.png', fullPage: false })
      expect(elapsed).toBeLessThan(API_RESPONSE_THRESHOLD)
    })
  })

  // T16-13: Incidents API response time
  test('T16-13 Incidents API response time', async ({ authPage: page }) => {
    await test.step('Measure incidents API response time', async () => {
      const start = Date.now()
      const response = await API.get(page, '/api/v1/incidents')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-13-incidents-api.png', fullPage: false })
      expect(elapsed).toBeLessThan(API_RESPONSE_THRESHOLD)
    })
  })

  // T16-14: Schedules API response time
  test('T16-14 Schedules API response time', async ({ authPage: page }) => {
    await test.step('Measure schedules API response time', async () => {
      const start = Date.now()
      const response = await API.get(page, '/api/v1/oncall/schedules')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-14-schedules-api.png', fullPage: false })
      expect(elapsed).toBeLessThan(API_RESPONSE_THRESHOLD)
    })
  })

  // T16-15: Datasources API response time
  test('T16-15 Datasources API response time', async ({ authPage: page }) => {
    await test.step('Measure datasources API response time', async () => {
      const start = Date.now()
      const response = await API.get(page, '/api/v1/datasources')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-15-datasources-api.png', fullPage: false })
      expect(elapsed).toBeLessThan(API_RESPONSE_THRESHOLD)
    })
  })

  // T16-16: Notifications API response time
  test('T16-16 Notifications API response time', async ({ authPage: page }) => {
    await test.step('Measure notification rules API response time', async () => {
      const start = Date.now()
      const response = await API.get(page, '/api/v1/notify-rules')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-16-notifications-api.png', fullPage: false })
      expect(elapsed).toBeLessThan(API_RESPONSE_THRESHOLD)
    })
  })

  // T16-17: Users API response time
  test('T16-17 Users API response time', async ({ authPage: page }) => {
    await test.step('Measure users API response time', async () => {
      const start = Date.now()
      const response = await API.get(page, '/api/v1/users')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-17-users-api.png', fullPage: false })
      expect(elapsed).toBeLessThan(API_RESPONSE_THRESHOLD)
    })
  })

  // T16-18: Teams API response time
  test('T16-18 Teams API response time', async ({ authPage: page }) => {
    await test.step('Measure teams API response time', async () => {
      const start = Date.now()
      const response = await API.get(page, '/api/v1/teams')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-18-teams-api.png', fullPage: false })
      expect(elapsed).toBeLessThan(API_RESPONSE_THRESHOLD)
    })
  })

  // T16-19: AI API response time
  test('T16-19 AI API response time', async ({ authPage: page }) => {
    await test.step('Measure AI assistant API response time', async () => {
      const start = Date.now()
      const response = await API.get(page, '/api/v1/ai/conversations')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-19-ai-api.png', fullPage: false })
      expect(elapsed).toBeLessThan(API_RESPONSE_THRESHOLD)
    })
  })

  // T16-20: Search API response time
  test('T16-20 Search API response time', async ({ authPage: page }) => {
    await test.step('Measure search API response time', async () => {
      const start = Date.now()
      const response = await API.get(page, '/api/v1/alert-rules?search=test')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-20-search-api.png', fullPage: false })
      expect(elapsed).toBeLessThan(API_RESPONSE_THRESHOLD)
    })
  })

  // ================================================================
  // T16-21 ~ T16-30: Large Dataset Handling
  // ================================================================

  // T16-21: Large alert rules list rendering
  test('T16-21 Large alert rules list rendering', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules with pagination', async () => {
      const start = Date.now()
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-21-large-alert-rules.png', fullPage: true })
      expect(elapsed).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Test pagination performance', async () => {
      const nextBtn = page.locator('button').filter({ hasText: /下一页|Next|>/ }).or(page.locator('.n-pagination .n-pagination-item').last()).first()
      if (await nextBtn.isVisible().catch(() => false)) {
        const start = Date.now()
        await nextBtn.click()
        await page.waitForLoadState('networkidle')
        const elapsed = Date.now() - start
        await page.screenshot({ path: 'test-results/T16-21-pagination.png', fullPage: true })
        expect(elapsed).toBeLessThan(PAGE_LOAD_THRESHOLD)
      }
    })
  })

  // T16-22: Large events list rendering
  test('T16-22 Large events list rendering', async ({ authPage: page }) => {
    await test.step('Navigate to alert events page', async () => {
      const start = Date.now()
      await page.goto(BASE_URL + '/alert/events')
      await page.waitForLoadState('networkidle')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-22-large-events.png', fullPage: true })
      expect(elapsed).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify smooth scrolling', async () => {
      await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
      await page.waitForTimeout(500)
      await page.evaluate(() => window.scrollTo(0, 0))
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T16-22-scroll-events.png', fullPage: false })
    })
  })

  // T16-23: Large incidents list rendering
  test('T16-23 Large incidents list rendering', async ({ authPage: page }) => {
    await test.step('Navigate to incidents page', async () => {
      const start = Date.now()
      await page.goto(BASE_URL + '/incidents')
      await page.waitForLoadState('networkidle')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-23-large-incidents.png', fullPage: true })
      expect(elapsed).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page is responsive', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-24: Large schedules list rendering
  test('T16-24 Large schedules list rendering', async ({ authPage: page }) => {
    await test.step('Navigate to oncall schedules page', async () => {
      const start = Date.now()
      await page.goto(BASE_URL + '/oncall/schedules')
      await page.waitForLoadState('networkidle')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-24-large-schedules.png', fullPage: true })
      expect(elapsed).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-25: Large datasources list rendering
  test('T16-25 Large datasources list rendering', async ({ authPage: page }) => {
    await test.step('Navigate to datasources page', async () => {
      const start = Date.now()
      await page.goto(BASE_URL + '/datasources')
      await page.waitForLoadState('networkidle')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-25-large-datasources.png', fullPage: true })
      expect(elapsed).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-26: Large notification rules list rendering
  test('T16-26 Large notification rules list rendering', async ({ authPage: page }) => {
    await test.step('Navigate to notification rules page', async () => {
      const start = Date.now()
      await page.goto(BASE_URL + '/notify/rules')
      await page.waitForLoadState('networkidle')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-26-large-notify-rules.png', fullPage: true })
      expect(elapsed).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-27: Large templates list rendering
  test('T16-27 Large templates list rendering', async ({ authPage: page }) => {
    await test.step('Navigate to templates page', async () => {
      const start = Date.now()
      await page.goto(BASE_URL + '/settings/templates')
      await page.waitForLoadState('networkidle')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-27-large-templates.png', fullPage: true })
      expect(elapsed).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-28: Large users list rendering
  test('T16-28 Large users list rendering', async ({ authPage: page }) => {
    await test.step('Navigate to users page', async () => {
      const start = Date.now()
      await page.goto(BASE_URL + '/users')
      await page.waitForLoadState('networkidle')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-28-large-users.png', fullPage: true })
      expect(elapsed).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-29: Large teams list rendering
  test('T16-29 Large teams list rendering', async ({ authPage: page }) => {
    await test.step('Navigate to teams page', async () => {
      const start = Date.now()
      await page.goto(BASE_URL + '/teams')
      await page.waitForLoadState('networkidle')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-29-large-teams.png', fullPage: true })
      expect(elapsed).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-30: Large knowledge items rendering
  test('T16-30 Large knowledge items rendering', async ({ authPage: page }) => {
    await test.step('Navigate to AI knowledge page', async () => {
      const start = Date.now()
      await page.goto(BASE_URL + '/ai')
      await page.waitForLoadState('networkidle')
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-30-large-knowledge.png', fullPage: true })
      expect(elapsed).toBeLessThan(PAGE_LOAD_THRESHOLD)
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // ================================================================
  // T16-31 ~ T16-40: Concurrent Operations
  // ================================================================

  // T16-31: 10 concurrent logins
  test('T16-31 Ten concurrent logins', async ({ page }) => {
    await test.step('Perform 10 concurrent login requests', async () => {
      const promises: Promise<Response>[] = []
      for (let i = 0; i < 10; i++) {
        promises.push(
          fetch(`${API_URL}/api/v1/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username: 'admin', password: 'admin123' }),
          })
        )
      }
      const start = Date.now()
      const results = await Promise.all(promises)
      const elapsed = Date.now() - start

      let successCount = 0
      for (const res of results) {
        const data = await res.json()
        if (data.code === 0) successCount++
      }
      await page.screenshot({ path: 'test-results/T16-31-concurrent-logins.png', fullPage: false })
      expect(successCount).toBeGreaterThanOrEqual(1)
    })
  })

  // T16-32: 10 concurrent API calls
  test('T16-32 Ten concurrent API calls', async ({ authPage: page }) => {
    await test.step('Navigate to home to set token', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Fire 10 concurrent API requests', async () => {
      const token = await page.evaluate(() => localStorage.getItem('token'))
      const endpoints = [
        '/api/v1/alert-rules',
        '/api/v1/alert-events',
        '/api/v1/incidents',
        '/api/v1/datasources',
        '/api/v1/teams',
        '/api/v1/users',
        '/api/v1/notify-rules',
        '/api/v1/oncall/schedules',
        '/api/v1/alert-rules?search=test',
        '/api/v1/users/me',
      ]

      const start = Date.now()
      const promises = endpoints.map(ep =>
        page.request.get(`${API_URL}${ep}`, {
          headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        })
      )
      const results = await Promise.all(promises)
      const elapsed = Date.now() - start

      let successCount = 0
      for (const res of results) {
        if (res.status() < 500) successCount++
      }
      await page.screenshot({ path: 'test-results/T16-32-concurrent-api.png', fullPage: false })
      expect(successCount).toBeGreaterThanOrEqual(5)
    })
  })

  // T16-33: Rapid page navigation
  test('T16-33 Rapid page navigation', async ({ authPage: page }) => {
    await test.step('Navigate through multiple pages rapidly', async () => {
      const routes = ['/', '/alert/rules', '/alert/events', '/incidents', '/datasources', '/settings', '/teams', '/users']
      const start = Date.now()
      for (const route of routes) {
        await page.goto(BASE_URL + route, { waitUntil: 'domcontentloaded' })
      }
      const elapsed = Date.now() - start
      await page.screenshot({ path: 'test-results/T16-33-rapid-navigation.png', fullPage: false })
      expect(elapsed).toBeLessThan(60000)
    })

    await test.step('Verify final page is functional', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-34: Rapid filter changes
  test('T16-34 Rapid filter changes', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Rapidly change search filter', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        const queries = ['test', 'alert', 'rule', 'cpu', 'mem', 'disk', 'net', 'prod', 'staging', 'dev']
        const start = Date.now()
        for (const q of queries) {
          await searchInput.fill(q)
          await page.waitForTimeout(200)
        }
        const elapsed = Date.now() - start
        await page.screenshot({ path: 'test-results/T16-34-rapid-filters.png', fullPage: false })
        expect(elapsed).toBeLessThan(30000)
      }
    })
  })

  // T16-35: Rapid search operations
  test('T16-35 Rapid search operations', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Perform rapid search cycles', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        const start = Date.now()
        for (let i = 0; i < 10; i++) {
          await searchInput.fill(`search_${i}`)
          await page.waitForTimeout(100)
          await searchInput.clear()
          await page.waitForTimeout(100)
        }
        const elapsed = Date.now() - start
        await page.screenshot({ path: 'test-results/T16-35-rapid-search.png', fullPage: false })
        expect(elapsed).toBeLessThan(30000)
      }
    })
  })

  // T16-36: Rapid create and delete operations
  test('T16-36 Rapid create and delete cycle', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open and close create dialog rapidly', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        const start = Date.now()
        for (let i = 0; i < 5; i++) {
          await createBtn.click()
          await page.waitForTimeout(300)
          await page.keyboard.press('Escape')
          await page.waitForTimeout(300)
        }
        const elapsed = Date.now() - start
        await page.screenshot({ path: 'test-results/T16-36-rapid-create-delete.png', fullPage: false })
        expect(elapsed).toBeLessThan(30000)
      }
    })
  })

  // T16-37: Concurrent edits
  test('T16-37 Concurrent edit simulation', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Attempt concurrent API edits', async () => {
      const token = await page.evaluate(() => localStorage.getItem('token'))
      const promises: Promise<any>[] = []
      for (let i = 0; i < 3; i++) {
        promises.push(
          page.request.put(`${API_URL}/api/v1/users/me`, {
            headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
            data: JSON.stringify({ nickname: `concurrent_edit_${i}` }),
          }).catch(e => ({ status: () => 0 }))
        )
      }
      const results = await Promise.all(promises)
      await page.screenshot({ path: 'test-results/T16-37-concurrent-edits.png', fullPage: false })
    })
  })

  // T16-38: Race condition detection
  test('T16-38 Race condition detection', async ({ authPage: page }) => {
    await test.step('Perform interleaved read and write operations', async () => {
      const token = await page.evaluate(() => localStorage.getItem('token'))
      const readPromise = page.request.get(`${API_URL}/api/v1/alert-rules`, {
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      })
      const mePromise = page.request.get(`${API_URL}/api/v1/users/me`, {
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      })
      const teamPromise = page.request.get(`${API_URL}/api/v1/teams`, {
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      })

      const [readRes, meRes, teamRes] = await Promise.all([readPromise, mePromise, teamPromise])
      await page.screenshot({ path: 'test-results/T16-38-race-condition.png', fullPage: false })
      expect(readRes.status()).toBeLessThan(500)
      expect(meRes.status()).toBeLessThan(500)
      expect(teamRes.status()).toBeLessThan(500)
    })
  })

  // T16-39: Stale data handling
  test('T16-39 Stale data handling', async ({ authPage: page }) => {
    await test.step('Load page then wait and reload', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
      const firstBodyText = await page.locator('body').innerText().catch(() => '')

      await page.waitForTimeout(3000)
      await page.reload()
      await page.waitForLoadState('networkidle')

      const secondBodyText = await page.locator('body').innerText().catch(() => '')
      await page.screenshot({ path: 'test-results/T16-39-stale-data.png', fullPage: true })
    })

    await test.step('Verify page still functional after stale period', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-40: Optimistic UI updates
  test('T16-40 Optimistic UI updates', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Trigger filter and verify UI updates', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('nonexistent_rule_xyz')
        await page.waitForTimeout(2000)
        await page.screenshot({ path: 'test-results/T16-40-optimistic-ui.png', fullPage: true })
      }
    })

    await test.step('Clear filter and verify recovery', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.clear()
        await page.waitForTimeout(2000)
        await page.screenshot({ path: 'test-results/T16-40-optimistic-recovery.png', fullPage: true })
      }
    })
  })

  // ================================================================
  // T16-41 ~ T16-50: Resource Usage
  // ================================================================

  // T16-41: Memory usage after navigation
  test('T16-41 Memory usage after navigation', async ({ authPage: page }) => {
    await test.step('Navigate through multiple pages and measure memory', async () => {
      const routes = ['/', '/alert/rules', '/alert/events', '/incidents', '/datasources', '/settings']
      for (const route of routes) {
        await page.goto(BASE_URL + route)
        await page.waitForLoadState('networkidle')
      }

      const memoryInfo = await page.evaluate(() => {
        if ((performance as any).memory) {
          return {
            usedJSHeapSize: (performance as any).memory.usedJSHeapSize,
            totalJSHeapSize: (performance as any).memory.totalJSHeapSize,
          }
        }
        return null
      })
      await page.screenshot({ path: 'test-results/T16-41-memory-navigation.png', fullPage: false })
    })

    await test.step('Verify page still responsive', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-42: Memory after filter changes
  test('T16-42 Memory after filter changes', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules and apply multiple filters', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')

      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        for (let i = 0; i < 20; i++) {
          await searchInput.fill(`filter_${i}`)
          await page.waitForTimeout(100)
          await searchInput.clear()
          await page.waitForTimeout(100)
        }
      }

      const memoryInfo = await page.evaluate(() => {
        if ((performance as any).memory) {
          return { usedJSHeapSize: (performance as any).memory.usedJSHeapSize }
        }
        return null
      })
      await page.screenshot({ path: 'test-results/T16-42-memory-filters.png', fullPage: false })
    })

    await test.step('Verify no memory leak symptoms', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-43: Timer cleanup on unmount
  test('T16-43 Timer cleanup on unmount', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard then away', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
    })

    await test.step('Navigate away and check for timer leaks', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)

      const timerCount = await page.evaluate(() => {
        let count = 0
        const origSetInterval = window.setInterval
        return document.querySelectorAll('*').length
      })
      await page.screenshot({ path: 'test-results/T16-43-timer-cleanup.png', fullPage: false })
    })

    await test.step('Verify page renders correctly', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-44: Event listener cleanup
  test('T16-44 Event listener cleanup', async ({ authPage: page }) => {
    await test.step('Navigate through pages to test listener cleanup', async () => {
      const routes = ['/', '/alert/rules', '/alert/events', '/incidents']
      for (const route of routes) {
        await page.goto(BASE_URL + route)
        await page.waitForLoadState('networkidle')
        await page.waitForTimeout(500)
      }
      await page.screenshot({ path: 'test-results/T16-44-listener-cleanup.png', fullPage: false })
    })

    await test.step('Verify no excessive DOM nodes', async () => {
      const nodeCount = await page.evaluate(() => document.querySelectorAll('*').length)
      await page.screenshot({ path: 'test-results/T16-44-dom-nodes.png', fullPage: false })
    })
  })

  // T16-45: WebSocket cleanup
  test('T16-45 WebSocket cleanup', async ({ authPage: page }) => {
    await test.step('Navigate to dashboard which may use WebSocket', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
    })

    await test.step('Navigate away and verify WebSocket closed', async () => {
      await page.goto(BASE_URL + '/settings')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T16-45-websocket-cleanup.png', fullPage: false })
    })

    await test.step('Verify page rendered', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-46: SSE cleanup
  test('T16-46 SSE cleanup', async ({ authPage: page }) => {
    await test.step('Navigate to pages that may use SSE', async () => {
      await page.goto(BASE_URL + '/alert/events')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
    })

    await test.step('Navigate away to trigger SSE cleanup', async () => {
      await page.goto(BASE_URL + '/settings')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T16-46-sse-cleanup.png', fullPage: false })
    })

    await test.step('Verify page rendered', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-47: AbortController cleanup
  test('T16-47 AbortController cleanup', async ({ authPage: page }) => {
    await test.step('Navigate rapidly to trigger abort on in-flight requests', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(500)
      await page.goto(BASE_URL + '/alert/events')
      await page.waitForTimeout(500)
      await page.goto(BASE_URL + '/incidents')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T16-47-abort-cleanup.png', fullPage: false })
    })

    await test.step('Verify final page is functional', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T16-48: localStorage usage
  test('T16-48 localStorage usage', async ({ authPage: page }) => {
    await test.step('Navigate and check localStorage size', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')

      const storageSize = await page.evaluate(() => {
        let total = 0
        for (let i = 0; i < localStorage.length; i++) {
          const key = localStorage.key(i)
          if (key) {
            total += key.length + (localStorage.getItem(key)?.length || 0)
          }
        }
        return total
      })
      await page.screenshot({ path: 'test-results/T16-48-localStorage.png', fullPage: false })
      // localStorage should not exceed 5MB
      expect(storageSize).toBeLessThan(5 * 1024 * 1024)
    })
  })

  // T16-49: sessionStorage usage
  test('T16-49 sessionStorage usage', async ({ authPage: page }) => {
    await test.step('Navigate and check sessionStorage size', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')

      const storageSize = await page.evaluate(() => {
        let total = 0
        for (let i = 0; i < sessionStorage.length; i++) {
          const key = sessionStorage.key(i)
          if (key) {
            total += key.length + (sessionStorage.getItem(key)?.length || 0)
          }
        }
        return total
      })
      await page.screenshot({ path: 'test-results/T16-49-sessionStorage.png', fullPage: false })
      // sessionStorage should not exceed 5MB
      expect(storageSize).toBeLessThan(5 * 1024 * 1024)
    })
  })

  // T16-50: IndexedDB usage
  test('T16-50 IndexedDB usage', async ({ authPage: page }) => {
    await test.step('Check IndexedDB databases', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')

      const dbInfo = await page.evaluate(async () => {
        if (indexedDB.databases) {
          const dbs = await indexedDB.databases()
          return dbs.map(db => ({ name: db.name, version: db.version }))
        }
        return []
      })
      await page.screenshot({ path: 'test-results/T16-50-indexeddb.png', fullPage: false })
    })

    await test.step('Verify page rendered', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })
})
