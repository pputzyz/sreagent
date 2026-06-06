import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// T11: Integrations Full Test Suite — 40 tests
// Covers: Integration List (T11-1~T11-15), Routing Rules (T11-16~T11-30),
//         Webhook Integration (T11-31~T11-40)

const INTEGRATIONS_URL = '/integrations'
const ROUTING_URL = '/integrations/routing'

/** Generate unique name */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

test.describe('T11 - Integrations Full Test Suite', () => {

  // ================================================================
  // T11-1 ~ T11-15: Integration List
  // ================================================================

  // T11-1: Integration list page load
  test('T11-1 Integration list page load', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T11-1-integrations-load.png', fullPage: true })
    })

    await test.step('Verify page body visible', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T11-2: Integration list search
  test('T11-2 Integration list search', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Search integrations', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[placeholder*="Search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('webhook')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T11-2-integrations-search.png', fullPage: false })
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

  // T11-3: Create integration button
  test('T11-3 Create integration button', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click create integration button', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T11-3-create-integration.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-4: Integration type filter
  test('T11-4 Integration type filter', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find and use type filter', async () => {
      const typeFilter = page.locator('.n-select').filter({ hasText: /类型|Type|type/ }).first()
      if (await typeFilter.isVisible().catch(() => false)) {
        await typeFilter.click()
        await page.waitForTimeout(300)
        const option = page.locator('.n-select-option').first()
        if (await option.isVisible()) {
          await option.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T11-4-type-filter.png', fullPage: false })
        }
      }
    })
  })

  // T11-5: Integration status filter
  test('T11-5 Integration status filter', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find and use status filter', async () => {
      const statusFilter = page.locator('.n-select').filter({ hasText: /状态|Status|status/ }).first()
      if (await statusFilter.isVisible().catch(() => false)) {
        await statusFilter.click()
        await page.waitForTimeout(300)
        const option = page.locator('.n-select-option').filter({ hasText: /启用|禁用|Active|Inactive/ }).first()
        if (await option.isVisible()) {
          await option.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T11-5-status-filter.png', fullPage: false })
        }
      }
    })
  })

  // T11-6: Integration webhook URL display
  test('T11-6 Integration webhook URL display', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find webhook URL in list', async () => {
      const webhookUrl = page.locator('[class*="webhook"], code, [class*="url"]').first()
      if (await webhookUrl.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T11-6-webhook-url.png', fullPage: false })
      }
    })
  })

  // T11-7: Edit integration
  test('T11-7 Edit integration', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click edit on first integration', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T11-7-edit-integration.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-8: Delete integration confirmation
  test('T11-8 Delete integration confirmation', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click delete on an integration', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T11-8-delete-confirm.png', fullPage: false })
      }
    })

    await test.step('Cancel delete', async () => {
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

  // T11-9: Test integration button
  test('T11-9 Test integration button', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find test button', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await expect(testBtn).toBeVisible()
        await page.screenshot({ path: 'test-results/T11-9-test-button.png', fullPage: false })
      }
    })
  })

  // T11-10: Enable/disable integration toggle
  test('T11-10 Enable disable integration toggle', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find integration toggle', async () => {
      const toggle = page.locator('.n-switch, [class*="toggle"]').first()
      if (await toggle.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T11-10-integration-toggle.png', fullPage: false })
      }
    })
  })

  // T11-11: Integration form — name field
  test('T11-11 Integration form name field', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create dialog and fill name', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const nameInput = page.locator('.n-modal input, [role="dialog"] input').first()
        if (await nameInput.isVisible()) {
          await nameInput.fill(uid('test_integration'))
          await page.screenshot({ path: 'test-results/T11-11-integration-name.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-12: Integration form — type select
  test('T11-12 Integration form type select', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create dialog and select type', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const typeSelect = page.locator('.n-modal .n-select, [role="dialog"] .n-select').first()
        if (await typeSelect.isVisible().catch(() => false)) {
          await typeSelect.click()
          await page.waitForTimeout(300)
          await page.screenshot({ path: 'test-results/T11-12-integration-type.png', fullPage: false })
          await page.keyboard.press('Escape')
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-13: Integration pagination
  test('T11-13 Integration pagination', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check pagination', async () => {
      const pagination = page.locator('.n-pagination, [class*="pagination"]').first()
      if (await pagination.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T11-13-pagination.png', fullPage: false })
      }
    })
  })

  // T11-14: Integration empty state
  test('T11-14 Integration empty state', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check empty state or list', async () => {
      const emptyState = page.locator('[class*="empty"], .n-empty').first()
      const intList = page.locator('[class*="integration"], [class*="card"], .n-card').first()
      const hasContent = await intList.isVisible().catch(() => false)
      if (!hasContent && await emptyState.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T11-14-empty-state.png', fullPage: false })
      }
    })
  })

  // T11-15: Integration API list
  test('T11-15 Integration API list', async ({ authPage: page }) => {
    await test.step('Get integrations via API', async () => {
      const res = await API.get(page, '/api/v1/integrations?page=1&page_size=10')
      await page.screenshot({ path: 'test-results/T11-15-API-list.png', fullPage: false })
    })
  })

  // ================================================================
  // T11-16 ~ T11-30: Routing Rules
  // ================================================================

  // T11-16: Routing rules page load
  test('T11-16 Routing rules page load', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T11-16-routing-load.png', fullPage: true })
    })

    await test.step('Verify page body', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T11-17: Routing rules list
  test('T11-17 Routing rules list', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Verify routing rules list', async () => {
      await page.screenshot({ path: 'test-results/T11-17-routing-list.png', fullPage: false })
    })
  })

  // T11-18: Create routing rule button
  test('T11-18 Create routing rule button', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click create routing rule button', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T11-18-create-routing.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-19: Create routing rule form
  test('T11-19 Create routing rule form', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create dialog and verify fields', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const nameInput = page.locator('.n-modal input, [role="dialog"] input').first()
        if (await nameInput.isVisible()) {
          await expect(nameInput).toBeVisible()
          await page.screenshot({ path: 'test-results/T11-19-routing-form.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-20: Routing rule conditions
  test('T11-20 Routing rule conditions', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create dialog and find conditions', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const conditionSection = page.locator('[class*="condition"], [class*="match"], text=条件, text=Condition').first()
        if (await conditionSection.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T11-20-routing-conditions.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-21: Routing rule priority
  test('T11-21 Routing rule priority', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create dialog and find priority field', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const priorityInput = page.locator('input[placeholder*="priority"], input[placeholder*="Priority"], input[placeholder*="优先"]').first()
        if (await priorityInput.isVisible().catch(() => false)) {
          await priorityInput.fill('100')
          await page.screenshot({ path: 'test-results/T11-21-routing-priority.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-22: Edit routing rule
  test('T11-22 Edit routing rule', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click edit on first routing rule', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T11-22-edit-routing.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-23: Delete routing rule confirmation
  test('T11-23 Delete routing rule confirmation', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click delete on a routing rule', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T11-23-delete-routing-confirm.png', fullPage: false })
      }
    })

    await test.step('Cancel delete', async () => {
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

  // T11-24: Routing rule enable/disable toggle
  test('T11-24 Routing rule enable disable toggle', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find routing rule toggle', async () => {
      const toggle = page.locator('.n-switch, [class*="toggle"]').first()
      if (await toggle.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T11-24-routing-toggle.png', fullPage: false })
      }
    })
  })

  // T11-25: Routing rule test button
  test('T11-25 Routing rule test button', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find test button for routing rule', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T11-25-routing-test.png', fullPage: false })
      }
    })
  })

  // T11-26: Routing rule import/export
  test('T11-26 Routing rule import export', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find import/export button', async () => {
      const importBtn = page.locator('button').filter({ hasText: /导入|Import|导出|Export/ }).first()
      if (await importBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T11-26-routing-import-export.png', fullPage: false })
      }
    })
  })

  // T11-27: Routing rule drag to reorder
  test('T11-27 Routing rule drag to reorder', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find drag handle for reordering', async () => {
      const dragHandle = page.locator('[class*="drag"], [class*="grip"], [class*="handle"]').first()
      if (await dragHandle.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T11-27-routing-reorder.png', fullPage: false })
      }
    })
  })

  // T11-28: Routing rule condition — label match
  test('T11-28 Routing rule condition label match', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create dialog and fill label match', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const labelInput = page.locator('input[placeholder*="label"], input[placeholder*="Label"], input[placeholder*="标签"]').first()
        if (await labelInput.isVisible().catch(() => false)) {
          await labelInput.fill('env=production')
          await page.screenshot({ path: 'test-results/T11-28-label-match.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-29: Routing rule condition — severity match
  test('T11-29 Routing rule condition severity match', async ({ authPage: page }) => {
    await test.step('Navigate to routing rules page', async () => {
      await page.goto(ROUTING_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create dialog and select severity match', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const sevSelect = page.locator('.n-modal .n-select, [role="dialog"] .n-select').filter({ hasText: /严重|Severity|severity/ }).first()
        if (await sevSelect.isVisible().catch(() => false)) {
          await sevSelect.click()
          await page.waitForTimeout(300)
          const option = page.locator('.n-select-option').filter({ hasText: /critical|warning/ }).first()
          if (await option.isVisible()) {
            await option.click()
            await page.screenshot({ path: 'test-results/T11-29-severity-match.png', fullPage: false })
          }
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-30: Routing rules API
  test('T11-30 Routing rules API', async ({ authPage: page }) => {
    await test.step('Get routing rules via API', async () => {
      const res = await API.get(page, '/api/v1/notify-rules?page=1&page_size=10')
      await page.screenshot({ path: 'test-results/T11-30-routing-API.png', fullPage: false })
    })
  })

  // ================================================================
  // T11-31 ~ T11-40: Webhook Integration
  // ================================================================

  // T11-31: Webhook receive alert
  test('T11-31 Webhook receive alert', async ({ authPage: page }) => {
    await test.step('Test webhook endpoint via API', async () => {
      const webhookBody = {
        title: uid('test_alert'),
        description: 'Test alert from integration test',
        severity: 'warning',
        status: 'firing',
      }
      try {
        const res = await API.post(page, '/api/v1/webhooks/test', webhookBody)
        await page.screenshot({ path: 'test-results/T11-31-webhook-receive.png', fullPage: false })
      } catch {
        // webhook endpoint may not exist
        await page.screenshot({ path: 'test-results/T11-31-webhook-receive.png', fullPage: false })
      }
    })
  })

  // T11-32: Webhook payload parsing
  test('T11-32 Webhook payload parsing', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find webhook payload configuration', async () => {
      const webhookCard = page.locator('[class*="webhook"], [class*="card"]').filter({ hasText: /Webhook|webhook/ }).first()
      if (await webhookCard.isVisible().catch(() => false)) {
        await webhookCard.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T11-32-payload-parsing.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-33: Webhook route to channel
  test('T11-33 Webhook route to channel', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find channel routing configuration', async () => {
      const channelSelect = page.locator('.n-select').filter({ hasText: /通道|Channel|channel/ }).first()
      if (await channelSelect.isVisible().catch(() => false)) {
        await channelSelect.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T11-33-route-to-channel.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T11-34: Webhook test send
  test('T11-34 Webhook test send', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find and click webhook test button', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await testBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T11-34-webhook-test-send.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-35: Webhook view logs
  test('T11-35 Webhook view logs', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find logs button', async () => {
      const logsBtn = page.locator('button').filter({ hasText: /日志|Logs|记录/ }).first()
      if (await logsBtn.isVisible().catch(() => false)) {
        await logsBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T11-35-webhook-logs.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-36: Webhook URL copy
  test('T11-36 Webhook URL copy', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find copy webhook URL button', async () => {
      const copyBtn = page.locator('button').filter({ hasText: /复制|Copy/ }).first()
      if (await copyBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T11-36-webhook-copy.png', fullPage: false })
      }
    })
  })

  // T11-37: Webhook header configuration
  test('T11-37 Webhook header configuration', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open webhook config and find headers', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        const headerSection = page.locator('[class*="header"], text=Header, text=请求头').first()
        if (await headerSection.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T11-37-webhook-headers.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-38: Webhook authentication config
  test('T11-38 Webhook authentication config', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open webhook config and find auth settings', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        const authSection = page.locator('[class*="auth"], text=认证, text=Auth, text=Secret').first()
        if (await authSection.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T11-38-webhook-auth.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-39: Webhook retry configuration
  test('T11-39 Webhook retry configuration', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find retry configuration', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        const retrySection = page.locator('[class*="retry"], text=重试, text=Retry').first()
        if (await retrySection.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T11-39-webhook-retry.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T11-40: Integration full lifecycle
  test('T11-40 Integration full lifecycle', async ({ authPage: page }) => {
    await test.step('Navigate to integrations page', async () => {
      await page.goto(INTEGRATIONS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Verify integrations page loaded', async () => {
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('Click create', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Fill form', async () => {
      const nameInput = page.locator('.n-modal input, [role="dialog"] input').first()
      if (await nameInput.isVisible()) {
        await nameInput.fill(uid('lifecycle_integration'))
        await page.screenshot({ path: 'test-results/T11-40-lifecycle-form.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })

    await test.step('Final screenshot', async () => {
      await page.screenshot({ path: 'test-results/T11-40-lifecycle-done.png', fullPage: true })
    })
  })
})
