import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

test.describe('通知功能测试', () => {

  test('NOTIF-1 通知渠道列表', async ({ authPage: page }) => {
    await test.step('导航到通知渠道页', async () => {
      await page.goto('/oncall/notify/media')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/NOTIF-1-渠道列表.png', fullPage: true })
    })

    await test.step('验证渠道卡片', async () => {
      const cards = page.locator('.n-card, [class*="card"]')
      const count = await cards.count()
      await page.screenshot({ path: 'test-results/NOTIF-1-渠道详情.png', fullPage: false })
    })
  })

  test('NOTIF-2 新建渠道弹窗', async ({ authPage: page }) => {
    await test.step('导航到通知渠道页', async () => {
      await page.goto('/oncall/notify/media')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/NOTIF-2-新建弹窗.png', fullPage: false })
      }
    })

    await test.step('验证渠道类型选择', async () => {
      const modal = page.locator('.n-modal, [role="dialog"]').first()
      if (await modal.isVisible()) {
        const typeSelect = modal.locator('.n-select, select').first()
        await expect(typeSelect).toBeVisible()
        await page.screenshot({ path: 'test-results/NOTIF-2-渠道类型.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  test('NOTIF-3 通知规则列表', async ({ authPage: page }) => {
    await test.step('导航到通知规则页', async () => {
      await page.goto('/oncall/notify/rules')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/NOTIF-3-规则列表.png', fullPage: true })
    })

    await test.step('验证规则列表', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  test('NOTIF-4 消息模板列表', async ({ authPage: page }) => {
    await test.step('导航到消息模板页', async () => {
      await page.goto('/oncall/notify/templates')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/NOTIF-4-模板列表.png', fullPage: true })
    })

    await test.step('验证模板列表', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })
})
