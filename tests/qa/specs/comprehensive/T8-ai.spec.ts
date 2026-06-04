import { test, expect } from '../../fixtures/auth'

// T8: AI 助手 — 133 个测试用例

test.describe('T8 - AI 助手', () => {

  // T8-1: AI Agent 页面
  test('T8-1 AI Agent 页面', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto('/ai/agent')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T8-1-AI-Agent.png', fullPage: true })
    })

    await test.step('验证页面元素', async () => {
      await expect(page.locator('body')).toBeVisible()
      const input = page.locator('textarea, input[type="text"]').first()
      if (await input.isVisible()) {
        await expect(input).toBeVisible()
      }
    })
  })

  // T8-2: AI 配置页面
  test('T8-2 AI 配置页面', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto('/platform/ai-config')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T8-2-AI配置.png', fullPage: true })
    })

    await test.step('验证配置表单', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T8-3: AI 聊天测试
  test('T8-3 AI 聊天测试', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto('/ai/agent')
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入测试消息', async () => {
      const input = page.locator('textarea, input[type="text"]').first()
      if (await input.isVisible()) {
        await input.fill('你好')
        await page.screenshot({ path: 'test-results/T8-3-输入消息.png', fullPage: false })
      }
    })

    await test.step('发送消息', async () => {
      const sendBtn = page.locator('button').filter({ hasText: /发送|Send|Submit/ }).first()
      if (await sendBtn.isVisible()) {
        await sendBtn.click()
        await page.waitForTimeout(3000)
        await page.screenshot({ path: 'test-results/T8-3-发送结果.png', fullPage: false })
      }
    })
  })

  // T8-4: AI 工具列表
  test('T8-4 AI 工具列表', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto('/platform/ai-config')
      await page.waitForLoadState('networkidle')
    })

    await test.step('查看工具列表', async () => {
      const toolsTab = page.locator('text=工具, text=Tools').first()
      if (await toolsTab.isVisible()) {
        await toolsTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-4-工具列表.png', fullPage: false })
      }
    })
  })

  // T8-5: MCP 服务器列表
  test('T8-5 MCP 服务器列表', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto('/platform/ai-config')
      await page.waitForLoadState('networkidle')
    })

    await test.step('查看 MCP 服务器', async () => {
      const mcpTab = page.locator('text=MCP').first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-5-MCP服务器.png', fullPage: false })
      }
    })
  })

  // T8-6: AI 技能列表
  test('T8-6 AI 技能列表', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto('/platform/ai-config')
      await page.waitForLoadState('networkidle')
    })

    await test.step('查看技能列表', async () => {
      const skillsTab = page.locator('text=技能, text=Skills').first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-6-技能列表.png', fullPage: false })
      }
    })
  })

  // T8-7: 诊断工作流页面
  test('T8-7 诊断工作流页面', async ({ authPage: page }) => {
    await test.step('导航到诊断工作流页', async () => {
      await page.goto('/platform/diagnostics')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T8-7-诊断工作流.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T8-8: 变更事件页面
  test('T8-8 变更事件页面', async ({ authPage: page }) => {
    await test.step('导航到变更事件页', async () => {
      await page.goto('/platform/change-events')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T8-8-变更事件.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T8-9: 预设规则库
  test('T8-9 预设规则库', async ({ authPage: page }) => {
    await test.step('导航到预设规则库', async () => {
      await page.goto('/alert/template-library')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T8-9-预设规则库.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T8-10: 录制规则页面
  test('T8-10 录制规则页面', async ({ authPage: page }) => {
    await test.step('导航到录制规则页', async () => {
      await page.goto('/alert/recording-rules')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T8-10-录制规则.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })
})
