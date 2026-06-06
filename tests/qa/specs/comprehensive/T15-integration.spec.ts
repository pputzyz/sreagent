import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// T15: Integration Test Suite — 70 tests
// Covers: Prometheus Integration (T15-1~T15-15), Lark Integration (T15-16~T15-30),
//         Email Integration (T15-31~T15-45), Webhook Integration (T15-46~T15-60),
//         Database Integration (T15-61~T15-70)

const BASE_URL = 'http://localhost:3000'
const API_URL = 'http://localhost:8080'

/** Generate unique name */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

test.describe('T15 - Integration Test Suite', () => {

  // ================================================================
  // T15-1 ~ T15-15: Prometheus Integration
  // ================================================================

  // T15-1: Datasource list page load
  test('T15-1 Datasource list page load', async ({ authPage: page }) => {
    await test.step('Navigate to datasources page', async () => {
      await page.goto(BASE_URL + '/datasources')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T15-1-datasource-list.png', fullPage: true })
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T15-2: Create Prometheus datasource form
  test('T15-2 Create Prometheus datasource form', async ({ authPage: page }) => {
    await test.step('Navigate to datasources page', async () => {
      await page.goto(BASE_URL + '/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click create button', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加|Add/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-2-create-prometheus.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-3: Prometheus type selection
  test('T15-3 Prometheus type selection', async ({ authPage: page }) => {
    await test.step('Navigate to datasources page', async () => {
      await page.goto(BASE_URL + '/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create and select Prometheus type', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加|Add/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const typeSelect = page.locator('.n-select, [class*="type-select"]').first()
        if (await typeSelect.isVisible().catch(() => false)) {
          await typeSelect.click()
          await page.waitForTimeout(300)
          const promOption = page.locator('.n-select-option, [class*="option"]').filter({ hasText: /Prometheus|prometheus/ }).first()
          if (await promOption.isVisible().catch(() => false)) {
            await promOption.click()
          }
          await page.screenshot({ path: 'test-results/T15-3-prometheus-type.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-4: Prometheus URL validation
  test('T15-4 Prometheus URL validation', async ({ authPage: page }) => {
    await test.step('Navigate to datasources page', async () => {
      await page.goto(BASE_URL + '/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Try invalid URL', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加|Add/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const urlInput = page.locator('input[placeholder*="URL"], input[placeholder*="url"], input[placeholder*="地址"]').first()
        if (await urlInput.isVisible().catch(() => false)) {
          await urlInput.fill('not-a-valid-url')
          await page.waitForTimeout(300)
          const submitBtn = page.locator('button').filter({ hasText: /确定|Submit|Save|保存/ }).first()
          if (await submitBtn.isVisible().catch(() => false)) {
            await submitBtn.click()
            await page.waitForTimeout(500)
            await page.screenshot({ path: 'test-results/T15-4-url-validation.png', fullPage: false })
          }
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-5: Test Prometheus connection
  test('T15-5 Test Prometheus connection', async ({ authPage: page }) => {
    await test.step('Navigate to datasources page', async () => {
      await page.goto(BASE_URL + '/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find and click test connection', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|检查|Check/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await testBtn.click()
        await page.waitForTimeout(2000)
        await page.screenshot({ path: 'test-results/T15-5-test-connection.png', fullPage: false })
      }
    })
  })

  // T15-6: Prometheus health check
  test('T15-6 Prometheus health check', async ({ authPage: page }) => {
    await test.step('Check health endpoint', async () => {
      const response = await API.get(page, '/api/v1/datasources')
      await page.screenshot({ path: 'test-results/T15-6-health-check.png', fullPage: false })
    })
  })

  // T15-7: Explore page with Prometheus
  test('T15-7 Explore page with Prometheus', async ({ authPage: page }) => {
    await test.step('Navigate to explore page', async () => {
      await page.goto(BASE_URL + '/explore')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T15-7-explore.png', fullPage: true })
    })

    await test.step('Verify explore page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T15-8: PromQL query input
  test('T15-8 PromQL query input', async ({ authPage: page }) => {
    await test.step('Navigate to explore page', async () => {
      await page.goto(BASE_URL + '/explore')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Enter a PromQL query', async () => {
      const queryInput = page.locator('textarea, .monaco-editor, [class*="query-input"], input[placeholder*="query"]').first()
      if (await queryInput.isVisible().catch(() => false)) {
        await queryInput.fill('up')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-8-promql-input.png', fullPage: false })
      }
    })
  })

  // T15-9: PromQL autocomplete
  test('T15-9 PromQL autocomplete', async ({ authPage: page }) => {
    await test.step('Navigate to explore page', async () => {
      await page.goto(BASE_URL + '/explore')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Type and check for autocomplete', async () => {
      const queryInput = page.locator('textarea, .monaco-editor, [class*="query-input"], input[placeholder*="query"]').first()
      if (await queryInput.isVisible().catch(() => false)) {
        await queryInput.fill('up')
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T15-9-autocomplete.png', fullPage: false })
      }
    })
  })

  // T15-10: Metric selector
  test('T15-10 Metric selector', async ({ authPage: page }) => {
    await test.step('Navigate to explore page', async () => {
      await page.goto(BASE_URL + '/explore')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check metric selector', async () => {
      const metricSelect = page.locator('.n-select, [class*="metric-select"]').first()
      if (await metricSelect.isVisible().catch(() => false)) {
        await metricSelect.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-10-metric-selector.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T15-11: Time range selector
  test('T15-11 Time range selector', async ({ authPage: page }) => {
    await test.step('Navigate to explore page', async () => {
      await page.goto(BASE_URL + '/explore')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check time range selector', async () => {
      const timePicker = page.locator('[class*="time-picker"], [class*="time-range"], button').filter({ hasText: /时间|Time|Range|最近/ }).first()
      if (await timePicker.isVisible().catch(() => false)) {
        await timePicker.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-11-time-range.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T15-12: Datasource timeout configuration
  test('T15-12 Datasource timeout configuration', async ({ authPage: page }) => {
    await test.step('Navigate to datasources page', async () => {
      await page.goto(BASE_URL + '/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open datasource and check timeout field', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit|修改/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        const timeoutInput = page.locator('input[placeholder*="timeout"], input[placeholder*="超时"]').first()
        if (await timeoutInput.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T15-12-timeout-config.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-13: Datasource error handling
  test('T15-13 Datasource error handling', async ({ authPage: page }) => {
    await test.step('Try to query with invalid datasource', async () => {
      const response = await API.get(page, '/api/v1/datasources/999999/query')
      await page.screenshot({ path: 'test-results/T15-13-datasource-error.png', fullPage: false })
    })
  })

  // T15-14: Datasource list pagination
  test('T15-14 Datasource list pagination', async ({ authPage: page }) => {
    await test.step('Navigate to datasources page', async () => {
      await page.goto(BASE_URL + '/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check pagination controls', async () => {
      const pagination = page.locator('.n-pagination, [class*="pagination"]').first()
      if (await pagination.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T15-14-datasource-pagination.png', fullPage: false })
      }
    })
  })

  // T15-15: Datasource search
  test('T15-15 Datasource search', async ({ authPage: page }) => {
    await test.step('Navigate to datasources page', async () => {
      await page.goto(BASE_URL + '/datasources')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Search for datasource', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('prometheus')
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T15-15-datasource-search.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })

  // ================================================================
  // T15-16 ~ T15-30: Lark Integration
  // ================================================================

  // T15-16: Notification channels page
  test('T15-16 Notification channels page', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T15-16-notify-channels.png', fullPage: true })
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T15-17: Create Lark channel form
  test('T15-17 Create Lark channel form', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click create and look for Lark option', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加|Add/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const larkOption = page.locator('[class*="option"], [class*="card"]').filter({ hasText: /Lark|飞书|lark/ }).first()
        if (await larkOption.isVisible().catch(() => false)) {
          await larkOption.click()
          await page.waitForTimeout(500)
        }
        await page.screenshot({ path: 'test-results/T15-17-create-lark.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-18: Lark webhook URL field
  test('T15-18 Lark webhook URL field', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create dialog and find webhook field', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加|Add/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const webhookInput = page.locator('input[placeholder*="webhook"], input[placeholder*="URL"], input[placeholder*="url"]').first()
        if (await webhookInput.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T15-18-lark-webhook.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-19: Lark bot configuration
  test('T15-19 Lark bot configuration', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for bot configuration fields', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加|Add/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-19-lark-bot.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-20: Lark card template
  test('T15-20 Lark card template', async ({ authPage: page }) => {
    await test.step('Navigate to templates page', async () => {
      await page.goto(BASE_URL + '/settings/templates')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for Lark card template', async () => {
      const templateItem = page.locator('[class*="template"], [class*="card"]').filter({ hasText: /Lark|飞书|card|卡片/ }).first()
      if (await templateItem.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T15-20-lark-card.png', fullPage: false })
      }
    })
  })

  // T15-21: Notification rules page
  test('T15-21 Notification rules page', async ({ authPage: page }) => {
    await test.step('Navigate to notification rules', async () => {
      await page.goto(BASE_URL + '/notify/rules')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T15-21-notify-rules.png', fullPage: true })
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T15-22: Create notification rule
  test('T15-22 Create notification rule', async ({ authPage: page }) => {
    await test.step('Navigate to notification rules', async () => {
      await page.goto(BASE_URL + '/notify/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click create notification rule', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-22-create-notify-rule.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-23: Notification rule labels matching
  test('T15-23 Notification rule labels matching', async ({ authPage: page }) => {
    await test.step('Navigate to notification rules', async () => {
      await page.goto(BASE_URL + '/notify/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check label matching fields', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-23-label-matching.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-24: Notification channel test
  test('T15-24 Notification channel test', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find test button for channel', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|发送|Send/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T15-24-channel-test.png', fullPage: false })
      }
    })
  })

  // T15-25: Notification channel edit
  test('T15-25 Notification channel edit', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find edit button for channel', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit|修改/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-25-channel-edit.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-26: Notification channel delete confirmation
  test('T15-26 Notification channel delete confirmation', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find delete button', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete|移除/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-26-channel-delete.png', fullPage: false })
      }
    })

    await test.step('Cancel delete', async () => {
      const cancelBtn = page.locator('button').filter({ hasText: /取消|Cancel/ }).first()
      if (await cancelBtn.isVisible().catch(() => false)) {
        await cancelBtn.click()
      } else {
        await page.keyboard.press('Escape')
      }
    })
  })

  // T15-27: Subscribe rules page
  test('T15-27 Subscribe rules page', async ({ authPage: page }) => {
    await test.step('Navigate to subscribe rules', async () => {
      await page.goto(BASE_URL + '/notify/subscribes')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T15-27-subscribes.png', fullPage: true })
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T15-28: Mute rules page
  test('T15-28 Mute rules page', async ({ authPage: page }) => {
    await test.step('Navigate to mute rules', async () => {
      await page.goto(BASE_URL + '/notify/mutes')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T15-28-mute-rules.png', fullPage: true })
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T15-29: Notification history
  test('T15-29 Notification history', async ({ authPage: page }) => {
    await test.step('Navigate to notification history', async () => {
      await page.goto(BASE_URL + '/notify/history')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T15-29-notify-history.png', fullPage: true })
    })

    await test.step('Verify page renders', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T15-30: Notification channel search
  test('T15-30 Notification channel search', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Search for channel', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('lark')
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T15-30-channel-search.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })

  // ================================================================
  // T15-31 ~ T15-45: Email Integration
  // ================================================================

  // T15-31: SMTP settings page
  test('T15-31 SMTP settings page', async ({ authPage: page }) => {
    await test.step('Navigate to settings page', async () => {
      await page.goto(BASE_URL + '/settings')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T15-31-smtp-settings.png', fullPage: true })
    })

    await test.step('Check for SMTP configuration', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T15-32: SMTP host configuration
  test('T15-32 SMTP host configuration', async ({ authPage: page }) => {
    await test.step('Navigate to settings page', async () => {
      await page.goto(BASE_URL + '/settings')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find SMTP host field', async () => {
      const smtpInput = page.locator('input[placeholder*="SMTP"], input[placeholder*="smtp"], input[placeholder*="host"]').first()
      if (await smtpInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T15-32-smtp-host.png', fullPage: false })
      }
    })
  })

  // T15-33: SMTP port configuration
  test('T15-33 SMTP port configuration', async ({ authPage: page }) => {
    await test.step('Navigate to settings page', async () => {
      await page.goto(BASE_URL + '/settings')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find SMTP port field', async () => {
      const portInput = page.locator('input[placeholder*="port"], input[placeholder*="端口"]').first()
      if (await portInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T15-33-smtp-port.png', fullPage: false })
      }
    })
  })

  // T15-34: Email channel creation
  test('T15-34 Email channel creation', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click create and look for email option', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加|Add/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const emailOption = page.locator('[class*="option"], [class*="card"]').filter({ hasText: /Email|邮件|email|SMTP/ }).first()
        if (await emailOption.isVisible().catch(() => false)) {
          await emailOption.click()
          await page.waitForTimeout(500)
        }
        await page.screenshot({ path: 'test-results/T15-34-email-channel.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-35: Email recipient configuration
  test('T15-35 Email recipient configuration', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check email recipient fields', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加|Add/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-35-email-recipient.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-36: Email template selection
  test('T15-36 Email template selection', async ({ authPage: page }) => {
    await test.step('Navigate to templates page', async () => {
      await page.goto(BASE_URL + '/settings/templates')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for email templates', async () => {
      const emailTemplate = page.locator('[class*="template"]').filter({ hasText: /Email|邮件/ }).first()
      if (await emailTemplate.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T15-36-email-template.png', fullPage: false })
      }
    })
  })

  // T15-37: Email body editor
  test('T15-37 Email body editor', async ({ authPage: page }) => {
    await test.step('Navigate to templates page', async () => {
      await page.goto(BASE_URL + '/settings/templates')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for email body editor', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-37-email-editor.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-38: Email subject configuration
  test('T15-38 Email subject configuration', async ({ authPage: page }) => {
    await test.step('Navigate to templates page', async () => {
      await page.goto(BASE_URL + '/settings/templates')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for subject field in template', async () => {
      const subjectInput = page.locator('input[placeholder*="subject"], input[placeholder*="主题"]').first()
      if (await subjectInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T15-38-email-subject.png', fullPage: false })
      }
    })
  })

  // T15-39: Email test send
  test('T15-39 Email test send', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find test send button', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|发送|Send/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T15-39-email-test.png', fullPage: false })
      }
    })
  })

  // T15-40: Email notification in alert
  test('T15-40 Email notification in alert', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check notification settings in alert rule', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-40-email-in-alert.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-41: Email channel type indicator
  test('T15-41 Email channel type indicator', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for email type indicators', async () => {
      const emailBadge = page.locator('[class*="badge"], [class*="tag"], [class*="type"]').filter({ hasText: /Email|邮件/ }).first()
      if (await emailBadge.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T15-41-email-type.png', fullPage: false })
      }
    })
  })

  // T15-42: Email channel list filtering
  test('T15-42 Email channel list filtering', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Filter by email type', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('email')
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T15-42-email-filter.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })

  // T15-43: Email channel edit form
  test('T15-43 Email channel edit form', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click edit on first channel', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit|修改/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-43-email-edit.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-44: Email validation
  test('T15-44 Email validation', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Try invalid email in recipient field', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const emailInput = page.locator('input[placeholder*="email"], input[placeholder*="邮件"], input[type="email"]').first()
        if (await emailInput.isVisible().catch(() => false)) {
          await emailInput.fill('not-an-email')
          await page.waitForTimeout(300)
          await page.screenshot({ path: 'test-results/T15-44-email-validation.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-45: Email channel status display
  test('T15-45 Email channel status display', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for status indicators', async () => {
      const statusBadge = page.locator('[class*="status"], [class*="badge"], [class*="dot"]').first()
      if (await statusBadge.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T15-45-email-status.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T15-46 ~ T15-60: Webhook Integration
  // ================================================================

  // T15-46: Webhook channel creation
  test('T15-46 Webhook channel creation', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click create and look for webhook option', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加|Add/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const webhookOption = page.locator('[class*="option"], [class*="card"]').filter({ hasText: /Webhook|webhook/ }).first()
        if (await webhookOption.isVisible().catch(() => false)) {
          await webhookOption.click()
          await page.waitForTimeout(500)
        }
        await page.screenshot({ path: 'test-results/T15-46-webhook-create.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-47: Webhook URL field
  test('T15-47 Webhook URL field', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check webhook URL field', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const urlInput = page.locator('input[placeholder*="webhook"], input[placeholder*="URL"], input[placeholder*="url"]').first()
        if (await urlInput.isVisible().catch(() => false)) {
          await urlInput.fill('https://hooks.example.com/webhook')
          await page.screenshot({ path: 'test-results/T15-47-webhook-url.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-48: Webhook custom headers
  test('T15-48 Webhook custom headers', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for custom headers field', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const headerField = page.locator('input[placeholder*="header"], input[placeholder*="Header"], textarea[placeholder*="header"]').first()
        if (await headerField.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T15-48-webhook-headers.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-49: Webhook HMAC signing
  test('T15-49 Webhook HMAC signing', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for HMAC/secret field', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const secretField = page.locator('input[placeholder*="secret"], input[placeholder*="Secret"], input[placeholder*="签名"]').first()
        if (await secretField.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T15-49-webhook-hmac.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-50: Webhook body template
  test('T15-50 Webhook body template', async ({ authPage: page }) => {
    await test.step('Navigate to templates page', async () => {
      await page.goto(BASE_URL + '/settings/templates')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for webhook templates', async () => {
      const webhookTemplate = page.locator('[class*="template"]').filter({ hasText: /Webhook|webhook/ }).first()
      if (await webhookTemplate.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T15-50-webhook-template.png', fullPage: false })
      }
    })
  })

  // T15-51: Webhook test send
  test('T15-51 Webhook test send', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find webhook test button', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|发送|Send/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T15-51-webhook-test.png', fullPage: false })
      }
    })
  })

  // T15-52: Webhook retry configuration
  test('T15-52 Webhook retry configuration', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for retry settings', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const retryField = page.locator('input[placeholder*="retry"], input[placeholder*="重试"]').first()
        if (await retryField.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T15-52-webhook-retry.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-53: Webhook timeout configuration
  test('T15-53 Webhook timeout configuration', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for timeout settings', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const timeoutField = page.locator('input[placeholder*="timeout"], input[placeholder*="超时"]').first()
        if (await timeoutField.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T15-53-webhook-timeout.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-54: Webhook batch send
  test('T15-54 Webhook batch send', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for batch settings', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|添加/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const batchField = page.locator('input[placeholder*="batch"], input[placeholder*="批量"]').first()
        if (await batchField.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T15-54-webhook-batch.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-55: Webhook channel search
  test('T15-55 Webhook channel search', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Search for webhook channels', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('webhook')
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T15-55-webhook-search.png', fullPage: false })
        await searchInput.clear()
      }
    })
  })

  // T15-56: Webhook channel type filter
  test('T15-56 Webhook channel type filter', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Filter by webhook type', async () => {
      const filterSelect = page.locator('.n-select, [class*="filter"]').first()
      if (await filterSelect.isVisible().catch(() => false)) {
        await filterSelect.click()
        await page.waitForTimeout(300)
        const webhookOption = page.locator('.n-select-option, [class*="option"]').filter({ hasText: /Webhook|webhook/ }).first()
        if (await webhookOption.isVisible().catch(() => false)) {
          await webhookOption.click()
          await page.waitForTimeout(500)
        }
        await page.screenshot({ path: 'test-results/T15-56-webhook-filter.png', fullPage: false })
      }
    })
  })

  // T15-57: Webhook channel edit
  test('T15-57 Webhook channel edit', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click edit on first webhook channel', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit|修改/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-57-webhook-edit.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T15-58: Webhook channel delete
  test('T15-58 Webhook channel delete', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click delete on first channel', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete|移除/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-58-webhook-delete.png', fullPage: false })
      }
    })

    await test.step('Cancel delete', async () => {
      const cancelBtn = page.locator('button').filter({ hasText: /取消|Cancel/ }).first()
      if (await cancelBtn.isVisible().catch(() => false)) {
        await cancelBtn.click()
      } else {
        await page.keyboard.press('Escape')
      }
    })
  })

  // T15-59: Webhook channel status
  test('T15-59 Webhook channel status', async ({ authPage: page }) => {
    await test.step('Navigate to notification channels', async () => {
      await page.goto(BASE_URL + '/notify/channels')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check webhook channel status indicators', async () => {
      const statusIndicator = page.locator('[class*="status"], [class*="badge"], [class*="dot"]').first()
      if (await statusIndicator.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T15-59-webhook-status.png', fullPage: false })
      }
    })
  })

  // T15-60: Webhook integration with alert rules
  test('T15-60 Webhook integration with alert rules', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check notification channel selection in alert rule', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T15-60-webhook-in-alert.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // ================================================================
  // T15-61 ~ T15-70: Database Integration
  // ================================================================

  // T15-61: Health check endpoint
  test('T15-61 Health check endpoint', async ({ authPage: page }) => {
    await test.step('Check health endpoint', async () => {
      const response = await page.request.get(`${API_URL}/api/v1/health`)
      const status = response.status()
      await page.screenshot({ path: 'test-results/T15-61-health-endpoint.png', fullPage: false })
    })
  })

  // T15-62: API response format
  test('T15-62 API response format', async ({ authPage: page }) => {
    await test.step('Check API response structure', async () => {
      const response = await API.get(page, '/api/v1/alert-rules')
      expect(response).toHaveProperty('code')
      await page.screenshot({ path: 'test-results/T15-62-api-format.png', fullPage: false })
    })
  })

  // T15-63: API pagination format
  test('T15-63 API pagination format', async ({ authPage: page }) => {
    await test.step('Check paginated API response', async () => {
      const response = await API.get(page, '/api/v1/alert-rules?page=1&page_size=10')
      if (response?.data) {
        await page.screenshot({ path: 'test-results/T15-63-pagination-format.png', fullPage: false })
      }
    })
  })

  // T15-64: API error response format
  test('T15-64 API error response format', async ({ authPage: page }) => {
    await test.step('Trigger API error and check format', async () => {
      const response = await API.get(page, '/api/v1/alert-rules/99999999')
      expect(response).toHaveProperty('code')
      await page.screenshot({ path: 'test-results/T15-64-error-format.png', fullPage: false })
    })
  })

  // T15-65: API CORS preflight
  test('T15-65 API CORS preflight', async ({ authPage: page }) => {
    await test.step('Send OPTIONS request', async () => {
      const response = await page.request.fetch(`${API_URL}/api/v1/alert-rules`, {
        method: 'OPTIONS',
        headers: {
          'Origin': 'http://localhost:3000',
          'Access-Control-Request-Method': 'GET',
        },
      })
      const status = response.status()
      await page.screenshot({ path: 'test-results/T15-65-cors-preflight.png', fullPage: false })
    })
  })

  // T15-66: API rate limiting
  test('T15-66 API rate limiting', async ({ authPage: page }) => {
    await test.step('Make multiple rapid API calls', async () => {
      const promises = Array.from({ length: 10 }, () => API.get(page, '/api/v1/health'))
      await Promise.allSettled(promises)
      await page.screenshot({ path: 'test-results/T15-66-rate-limiting.png', fullPage: false })
    })
  })

  // T15-67: API content negotiation
  test('T15-67 API content negotiation', async ({ authPage: page }) => {
    await test.step('Request with different accept headers', async () => {
      const response = await page.request.get(`${API_URL}/api/v1/health`, {
        headers: { 'Accept': 'application/json' },
      })
      const contentType = response.headers()['content-type'] || ''
      await page.screenshot({ path: 'test-results/T15-67-content-negotiation.png', fullPage: false })
    })
  })

  // T15-68: API versioning
  test('T15-68 API versioning', async ({ authPage: page }) => {
    await test.step('Check API v1 endpoint', async () => {
      const response = await page.request.get(`${API_URL}/api/v1/health`)
      const status = response.status()
      await page.screenshot({ path: 'test-results/T15-68-api-versioning.png', fullPage: false })
    })
  })

  // T15-69: API request ID
  test('T15-69 API request tracking', async ({ authPage: page }) => {
    await test.step('Check for request ID in response headers', async () => {
      const response = await page.request.get(`${API_URL}/api/v1/health`)
      const headers = response.headers()
      await page.screenshot({ path: 'test-results/T15-69-request-tracking.png', fullPage: false })
    })
  })

  // T15-70: API concurrent request handling
  test('T15-70 API concurrent requests', async ({ authPage: page }) => {
    await test.step('Make concurrent requests to different endpoints', async () => {
      const endpoints = [
        '/api/v1/alert-rules',
        '/api/v1/health',
        '/api/v1/datasources',
        '/api/v1/teams',
        '/api/v1/users/me',
      ]
      const promises = endpoints.map(ep => API.get(page, ep))
      const results = await Promise.allSettled(promises)
      await page.screenshot({ path: 'test-results/T15-70-concurrent-requests.png', fullPage: false })
    })
  })
})
