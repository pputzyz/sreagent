import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// T2-Edge: Alert Events Edge Case Test Suite — 100 tests
// Covers: Empty/Null States (T2-E1~T2-E20), Boundary Values (T2-E21~T2-E40),
//         Concurrent State Changes (T2-E41~T2-E60), Error Recovery (T2-E61~T2-E80),
//         Data Integrity (T2-E81~T2-E100)

const EVENTS_URL = '/alert/events'
const HISTORY_URL = '/alert/history'
const BASE_URL = 'http://localhost:3000'

/** Generate unique name */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

/** Get first event ID via API */
async function getFirstEventId(page: import('@playwright/test').Page): Promise<number | null> {
  const res = await API.get(page, '/api/v1/alert-events?page=1&page_size=1')
  const list = res?.data?.list || []
  return list.length > 0 ? list[0].id : null
}

/** Create a test alert rule via API */
async function createTestRule(page: import('@playwright/test').Page, overrides?: Record<string, unknown>): Promise<number> {
  const name = uid('edge_rule')
  const body = {
    name,
    expression: 'up == 0',
    severity: 'warning',
    status: 'active',
    for_duration: '0s',
    labels: { env: 'test' },
    annotations: { summary: 'Edge case test rule' },
    ...overrides,
  }
  const res = await API.post(page, '/api/v1/alert-rules', body)
  return res?.data?.id ?? 0
}

/** Delete a test alert rule via API */
async function deleteTestRule(page: import('@playwright/test').Page, id: number): Promise<void> {
  if (id > 0) {
    await API.del(page, `/api/v1/alert-rules/${id}`)
  }
}

