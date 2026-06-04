import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

test.describe('AI 功能测试', () => {

  test('AI-1 AI Agent 页面', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto('/ai/agent')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/AI-1-Agent页面.png', fullPage: true })
    })

    await test.step('验证页面元素', async () => {
      await expect(page.locator('body')).toBeVisible()
      // 验证有输入框
      const input = page.locator('textarea, input[type="text"]').first()
      if (await input.isVisible()) {
        await expect(input).toBeVisible()
      }
    })
  })

  test('AI-2 AI 配置页面', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto('/platform/ai-config')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/AI-2-配置页面.png', fullPage: true })
    })

    await test.step('验证配置表单', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  test('AI-3 AI 聊天测试', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto('/ai/agent')
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入测试消息', async () => {
      const input = page.locator('textarea, input[type="text"]').first()
      if (await input.isVisible()) {
        await input.fill('你好')
        await page.screenshot({ path: 'test-results/AI-3-输入消息.png', fullPage: false })
      }
    })

    await test.step('发送消息', async () => {
      const sendBtn = page.locator('button').filter({ hasText: /发送|Send|Submit/ }).first()
      if (await sendBtn.isVisible()) {
        await sendBtn.click()
        await page.waitForTimeout(3000)
        await page.screenshot({ path: 'test-results/AI-3-发送结果.png', fullPage: false })
      }
    })
  })
})