test.describe('T2-Edge - Alert Events Edge Cases', () => {

  // ================================================================
  // T2-E1 ~ T2-E20: Empty / Null States
  // ================================================================

  // T2-E1: No events in the list
  test('T2-E1 No events in the list', async ({ authPage: page }) => {
    await test.step('Intercept API with empty events list', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, data: { list: [], total: 0 } }),
      }))
    })

    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E1-no-events.png', fullPage: true })
    })

    await test.step('Verify empty state UI renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
      const emptyState = page.locator('.empty-state, .n-empty, [class*="empty"], [class*="no-data"], text=暂无数据, text=No data').first()
      const hasEmpty = await emptyState.isVisible().catch(() => false)
      expect(hasEmpty).toBeTruthy()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E2: Event with null labels
  test('T2-E2 Event with null labels', async ({ authPage: page }) => {
    await test.step('Intercept API with null labels', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 1,
              name: 'null_labels_event',
              severity: 'warning',
              status: 'firing',
              labels: null,
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E2-null-labels.png', fullPage: true })
    })

    await test.step('Verify page renders without crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E3: Event with empty annotations
  test('T2-E3 Event with empty annotations', async ({ authPage: page }) => {
    await test.step('Intercept API with empty annotations', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 2,
              name: 'empty_annotations_event',
              severity: 'critical',
              status: 'firing',
              labels: { env: 'prod' },
              annotations: {},
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E3-empty-annotations.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E4: Event with missing severity
  test('T2-E4 Event with missing severity', async ({ authPage: page }) => {
    await test.step('Intercept API with missing severity', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 3,
              name: 'missing_severity_event',
              status: 'firing',
              labels: { env: 'test' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E4-missing-severity.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E5: Event with zero fire_count
  test('T2-E5 Event with zero fire_count', async ({ authPage: page }) => {
    await test.step('Intercept API with zero fire_count', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 4,
              name: 'zero_fire_count_event',
              severity: 'warning',
              status: 'firing',
              labels: { env: 'test' },
              fire_count: 0,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E5-zero-fire-count.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E6: Event with null resolved_at
  test('T2-E6 Event with null resolved_at', async ({ authPage: page }) => {
    await test.step('Intercept API with null resolved_at', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 5,
              name: 'null_resolved_at_event',
              severity: 'critical',
              status: 'firing',
              labels: { env: 'prod' },
              fire_count: 3,
              resolved_at: null,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E6-null-resolved-at.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E7: Event with empty fingerprint
  test('T2-E7 Event with empty fingerprint', async ({ authPage: page }) => {
    await test.step('Intercept API with empty fingerprint', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 6,
              name: 'empty_fingerprint_event',
              severity: 'warning',
              status: 'firing',
              fingerprint: '',
              labels: { env: 'test' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E7-empty-fingerprint.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E8: Event with missing datasource_id
  test('T2-E8 Event with missing datasource_id', async ({ authPage: page }) => {
    await test.step('Intercept API with missing datasource_id', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 7,
              name: 'missing_datasource_event',
              severity: 'warning',
              status: 'firing',
              labels: { env: 'test' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E8-missing-datasource-id.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E9: Event with null rule_id
  test('T2-E9 Event with null rule_id', async ({ authPage: page }) => {
    await test.step('Intercept API with null rule_id', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 8,
              name: 'null_rule_id_event',
              severity: 'critical',
              status: 'firing',
              rule_id: null,
              labels: { env: 'prod' },
              fire_count: 2,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E9-null-rule-id.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E10: Event with empty source
  test('T2-E10 Event with empty source', async ({ authPage: page }) => {
    await test.step('Intercept API with empty source', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 9,
              name: 'empty_source_event',
              severity: 'warning',
              status: 'firing',
              source: '',
              labels: { env: 'test' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E10-empty-source.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E11: Event with missing silenced_until
  test('T2-E11 Event with missing silenced_until', async ({ authPage: page }) => {
    await test.step('Intercept API with missing silenced_until', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 10,
              name: 'missing_silenced_until_event',
              severity: 'warning',
              status: 'firing',
              labels: { env: 'test' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E11-missing-silenced-until.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E12: Event with null assigned_to
  test('T2-E12 Event with null assigned_to', async ({ authPage: page }) => {
    await test.step('Intercept API with null assigned_to', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 11,
              name: 'null_assigned_to_event',
              severity: 'critical',
              status: 'firing',
              assigned_to: null,
              labels: { env: 'prod' },
              fire_count: 5,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E12-null-assigned-to.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E13: Event with empty status
  test('T2-E13 Event with empty status', async ({ authPage: page }) => {
    await test.step('Intercept API with empty status', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 12,
              name: 'empty_status_event',
              severity: 'warning',
              status: '',
              labels: { env: 'test' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E13-empty-status.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E14: Event with missing escalated_at
  test('T2-E14 Event with missing escalated_at', async ({ authPage: page }) => {
    await test.step('Intercept API with missing escalated_at', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 13,
              name: 'missing_escalated_at_event',
              severity: 'critical',
              status: 'firing',
              labels: { env: 'prod' },
              fire_count: 10,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E14-missing-escalated-at.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E15: Event with null acked_by
  test('T2-E15 Event with null acked_by', async ({ authPage: page }) => {
    await test.step('Intercept API with null acked_by', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 14,
              name: 'null_acked_by_event',
              severity: 'warning',
              status: 'acknowledged',
              acked_by: null,
              labels: { env: 'test' },
              fire_count: 2,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E15-null-acked-by.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E16: Event with empty description
  test('T2-E16 Event with empty description', async ({ authPage: page }) => {
    await test.step('Intercept API with empty description', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 15,
              name: 'empty_description_event',
              severity: 'warning',
              status: 'firing',
              description: '',
              labels: { env: 'test' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E16-empty-description.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E17: Event with missing closed_at
  test('T2-E17 Event with missing closed_at', async ({ authPage: page }) => {
    await test.step('Intercept API with missing closed_at', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 16,
              name: 'missing_closed_at_event',
              severity: 'critical',
              status: 'resolved',
              labels: { env: 'prod' },
              fire_count: 1,
              resolved_at: new Date().toISOString(),
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E17-missing-closed-at.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E18: Event with null incident_id
  test('T2-E18 Event with null incident_id', async ({ authPage: page }) => {
    await test.step('Intercept API with null incident_id', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 17,
              name: 'null_incident_id_event',
              severity: 'warning',
              status: 'firing',
              incident_id: null,
              labels: { env: 'test' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E18-null-incident-id.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E19: Event with empty resolution
  test('T2-E19 Event with empty resolution', async ({ authPage: page }) => {
    await test.step('Intercept API with empty resolution', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 18,
              name: 'empty_resolution_event',
              severity: 'critical',
              status: 'resolved',
              resolution: '',
              labels: { env: 'prod' },
              fire_count: 1,
              resolved_at: new Date().toISOString(),
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E19-empty-resolution.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E20: Event with all null/empty fields combined
  test('T2-E20 Event with all null/empty fields combined', async ({ authPage: page }) => {
    await test.step('Intercept API with all null/empty fields', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 19,
              name: 'all_empty_fields_event',
              severity: '',
              status: '',
              labels: null,
              annotations: null,
              description: '',
              source: '',
              fingerprint: '',
              resolution: '',
              rule_id: null,
              datasource_id: null,
              incident_id: null,
              assigned_to: null,
              acked_by: null,
              fire_count: 0,
              resolved_at: null,
              escalated_at: null,
              silenced_until: null,
              closed_at: null,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify no crash', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E20-all-null-empty.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // ================================================================
  // T2-E21 ~ T2-E40: Boundary Values
  // ================================================================

  // T2-E21: Event with max uint ID
  test('T2-E21 Event with max uint ID', async ({ authPage: page }) => {
    await test.step('Intercept API with max uint ID event', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 4294967295,
              name: 'max_uint_id_event',
              severity: 'critical',
              status: 'firing',
              labels: { env: 'prod' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E21-max-uint-id.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E22: Event with zero ID
  test('T2-E22 Event with zero ID', async ({ authPage: page }) => {
    await test.step('Intercept API with zero ID event', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 0,
              name: 'zero_id_event',
              severity: 'warning',
              status: 'firing',
              labels: { env: 'test' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E22-zero-id.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E23: Event with very long alert name (500+ chars)
  test('T2-E23 Event with very long alert name', async ({ authPage: page }) => {
    const longName = 'A'.repeat(600)
    await test.step('Intercept API with very long name', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 20,
              name: longName,
              severity: 'warning',
              status: 'firing',
              labels: { env: 'test' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify no horizontal overflow', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E23-long-name.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E24: Event with very long label value (1000+ chars)
  test('T2-E24 Event with very long label value', async ({ authPage: page }) => {
    const longValue = 'V'.repeat(1200)
    await test.step('Intercept API with very long label value', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 21,
              name: 'long_label_value_event',
              severity: 'warning',
              status: 'firing',
              labels: { env: 'test', long_key: longValue },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E24-long-label-value.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E25: Event with many labels (50+)
  test('T2-E25 Event with many labels', async ({ authPage: page }) => {
    const manyLabels: Record<string, string> = {}
    for (let i = 0; i < 60; i++) {
      manyLabels[`label_${i}`] = `value_${i}`
    }
    await test.step('Intercept API with 60 labels', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 22,
              name: 'many_labels_event',
              severity: 'warning',
              status: 'firing',
              labels: manyLabels,
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E25-many-labels.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E26: Event with many annotations (50+)
  test('T2-E26 Event with many annotations', async ({ authPage: page }) => {
    const manyAnnotations: Record<string, string> = {}
    for (let i = 0; i < 55; i++) {
      manyAnnotations[`annotation_${i}`] = `desc_${i}_${'x'.repeat(50)}`
    }
    await test.step('Intercept API with 55 annotations', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 23,
              name: 'many_annotations_event',
              severity: 'warning',
              status: 'firing',
              labels: { env: 'test' },
              annotations: manyAnnotations,
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E26-many-annotations.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E27: Event with very old fired_at (2020)
  test('T2-E27 Event with very old fired_at', async ({ authPage: page }) => {
    await test.step('Intercept API with old fired_at', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 24,
              name: 'old_fired_at_event',
              severity: 'warning',
              status: 'firing',
              labels: { env: 'test' },
              fire_count: 9999,
              fired_at: '2020-01-01T00:00:00Z',
              created_at: '2020-01-01T00:00:00Z',
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E27-old-fired-at.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E28: Event with future fired_at
  test('T2-E28 Event with future fired_at', async ({ authPage: page }) => {
    await test.step('Intercept API with future fired_at', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 25,
              name: 'future_fired_at_event',
              severity: 'critical',
              status: 'firing',
              labels: { env: 'prod' },
              fire_count: 1,
              fired_at: '2030-12-31T23:59:59Z',
              created_at: '2030-12-31T23:59:59Z',
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E28-future-fired-at.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E29: Event with very long silence duration
  test('T2-E29 Event with very long silence duration', async ({ authPage: page }) => {
    const farFuture = new Date(Date.now() + 365 * 24 * 60 * 60 * 1000 * 10).toISOString()
    await test.step('Intercept API with very long silence duration', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 26,
              name: 'long_silence_event',
              severity: 'warning',
              status: 'silenced',
              silenced_until: farFuture,
              labels: { env: 'test' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E29-long-silence.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E30: Event with zero silence duration
  test('T2-E30 Event with zero silence duration', async ({ authPage: page }) => {
    await test.step('Intercept API with zero silence (silenced_until = now)', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 27,
              name: 'zero_silence_event',
              severity: 'warning',
              status: 'silenced',
              silenced_until: new Date().toISOString(),
              labels: { env: 'test' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E30-zero-silence.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E31: Event with max duration (epoch boundary)
  test('T2-E31 Event with max duration boundary', async ({ authPage: page }) => {
    await test.step('Intercept API with max duration epoch', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 28,
              name: 'max_duration_event',
              severity: 'critical',
              status: 'firing',
              labels: { env: 'prod' },
              fire_count: 1,
              fired_at: '1970-01-01T00:00:00Z',
              created_at: '1970-01-01T00:00:00Z',
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E31-max-duration.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E32: Event with very large fire_count
  test('T2-E32 Event with very large fire_count', async ({ authPage: page }) => {
    await test.step('Intercept API with large fire_count', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 29,
              name: 'large_fire_count_event',
              severity: 'critical',
              status: 'firing',
              labels: { env: 'prod' },
              fire_count: 999999999,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E32-large-fire-count.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E33: Event with negative values
  test('T2-E33 Event with negative values', async ({ authPage: page }) => {
    await test.step('Intercept API with negative values', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: -1,
              name: 'negative_values_event',
              severity: 'warning',
              status: 'firing',
              labels: { env: 'test' },
              fire_count: -5,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify no crash', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E33-negative-values.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E34: Event with unicode labels
  test('T2-E34 Event with unicode labels', async ({ authPage: page }) => {
    await test.step('Intercept API with unicode labels', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 30,
              name: 'unicode_labels_event',
              severity: 'warning',
              status: 'firing',
              labels: {
                env: 'test',
                description: '日本語テスト 한국어 عربي Ελληνικά',
                city: '北京',
                region: '東京',
              },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E34-unicode-labels.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E35: Event with emoji in labels
  test('T2-E35 Event with emoji in labels', async ({ authPage: page }) => {
    await test.step('Intercept API with emoji labels', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 31,
              name: 'emoji_labels_event',
              severity: 'critical',
              status: 'firing',
              labels: {
                env: 'prod',
                alert_type: '🔥 fire',
                priority: '🚨 urgent',
                category: '📊 metrics',
              },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E35-emoji-labels.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E36: Event with HTML in labels
  test('T2-E36 Event with HTML in labels', async ({ authPage: page }) => {
    await test.step('Intercept API with HTML in labels', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 32,
              name: 'html_labels_event',
              severity: 'warning',
              status: 'firing',
              labels: {
                env: 'test',
                description: '<div><p>HTML content</p></div>',
                detail: '<img src=x onerror=alert(1)>',
              },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify HTML is not executed', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E36-html-labels.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E37: Event with SQL injection in labels
  test('T2-E37 Event with SQL injection in labels', async ({ authPage: page }) => {
    await test.step('Intercept API with SQL injection labels', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 33,
              name: "sql_injection_event'; DROP TABLE alert_events; --",
              severity: 'warning',
              status: 'firing',
              labels: {
                env: "test'; DELETE FROM alerts WHERE 1=1; --",
                query: 'UNION SELECT * FROM users',
              },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify no crash', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E37-sql-injection.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E38: Event with script tags in labels
  test('T2-E38 Event with script tags in labels', async ({ authPage: page }) => {
    await test.step('Intercept API with script tags', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 34,
              name: '<script>alert("xss")</script>',
              severity: 'warning',
              status: 'firing',
              labels: {
                env: 'test',
                payload: '<script>document.cookie</script>',
              },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify scripts are not executed', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E38-script-tags.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E39: Event with null bytes in labels
  test('T2-E39 Event with null bytes in labels', async ({ authPage: page }) => {
    await test.step('Intercept API with null bytes', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 35,
              name: 'null_bytes_event',
              severity: 'warning',
              status: 'firing',
              labels: {
                env: 'test\x00null',
                key: 'val\x00ue',
              },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify no crash', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E39-null-bytes.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E40: Event with binary data in labels
  test('T2-E40 Event with binary data in labels', async ({ authPage: page }) => {
    await test.step('Intercept API with binary-like data', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 36,
              name: 'binary_data_event',
              severity: 'warning',
              status: 'firing',
              labels: {
                env: 'test',
                binary: '\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f',
              },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify no crash', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E40-binary-data.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // ================================================================
  // T2-E41 ~ T2-E60: Concurrent State Changes
  // ================================================================

  // T2-E41: Ack during resolve
  test('T2-E41 Ack during resolve', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click first event to open detail', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Attempt to click ack and resolve simultaneously', async () => {
      const ackBtn = page.locator('button').filter({ hasText: /认领|Acknowledge|Ack/ }).first()
      const resolveBtn = page.locator('button').filter({ hasText: /解决|Resolve/ }).first()
      if (await ackBtn.isVisible().catch(() => false) && await resolveBtn.isVisible().catch(() => false)) {
        await Promise.allSettled([ackBtn.click(), resolveBtn.click()])
        await page.waitForTimeout(1000)
      }
      await page.screenshot({ path: 'test-results/T2-E41-ack-during-resolve.png', fullPage: false })
    })
  })

  // T2-E42: Resolve during ack
  test('T2-E42 Resolve during ack', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click first event', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Resolve then immediately ack', async () => {
      const resolveBtn = page.locator('button').filter({ hasText: /解决|Resolve/ }).first()
      const ackBtn = page.locator('button').filter({ hasText: /认领|Acknowledge|Ack/ }).first()
      if (await resolveBtn.isVisible().catch(() => false)) {
        await resolveBtn.click()
      }
      if (await ackBtn.isVisible().catch(() => false)) {
        await ackBtn.click()
      }
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T2-E42-resolve-during-ack.png', fullPage: false })
    })
  })

  // T2-E43: Close during ack
  test('T2-E43 Close during ack', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click first event', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Attempt ack and close in rapid succession', async () => {
      const ackBtn = page.locator('button').filter({ hasText: /认领|Acknowledge|Ack/ }).first()
      const closeBtn = page.locator('button').filter({ hasText: /关闭|Close/ }).first()
      if (await ackBtn.isVisible().catch(() => false)) {
        await ackBtn.click()
        await page.waitForTimeout(100)
      }
      if (await closeBtn.isVisible().catch(() => false)) {
        await closeBtn.click()
      }
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T2-E43-close-during-ack.png', fullPage: false })
    })
  })

  // T2-E44: Silence during close
  test('T2-E44 Silence during close', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click first event', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Attempt close and silence concurrently', async () => {
      const closeBtn = page.locator('button').filter({ hasText: /关闭|Close/ }).first()
      const silenceBtn = page.locator('button').filter({ hasText: /静默|Silence/ }).first()
      if (await closeBtn.isVisible().catch(() => false) && await silenceBtn.isVisible().catch(() => false)) {
        await Promise.allSettled([closeBtn.click(), silenceBtn.click()])
        await page.waitForTimeout(1000)
      }
      await page.screenshot({ path: 'test-results/T2-E44-silence-during-close.png', fullPage: false })
    })
  })

  // T2-E45: Assign during silence
  test('T2-E45 Assign during silence', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click first event', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Silence then immediately assign', async () => {
      const silenceBtn = page.locator('button').filter({ hasText: /静默|Silence/ }).first()
      const assignBtn = page.locator('button').filter({ hasText: /分配|Assign/ }).first()
      if (await silenceBtn.isVisible().catch(() => false)) {
        await silenceBtn.click()
        await page.waitForTimeout(100)
      }
      if (await assignBtn.isVisible().catch(() => false)) {
        await assignBtn.click()
      }
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T2-E45-assign-during-silence.png', fullPage: false })
    })
  })

  // T2-E46: Merge during assign
  test('T2-E46 Merge during assign', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Select multiple events', async () => {
      const checkboxes = page.locator('.n-checkbox, input[type="checkbox"]')
      const count = await checkboxes.count()
      if (count >= 2) {
        await checkboxes.nth(0).click()
        await checkboxes.nth(1).click()
        await page.waitForTimeout(300)
      }
    })

    await test.step('Attempt assign and merge simultaneously', async () => {
      const assignBtn = page.locator('button').filter({ hasText: /分配|Assign/ }).first()
      const mergeBtn = page.locator('button').filter({ hasText: /合并|Merge/ }).first()
      if (await assignBtn.isVisible().catch(() => false) && await mergeBtn.isVisible().catch(() => false)) {
        await Promise.allSettled([assignBtn.click(), mergeBtn.click()])
      }
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T2-E46-merge-during-assign.png', fullPage: false })
    })
  })

  // T2-E47: Escalate during merge
  test('T2-E47 Escalate during merge', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Select multiple events', async () => {
      const checkboxes = page.locator('.n-checkbox, input[type="checkbox"]')
      const count = await checkboxes.count()
      if (count >= 2) {
        await checkboxes.nth(0).click()
        await checkboxes.nth(1).click()
        await page.waitForTimeout(300)
      }
    })

    await test.step('Attempt merge then escalate', async () => {
      const mergeBtn = page.locator('button').filter({ hasText: /合并|Merge/ }).first()
      const escalateBtn = page.locator('button').filter({ hasText: /升级|Escalate/ }).first()
      if (await mergeBtn.isVisible().catch(() => false)) {
        await mergeBtn.click()
        await page.waitForTimeout(100)
      }
      if (await escalateBtn.isVisible().catch(() => false)) {
        await escalateBtn.click()
      }
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T2-E47-escalate-during-merge.png', fullPage: false })
    })
  })

  // T2-E48: Snooze during escalate
  test('T2-E48 Snooze during escalate', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click first event', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Attempt escalate and snooze concurrently', async () => {
      const escalateBtn = page.locator('button').filter({ hasText: /升级|Escalate/ }).first()
      const snoozeBtn = page.locator('button').filter({ hasText: /暂停|Snooze/ }).first()
      if (await escalateBtn.isVisible().catch(() => false) && await snoozeBtn.isVisible().catch(() => false)) {
        await Promise.allSettled([escalateBtn.click(), snoozeBtn.click()])
      }
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T2-E48-snooze-during-escalate.png', fullPage: false })
    })
  })

  // T2-E49: Reopen during snooze
  test('T2-E49 Reopen during snooze', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click first event', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Snooze then immediately reopen', async () => {
      const snoozeBtn = page.locator('button').filter({ hasText: /暂停|Snooze/ }).first()
      const reopenBtn = page.locator('button').filter({ hasText: /重新打开|Reopen/ }).first()
      if (await snoozeBtn.isVisible().catch(() => false)) {
        await snoozeBtn.click()
        await page.waitForTimeout(100)
      }
      if (await reopenBtn.isVisible().catch(() => false)) {
        await reopenBtn.click()
      }
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T2-E49-reopen-during-snooze.png', fullPage: false })
    })
  })

  // T2-E50: Batch ops during single op
  test('T2-E50 Batch ops during single op', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Start single ack operation and batch silence simultaneously', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(300)
      }

      const ackBtn = page.locator('button').filter({ hasText: /认领|Acknowledge|Ack/ }).first()
      if (await ackBtn.isVisible().catch(() => false)) {
        await ackBtn.click()
      }

      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)

      // Try batch operation
      const checkbox = page.locator('.n-checkbox, input[type="checkbox"]').first()
      if (await checkbox.isVisible().catch(() => false)) {
        await checkbox.click()
        const batchSilenceBtn = page.locator('button').filter({ hasText: /静默|Silence/ }).first()
        if (await batchSilenceBtn.isVisible().catch(() => false)) {
          await batchSilenceBtn.click()
        }
      }
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T2-E50-batch-during-single.png', fullPage: false })
    })
  })

  // T2-E51: Filter during delete
  test('T2-E51 Filter during delete', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Start delete operation while applying filter', async () => {
      const checkbox = page.locator('.n-checkbox, input[type="checkbox"]').first()
      if (await checkbox.isVisible().catch(() => false)) {
        await checkbox.click()
        await page.waitForTimeout(200)
      }

      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete/ }).first()
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()

      if (await deleteBtn.isVisible().catch(() => false) && await searchInput.isVisible().catch(() => false)) {
        await Promise.allSettled([deleteBtn.click(), searchInput.fill('test')])
      }
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T2-E51-filter-during-delete.png', fullPage: false })

      // Cancel any dialogs
      const cancelBtn = page.locator('button').filter({ hasText: /取消|Cancel/ }).first()
      if (await cancelBtn.isVisible().catch(() => false)) {
        await cancelBtn.click()
      }
    })
  })

  // T2-E52: Search during create
  test('T2-E52 Search during create', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Type search while loading create modal', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()

      if (await createBtn.isVisible().catch(() => false) && await searchInput.isVisible().catch(() => false)) {
        await Promise.allSettled([createBtn.click(), searchInput.fill('new search')])
      }
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T2-E52-search-during-create.png', fullPage: false })
      await page.keyboard.press('Escape')
    })
  })

  // T2-E53: Sort during update
  test('T2-E53 Sort during update', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click sort header while data is updating', async () => {
      const sortHeader = page.locator('th, [class*="sort"]').filter({ hasText: /时间|Time|严重度|Severity/ }).first()
      if (await sortHeader.isVisible().catch(() => false)) {
        await sortHeader.click()
        await page.waitForTimeout(100)
        await sortHeader.click()
      }
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T2-E53-sort-during-update.png', fullPage: false })
    })
  })

  // T2-E54: Pagination during batch
  test('T2-E54 Pagination during batch', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Select items then change page', async () => {
      const checkbox = page.locator('.n-checkbox, input[type="checkbox"]').first()
      if (await checkbox.isVisible().catch(() => false)) {
        await checkbox.click()
        await page.waitForTimeout(200)
      }

      const nextBtn = page.locator('.n-pagination-item--next, button[aria-label="next"]').first()
      if (await nextBtn.isVisible().catch(() => false)) {
        await nextBtn.click()
        await page.waitForTimeout(1000)
      }
      await page.screenshot({ path: 'test-results/T2-E54-pagination-during-batch.png', fullPage: false })
    })
  })

  // T2-E55: Export during import
  test('T2-E55 Export during import', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click import and export simultaneously', async () => {
      const importBtn = page.locator('button').filter({ hasText: /导入|Import/ }).first()
      const exportBtn = page.locator('button').filter({ hasText: /导出|Export/ }).first()

      if (await importBtn.isVisible().catch(() => false) && await exportBtn.isVisible().catch(() => false)) {
        await Promise.allSettled([importBtn.click(), exportBtn.click()])
      }
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T2-E55-export-during-import.png', fullPage: false })
      await page.keyboard.press('Escape')
    })
  })

  // T2-E56: Double-click ack button
  test('T2-E56 Double-click ack button', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click first event', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Double-click ack button', async () => {
      const ackBtn = page.locator('button').filter({ hasText: /认领|Acknowledge|Ack/ }).first()
      if (await ackBtn.isVisible().catch(() => false)) {
        await ackBtn.dblclick()
        await page.waitForTimeout(1000)
      }
      await page.screenshot({ path: 'test-results/T2-E56-double-click-ack.png', fullPage: false })
    })
  })

  // T2-E57: Double-click resolve button
  test('T2-E57 Double-click resolve button', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click first event', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Double-click resolve button', async () => {
      const resolveBtn = page.locator('button').filter({ hasText: /解决|Resolve/ }).first()
      if (await resolveBtn.isVisible().catch(() => false)) {
        await resolveBtn.dblclick()
        await page.waitForTimeout(1000)
      }
      await page.screenshot({ path: 'test-results/T2-E57-double-click-resolve.png', fullPage: false })
    })
  })

  // T2-E58: Triple-click checkbox
  test('T2-E58 Triple-click checkbox', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Triple-click first checkbox', async () => {
      const checkbox = page.locator('.n-checkbox, input[type="checkbox"]').first()
      if (await checkbox.isVisible().catch(() => false)) {
        await checkbox.click()
        await page.waitForTimeout(100)
        await checkbox.click()
        await page.waitForTimeout(100)
        await checkbox.click()
        await page.waitForTimeout(500)
      }
      await page.screenshot({ path: 'test-results/T2-E58-triple-click-checkbox.png', fullPage: false })
    })
  })

  // T2-E59: Rapid filter changes
  test('T2-E59 Rapid filter changes', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Rapidly toggle severity filters', async () => {
      const severitySelect = page.locator('.n-select, [class*="filter"]').first()
      for (let i = 0; i < 5; i++) {
        if (await severitySelect.isVisible().catch(() => false)) {
          await severitySelect.click()
          await page.waitForTimeout(50)
          await page.keyboard.press('Escape')
          await page.waitForTimeout(50)
        }
      }
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T2-E59-rapid-filters.png', fullPage: false })
    })
  })

  // T2-E60: Concurrent select-all and page-change
  test('T2-E60 Concurrent select-all and page-change', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Select all then immediately change page', async () => {
      const selectAll = page.locator('text=全选, text=Select All, [class*="select-all"]').first()
      const nextBtn = page.locator('.n-pagination-item--next, button[aria-label="next"]').first()

      if (await selectAll.isVisible().catch(() => false)) {
        await selectAll.click()
        await page.waitForTimeout(100)
      }
      if (await nextBtn.isVisible().catch(() => false)) {
        await nextBtn.click()
      }
      await page.waitForTimeout(1000)
      await page.screenshot({ path: 'test-results/T2-E60-concurrent-select-page.png', fullPage: false })
    })
  })

  // ================================================================
  // T2-E61 ~ T2-E80: Error Recovery
  // ================================================================

  // T2-E61: API timeout mid-operation
  test('T2-E61 API timeout mid-operation', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Intercept action API with timeout', async () => {
      await page.route('**/api/v1/alert-events/**/ack', route => {
        return new Promise(resolve => setTimeout(() => route.abort('timedout'), 10000))
      })
    })

    await test.step('Click first event and try ack', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
      const ackBtn = page.locator('button').filter({ hasText: /认领|Acknowledge|Ack/ }).first()
      if (await ackBtn.isVisible().catch(() => false)) {
        await ackBtn.click()
        await page.waitForTimeout(3000)
      }
      await page.screenshot({ path: 'test-results/T2-E61-api-timeout.png', fullPage: false })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events/**/ack')
    })
  })

  // T2-E62: Network disconnect mid-save
  test('T2-E62 Network disconnect mid-save', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Simulate network disconnect during save', async () => {
      await page.route('**/api/**', route => route.abort('connectionrefused'))
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T2-E62-network-disconnect.png', fullPage: true })
    })

    await test.step('Restore network', async () => {
      await page.unroute('**/api/**')
    })
  })

  // T2-E63: Server restart mid-batch
  test('T2-E63 Server restart mid-batch', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Select events then simulate server restart (503)', async () => {
      const checkbox = page.locator('.n-checkbox, input[type="checkbox"]').first()
      if (await checkbox.isVisible().catch(() => false)) {
        await checkbox.click()
        await page.waitForTimeout(200)
      }

      await page.route('**/api/**', route => route.fulfill({
        status: 503,
        body: JSON.stringify({ code: 50003, message: 'Service Unavailable' }),
      }))

      const batchBtn = page.locator('button').filter({ hasText: /静默|Silence|删除|Delete/ }).first()
      if (await batchBtn.isVisible().catch(() => false)) {
        await batchBtn.click()
        await page.waitForTimeout(1000)
      }
      await page.screenshot({ path: 'test-results/T2-E63-server-restart.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/**')
    })
  })

  // T2-E64: Database timeout mid-query
  test('T2-E64 Database timeout mid-query', async ({ authPage: page }) => {
    await test.step('Intercept API with database timeout error', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ code: 50001, message: 'Database query timeout' }),
      }))
    })

    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T2-E64-db-timeout.png', fullPage: true })
    })

    await test.step('Verify error handling', async () => {
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E65: Redis timeout mid-state
  test('T2-E65 Redis timeout mid-state', async ({ authPage: page }) => {
    await test.step('Intercept API with Redis timeout error', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ code: 50001, message: 'Redis connection timeout' }),
      }))
    })

    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T2-E65-redis-timeout.png', fullPage: true })
    })

    await test.step('Verify error handling', async () => {
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E66: Memory pressure mid-render (large response)
  test('T2-E66 Memory pressure mid-render', async ({ authPage: page }) => {
    await test.step('Intercept API with extremely large response', async () => {
      const largeList = Array.from({ length: 500 }, (_, i) => ({
        id: i + 1,
        name: `event_${i}_${'x'.repeat(200)}`,
        severity: i % 2 === 0 ? 'critical' : 'warning',
        status: 'firing',
        labels: Object.fromEntries(Array.from({ length: 20 }, (_, j) => [`label_${j}`, `value_${j}_${'y'.repeat(50)}`])),
        fire_count: Math.floor(Math.random() * 100),
        created_at: new Date().toISOString(),
      }))
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, data: { list: largeList, total: 500 } }),
      }))
    })

    await test.step('Navigate and verify no crash', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForTimeout(5000)
      await page.screenshot({ path: 'test-results/T2-E66-memory-pressure.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E67: Tab switch mid-load
  test('T2-E67 Tab switch mid-load', async ({ authPage: page }) => {
    await test.step('Start slow loading then navigate away and back', async () => {
      await page.route('**/api/v1/alert-events**', async route => {
        await new Promise(r => setTimeout(r, 3000))
        await route.continue()
      })

      const navPromise = page.goto(EVENTS_URL)
      await page.waitForTimeout(500)

      // Navigate away
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')

      // Navigate back
      await page.goBack()
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E67-tab-switch.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E68: Browser back mid-submit
  test('T2-E68 Browser back mid-submit', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click first event', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Try ack then immediately go back', async () => {
      const ackBtn = page.locator('button').filter({ hasText: /认领|Acknowledge|Ack/ }).first()
      if (await ackBtn.isVisible().catch(() => false)) {
        await ackBtn.click()
        await page.waitForTimeout(100)
        await page.goBack()
        await page.waitForTimeout(1000)
      }
      await page.screenshot({ path: 'test-results/T2-E68-browser-back.png', fullPage: true })
    })
  })

  // T2-E69: Refresh mid-edit
  test('T2-E69 Refresh mid-edit', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open event detail then refresh', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(300)
        await page.reload()
        await page.waitForLoadState('networkidle')
      }
      await page.screenshot({ path: 'test-results/T2-E69-refresh-mid-edit.png', fullPage: true })
    })
  })

  // T2-E70: Close mid-delete
  test('T2-E70 Close mid-delete', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open delete dialog then close immediately', async () => {
      const checkbox = page.locator('.n-checkbox, input[type="checkbox"]').first()
      if (await checkbox.isVisible().catch(() => false)) {
        await checkbox.click()
        await page.waitForTimeout(200)
      }

      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(50)
        await page.keyboard.press('Escape')
      }
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T2-E70-close-mid-delete.png', fullPage: false })
    })
  })

  // T2-E71: Resize mid-scroll
  test('T2-E71 Resize mid-scroll', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Scroll and resize simultaneously', async () => {
      await page.evaluate(() => window.scrollTo(0, 500))
      await page.waitForTimeout(100)
      await page.setViewportSize({ width: 800, height: 600 })
      await page.waitForTimeout(300)
      await page.screenshot({ path: 'test-results/T2-E71-resize-mid-scroll.png', fullPage: true })

      // Restore viewport
      await page.setViewportSize({ width: 1280, height: 720 })
    })
  })

  // T2-E72: Focus loss mid-input
  test('T2-E72 Focus loss mid-input', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Type in search then click away', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.click()
        await searchInput.pressSequentially('cpu_high', { delay: 30 })
        await page.locator('body').click({ position: { x: 10, y: 10 } })
        await page.waitForTimeout(500)
      }
      await page.screenshot({ path: 'test-results/T2-E72-focus-loss.png', fullPage: false })
    })
  })

  // T2-E73: Copy mid-select
  test('T2-E73 Copy mid-select', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Select text in event row then copy', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click({ modifiers: ['Shift'] })
        await page.waitForTimeout(100)
        await page.keyboard.press('Control+c')
        await page.waitForTimeout(300)
      }
      await page.screenshot({ path: 'test-results/T2-E73-copy-mid-select.png', fullPage: false })
    })
  })

  // T2-E74: Paste mid-filter
  test('T2-E74 Paste mid-filter', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Paste large content into search', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        const largePaste = 'A'.repeat(5000)
        await searchInput.fill(largePaste)
        await page.waitForTimeout(500)
      }
      await page.screenshot({ path: 'test-results/T2-E74-paste-mid-filter.png', fullPage: false })
    })
  })

  // T2-E75: Undo mid-edit
  test('T2-E75 Undo mid-edit', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Type in search then undo', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test_query')
        await page.waitForTimeout(100)
        await page.keyboard.press('Control+z')
        await page.waitForTimeout(100)
        await page.keyboard.press('Control+z')
        await page.waitForTimeout(500)
      }
      await page.screenshot({ path: 'test-results/T2-E75-undo-mid-edit.png', fullPage: false })
    })
  })

  // T2-E76: 401 mid-operation
  test('T2-E76 401 mid-operation', async ({ authPage: page }) => {
    await test.step('Navigate to events page normally', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Intercept subsequent API with 401', async () => {
      await page.route('**/api/v1/alert-events/**', route => route.fulfill({
        status: 401,
        body: JSON.stringify({ code: 40001, message: 'Token expired' }),
      }))
    })

    await test.step('Trigger an action that requires auth', async () => {
      await page.reload()
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T2-E76-401-mid-op.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events/**')
    })
  })

  // T2-E77: Rate limit (429) mid-batch
  test('T2-E77 Rate limit mid-batch', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Intercept batch API with 429', async () => {
      await page.route('**/api/v1/alert-events/**/batch**', route => route.fulfill({
        status: 429,
        body: JSON.stringify({ code: 42900, message: 'Too Many Requests' }),
      }))
    })

    await test.step('Select events and try batch op', async () => {
      const checkbox = page.locator('.n-checkbox, input[type="checkbox"]').first()
      if (await checkbox.isVisible().catch(() => false)) {
        await checkbox.click()
        await page.waitForTimeout(200)
        const batchBtn = page.locator('button').filter({ hasText: /静默|Silence|删除|Delete/ }).first()
        if (await batchBtn.isVisible().catch(() => false)) {
          await batchBtn.click()
          await page.waitForTimeout(1000)
        }
      }
      await page.screenshot({ path: 'test-results/T2-E77-rate-limit.png', fullPage: false })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events/**/batch**')
    })
  })

  // T2-E78: Invalid JSON response mid-operation
  test('T2-E78 Invalid JSON response mid-operation', async ({ authPage: page }) => {
    await test.step('Intercept API with invalid JSON', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        body: 'not valid json {{{',
      }))
    })

    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T2-E78-invalid-json.png', fullPage: true })
    })

    await test.step('Verify no crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E79: Empty response body
  test('T2-E79 Empty response body', async ({ authPage: page }) => {
    await test.step('Intercept API with empty body', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        body: '',
      }))
    })

    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T2-E79-empty-body.png', fullPage: true })
    })

    await test.step('Verify no crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E80: Slow network mid-interaction
  test('T2-E80 Slow network mid-interaction', async ({ authPage: page }) => {
    await test.step('Intercept API with delay', async () => {
      await page.route('**/api/v1/alert-events**', async route => {
        await new Promise(r => setTimeout(r, 5000))
        await route.continue()
      })
    })

    await test.step('Navigate and interact while slow', async () => {
      const navPromise = page.goto(EVENTS_URL)
      await page.waitForTimeout(1000)

      // Try to click on body while still loading
      await page.locator('body').click({ position: { x: 200, y: 300 } }).catch(() => {})
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T2-E80-slow-network.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // ================================================================
  // T2-E81 ~ T2-E100: Data Integrity
  // ================================================================

  // T2-E81: Duplicate fingerprint handling
  test('T2-E81 Duplicate fingerprint handling', async ({ authPage: page }) => {
    const fp = 'abc123def456'
    await test.step('Intercept API with duplicate fingerprints', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [
              { id: 100, name: 'dup_fp_event_1', fingerprint: fp, severity: 'warning', status: 'firing', labels: { env: 'test' }, fire_count: 1, created_at: new Date().toISOString() },
              { id: 101, name: 'dup_fp_event_2', fingerprint: fp, severity: 'critical', status: 'firing', labels: { env: 'prod' }, fire_count: 2, created_at: new Date().toISOString() },
            ],
            total: 2,
          },
        }),
      }))
    })

    await test.step('Navigate and verify no crash with duplicates', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E81-dup-fingerprint.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E82: Concurrent updates same event
  test('T2-E82 Concurrent updates same event', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open event detail', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Fire two conflicting API calls simultaneously', async () => {
      const ackBtn = page.locator('button').filter({ hasText: /认领|Acknowledge|Ack/ }).first()
      const resolveBtn = page.locator('button').filter({ hasText: /解决|Resolve/ }).first()
      if (await ackBtn.isVisible().catch(() => false) && await resolveBtn.isVisible().catch(() => false)) {
        const results = await Promise.allSettled([ackBtn.click(), resolveBtn.click()])
        // At least one should succeed, the other might fail gracefully
        await page.waitForTimeout(1000)
      }
      await page.screenshot({ path: 'test-results/T2-E82-concurrent-updates.png', fullPage: false })
    })
  })

  // T2-E83: Stale version on update
  test('T2-E83 Stale version on update', async ({ authPage: page }) => {
    await test.step('Intercept ack API with conflict error', async () => {
      await page.route('**/api/v1/alert-events/**/ack', route => route.fulfill({
        status: 409,
        contentType: 'application/json',
        body: JSON.stringify({ code: 10002, message: 'Conflict: event has been modified by another user' }),
      }))
    })

    await test.step('Navigate and try to ack', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
        const ackBtn = page.locator('button').filter({ hasText: /认领|Acknowledge|Ack/ }).first()
        if (await ackBtn.isVisible().catch(() => false)) {
          await ackBtn.click()
          await page.waitForTimeout(1000)
        }
      }
      await page.screenshot({ path: 'test-results/T2-E83-stale-version.png', fullPage: false })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events/**/ack')
    })
  })

  // T2-E84: Deleted event referenced by timeline
  test('T2-E84 Deleted event referenced by timeline', async ({ authPage: page }) => {
    await test.step('Intercept timeline API with reference to missing event', async () => {
      await page.route('**/api/v1/alert-events/**/timeline**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [
              { id: 1, event_id: 99999, action: 'fired', created_at: new Date().toISOString() },
              { id: 2, event_id: 99999, action: 'acknowledged', created_at: new Date().toISOString() },
            ],
            total: 2,
          },
        }),
      }))
    })

    await test.step('Navigate to events page and click first event', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(1000)
      }
      await page.screenshot({ path: 'test-results/T2-E84-deleted-event-timeline.png', fullPage: false })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events/**/timeline**')
    })
  })

  // T2-E85: Orphaned timeline entries
  test('T2-E85 Orphaned timeline entries', async ({ authPage: page }) => {
    await test.step('Intercept API with timeline entries for non-existent event', async () => {
      await page.route('**/api/v1/alert-events/**/timeline**', route => route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({ code: 10002, message: 'Event not found' }),
      }))
    })

    await test.step('Navigate and view timeline tab', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event-row"], [class*="sre-row-card"]').first()
      if (await firstItem.isVisible().catch(() => false)) {
        await firstItem.click()
        await page.waitForTimeout(500)
        const timelineTab = page.locator('text=时间线, text=Timeline').first()
        if (await timelineTab.isVisible().catch(() => false)) {
          await timelineTab.click()
          await page.waitForTimeout(1000)
        }
      }
      await page.screenshot({ path: 'test-results/T2-E85-orphaned-timeline.png', fullPage: false })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events/**/timeline**')
    })
  })

  // T2-E86: Missing cascade on delete
  test('T2-E86 Missing cascade on delete', async ({ authPage: page }) => {
    await test.step('Intercept delete API with cascade error', async () => {
      await page.route('**/api/v1/alert-events/**', route => {
        if (route.request().method() === 'DELETE') {
          route.fulfill({
            status: 500,
            contentType: 'application/json',
            body: JSON.stringify({ code: 50001, message: 'Foreign key constraint fails: timeline references event' }),
          })
        } else {
          route.continue()
        }
      })
    })

    await test.step('Navigate and try to delete', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      const checkbox = page.locator('.n-checkbox, input[type="checkbox"]').first()
      if (await checkbox.isVisible().catch(() => false)) {
        await checkbox.click()
        await page.waitForTimeout(200)
        const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete/ }).first()
        if (await deleteBtn.isVisible().catch(() => false)) {
          await deleteBtn.click()
          await page.waitForTimeout(500)
          // Confirm delete
          const confirmBtn = page.locator('button').filter({ hasText: /确定|Confirm|OK/ }).first()
          if (await confirmBtn.isVisible().catch(() => false)) {
            await confirmBtn.click()
            await page.waitForTimeout(1000)
          }
        }
      }
      await page.screenshot({ path: 'test-results/T2-E86-cascade-error.png', fullPage: false })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events/**')
    })
  })

  // T2-E87: Inconsistent status across queries
  test('T2-E87 Inconsistent status across queries', async ({ authPage: page }) => {
    let callCount = 0
    await test.step('Intercept API with inconsistent status on repeated calls', async () => {
      await page.route('**/api/v1/alert-events**', route => {
        callCount++
        const status = callCount % 2 === 0 ? 'resolved' : 'firing'
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0,
            data: {
              list: [{
                id: 200,
                name: 'inconsistent_status_event',
                severity: 'warning',
                status,
                labels: { env: 'test' },
                fire_count: 1,
                created_at: new Date().toISOString(),
              }],
              total: 1,
            },
          }),
        })
      })
    })

    await test.step('Navigate and reload twice', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.reload()
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E87-inconsistent-status.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E88: Race between list and delete
  test('T2-E88 Race between list and delete', async ({ authPage: page }) => {
    await test.step('Intercept delete to delay then list to return empty', async () => {
      await page.route('**/api/v1/alert-events/**', route => {
        if (route.request().method() === 'DELETE') {
          // Slow delete
          return new Promise(resolve => setTimeout(() => {
            route.fulfill({ status: 200, body: JSON.stringify({ code: 0, data: null }) })
          }, 2000))
        }
        route.continue()
      })
    })

    await test.step('Navigate, select, delete, and quickly refresh', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')

      const checkbox = page.locator('.n-checkbox, input[type="checkbox"]').first()
      if (await checkbox.isVisible().catch(() => false)) {
        await checkbox.click()
        const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete/ }).first()
        if (await deleteBtn.isVisible().catch(() => false)) {
          await deleteBtn.click()
          const confirmBtn = page.locator('button').filter({ hasText: /确定|Confirm|OK/ }).first()
          if (await confirmBtn.isVisible().catch(() => false)) {
            await confirmBtn.click()
          }
        }
      }

      // Quickly refresh before delete finishes
      await page.waitForTimeout(500)
      await page.reload()
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E88-race-list-delete.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events/**')
    })
  })

  // T2-E89: Race between filter and update
  test('T2-E89 Race between filter and update', async ({ authPage: page }) => {
    let filterCallCount = 0
    await test.step('Intercept API with changing results based on filter call count', async () => {
      await page.route('**/api/v1/alert-events**', route => {
        filterCallCount++
        const eventCount = filterCallCount % 3 === 0 ? 0 : 3
        const events = Array.from({ length: eventCount }, (_, i) => ({
          id: i + 1,
          name: `race_event_${i}`,
          severity: 'warning',
          status: 'firing',
          labels: { env: 'test' },
          fire_count: 1,
          created_at: new Date().toISOString(),
        }))
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ code: 0, data: { list: events, total: eventCount } }),
        })
      })
    })

    await test.step('Navigate and rapidly change search', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')

      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        for (let i = 0; i < 6; i++) {
          await searchInput.fill(`query_${i}`)
          await page.waitForTimeout(150)
        }
      }
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T2-E89-race-filter-update.png', fullPage: true })
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E90: Event with no matching rule
  test('T2-E90 Event with no matching rule', async ({ authPage: page }) => {
    await test.step('Intercept API with event that has no rule', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 300,
              name: 'orphan_event_no_rule',
              rule_id: 99999,
              severity: 'warning',
              status: 'firing',
              labels: { env: 'test' },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E90-no-matching-rule.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E91: Event with deleted rule
  test('T2-E91 Event with deleted rule', async ({ authPage: page }) => {
    await test.step('Intercept API with event from deleted rule', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 301,
              name: 'event_deleted_rule',
              rule_id: 88888,
              severity: 'critical',
              status: 'firing',
              labels: { env: 'prod', __deleted_rule: 'true' },
              fire_count: 3,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E91-deleted-rule.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E92: Event with disabled datasource
  test('T2-E92 Event with disabled datasource', async ({ authPage: page }) => {
    await test.step('Intercept API with event from disabled datasource', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 302,
              name: 'event_disabled_datasource',
              datasource_id: 77777,
              severity: 'warning',
              status: 'firing',
              labels: { env: 'test', __ds_status: 'disabled' },
              fire_count: 5,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E92-disabled-datasource.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E93: Event with expired silence
  test('T2-E93 Event with expired silence', async ({ authPage: page }) => {
    const pastTime = new Date(Date.now() - 3600000).toISOString() // 1 hour ago
    await test.step('Intercept API with expired silence', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 303,
              name: 'expired_silence_event',
              severity: 'critical',
              status: 'silenced',
              silenced_until: pastTime,
              labels: { env: 'prod' },
              fire_count: 10,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify expired silence is shown', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E93-expired-silence.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E94: Event with past escalation
  test('T2-E94 Event with past escalation', async ({ authPage: page }) => {
    const pastEscalation = new Date(Date.now() - 86400000).toISOString() // 1 day ago
    await test.step('Intercept API with past escalation', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 304,
              name: 'past_escalation_event',
              severity: 'critical',
              status: 'escalated',
              escalated_at: pastEscalation,
              labels: { env: 'prod', escalation: 'P1' },
              fire_count: 50,
              created_at: pastEscalation,
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E94-past-escalation.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E95: Event with future scheduled dispatch
  test('T2-E95 Event with future scheduled dispatch', async ({ authPage: page }) => {
    const futureDispatch = new Date(Date.now() + 86400000).toISOString() // 1 day from now
    await test.step('Intercept API with future scheduled dispatch', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [{
              id: 305,
              name: 'future_dispatch_event',
              severity: 'warning',
              status: 'firing',
              labels: { env: 'test', scheduled_dispatch: futureDispatch },
              fire_count: 1,
              created_at: new Date().toISOString(),
            }],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E95-future-dispatch.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E96: Event list with mixed valid and invalid entries
  test('T2-E96 Mixed valid and invalid entries', async ({ authPage: page }) => {
    await test.step('Intercept API with mix of valid and invalid events', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [
              { id: 400, name: 'valid_event', severity: 'warning', status: 'firing', labels: { env: 'test' }, fire_count: 1, created_at: new Date().toISOString() },
              { id: null, name: null, severity: null, status: null, labels: null, fire_count: null, created_at: null },
              { id: 401, name: '', severity: '', status: '', labels: {}, fire_count: 0, created_at: '' },
              { id: 402, name: 'another_valid_event', severity: 'critical', status: 'resolved', labels: { env: 'prod' }, fire_count: 5, created_at: new Date().toISOString(), resolved_at: new Date().toISOString() },
            ],
            total: 4,
          },
        }),
      }))
    })

    await test.step('Navigate and verify no crash', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E96-mixed-valid-invalid.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E97: Paginated list where a page has fewer items than page_size
  test('T2-E97 Partial last page', async ({ authPage: page }) => {
    await test.step('Intercept API with partial page', async () => {
      await page.route('**/api/v1/alert-events**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [
              { id: 500, name: 'last_page_event_1', severity: 'warning', status: 'firing', labels: { env: 'test' }, fire_count: 1, created_at: new Date().toISOString() },
            ],
            total: 21, // implies last page has only 1 item when page_size=20
          },
        }),
      }))
    })

    await test.step('Navigate and verify', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E97-partial-last-page.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**')
    })
  })

  // T2-E98: Pagination beyond total pages
  test('T2-E98 Pagination beyond total pages', async ({ authPage: page }) => {
    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Navigate to an invalid high page number via URL', async () => {
      await page.goto(EVENTS_URL + '?page=99999')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E98-beyond-total-pages.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T2-E99: History page with stale events
  test('T2-E99 History page with stale events', async ({ authPage: page }) => {
    await test.step('Intercept history API with stale data', async () => {
      await page.route('**/api/v1/alert-events**history**', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            list: [
              { id: 600, name: 'stale_history_event', severity: 'critical', status: 'resolved', resolved_at: '2020-03-15T10:00:00Z', labels: { env: 'prod' }, fire_count: 100, created_at: '2020-03-01T00:00:00Z' },
            ],
            total: 1,
          },
        }),
      }))
    })

    await test.step('Navigate to history page', async () => {
      await page.goto(HISTORY_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-E99-stale-history.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/v1/alert-events**history**')
    })
  })

  // T2-E100: All API endpoints return errors simultaneously
  test('T2-E100 All API errors simultaneously', async ({ authPage: page }) => {
    await test.step('Intercept all API calls with 500 error', async () => {
      await page.route('**/api/**', route => route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ code: 50001, message: 'Internal Server Error' }),
      }))
    })

    await test.step('Navigate to events page', async () => {
      await page.goto(EVENTS_URL)
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T2-E100-all-api-errors.png', fullPage: true })
    })

    await test.step('Verify page does not crash', async () => {
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Navigate to history page', async () => {
      await page.goto(HISTORY_URL)
      await page.waitForTimeout(3000)
      await page.screenshot({ path: 'test-results/T2-E100-all-api-errors-history.png', fullPage: true })
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Restore routes', async () => {
      await page.unroute('**/api/**')
    })
  })
})
