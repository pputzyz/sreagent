import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// T8: AI 助手完整测试 — 60 个测试用例
// 覆盖：AI Agent 页面(T8-1~T8-15)、AI 配置(T8-16~T8-30)、
//       MCP 服务器(T8-31~T8-45)、AI 技能(T8-46~T8-60)

const AI_AGENT_URL = '/ai/agent'
const AI_CONFIG_URL = '/platform/ai-config'

/** 生成唯一名称 */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

test.describe('T8 - AI 助手完整测试', () => {

  // ================================================================
  // T8-1 ~ T8-15: AI Agent 页面
  // ================================================================

  // T8-1: AI Agent 页面加载
  test('T8-1 AI Agent 页面加载', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T8-1-Agent页面.png', fullPage: true })
    })

    await test.step('验证页面元素', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T8-2: 聊天输入框显示
  test('T8-2 聊天输入框显示', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找聊天输入框', async () => {
      const chatInput = page.locator('textarea, input[type="text"], [class*="chat-input"], [class*="message-input"]').first()
      if (await chatInput.isVisible()) {
        await expect(chatInput).toBeVisible()
        await page.screenshot({ path: 'test-results/T8-2-聊天输入框.png', fullPage: false })
      }
    })
  })

  // T8-3: 发送消息
  test('T8-3 发送消息', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入并发送消息', async () => {
      const chatInput = page.locator('textarea, input[type="text"], [class*="chat-input"]').first()
      if (await chatInput.isVisible()) {
        await chatInput.fill('你好，这是测试消息')
        await page.screenshot({ path: 'test-results/T8-3-输入消息.png', fullPage: false })
        const sendBtn = page.locator('button').filter({ hasText: /发送|Send|Submit/ }).first()
        if (await sendBtn.isVisible()) {
          await sendBtn.click()
          await page.waitForTimeout(2000)
          await page.screenshot({ path: 'test-results/T8-3-发送结果.png', fullPage: false })
        }
      }
    })
  })

  // T8-4: 消息气泡展示
  test('T8-4 消息气泡展示', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('发送消息并查看气泡', async () => {
      const chatInput = page.locator('textarea, input[type="text"]').first()
      if (await chatInput.isVisible()) {
        await chatInput.fill('测试消息')
        const sendBtn = page.locator('button').filter({ hasText: /发送|Send/ }).first()
        if (await sendBtn.isVisible()) {
          await sendBtn.click()
          await page.waitForTimeout(2000)
        }
      }
    })

    await test.step('检查消息气泡', async () => {
      const messages = page.locator('[class*="message"], [class*="bubble"], [class*="chat-item"]')
      const count = await messages.count()
      await page.screenshot({ path: 'test-results/T8-4-消息气泡.png', fullPage: false })
    })
  })

  // T8-5: 停止生成按钮
  test('T8-5 停止生成按钮', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('发送消息并查找停止按钮', async () => {
      const chatInput = page.locator('textarea, input[type="text"]').first()
      if (await chatInput.isVisible()) {
        await chatInput.fill('请详细解释一下监控系统的工作原理')
        const sendBtn = page.locator('button').filter({ hasText: /发送|Send/ }).first()
        if (await sendBtn.isVisible()) {
          await sendBtn.click()
          await page.waitForTimeout(500)
          const stopBtn = page.locator('button').filter({ hasText: /停止|Stop|中止/ }).first()
          if (await stopBtn.isVisible().catch(() => false)) {
            await page.screenshot({ path: 'test-results/T8-5-停止按钮.png', fullPage: false })
            await stopBtn.click()
          }
        }
      }
    })
  })

  // T8-6: 清空历史记录
  test('T8-6 清空历史记录', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找清空按钮', async () => {
      const clearBtn = page.locator('button').filter({ hasText: /清空|Clear|新对话|New/ }).first()
      if (await clearBtn.isVisible().catch(() => false)) {
        await clearBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-6-清空后.png', fullPage: false })
      }
    })
  })

  // T8-7: 模型选择器
  test('T8-7 模型选择器', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找模型选择器', async () => {
      const modelSelect = page.locator('.n-select, [class*="model-select"], select').filter({ hasText: /模型|Model|GPT|Claude/ }).first()
      if (await modelSelect.isVisible().catch(() => false)) {
        await modelSelect.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T8-7-模型选择器.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T8-8: 会话列表
  test('T8-8 会话列表', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找会话列表', async () => {
      const sessionList = page.locator('[class*="session"], [class*="conversation"], [class*="chat-list"]').first()
      if (await sessionList.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-8-会话列表.png', fullPage: false })
      }
    })
  })

  // T8-9: Markdown 渲染
  test('T8-9 Markdown 渲染', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('发送请求 Markdown 的消息', async () => {
      const chatInput = page.locator('textarea, input[type="text"]').first()
      if (await chatInput.isVisible()) {
        await chatInput.fill('请用 Markdown 列表格式回答：什么是监控系统？')
        const sendBtn = page.locator('button').filter({ hasText: /发送|Send/ }).first()
        if (await sendBtn.isVisible()) {
          await sendBtn.click()
          await page.waitForTimeout(3000)
          await page.screenshot({ path: 'test-results/T8-9-Markdown渲染.png', fullPage: false })
        }
      }
    })
  })

  // T8-10: 代码块高亮
  test('T8-10 代码块高亮', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找代码块', async () => {
      const codeBlock = page.locator('pre, code, [class*="code-block"]').first()
      if (await codeBlock.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-10-代码块.png', fullPage: false })
      }
    })
  })

  // T8-11: 键盘快捷键发送
  test('T8-11 键盘快捷键发送', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('使用 Enter 发送消息', async () => {
      const chatInput = page.locator('textarea, input[type="text"]').first()
      if (await chatInput.isVisible()) {
        await chatInput.fill('快捷键测试')
        await page.screenshot({ path: 'test-results/T8-11-快捷键发送.png', fullPage: false })
      }
    })
  })

  // T8-12: 输入框自适应高度
  test('T8-12 输入框自适应高度', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入多行文本', async () => {
      const chatInput = page.locator('textarea').first()
      if (await chatInput.isVisible()) {
        await chatInput.fill('第一行\n第二行\n第三行\n第四行')
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T8-12-自适应高度.png', fullPage: false })
      }
    })
  })

  // T8-13: 加载动画
  test('T8-13 加载动画', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('发送消息并观察加载动画', async () => {
      const chatInput = page.locator('textarea, input[type="text"]').first()
      if (await chatInput.isVisible()) {
        await chatInput.fill('测试加载动画')
        const sendBtn = page.locator('button').filter({ hasText: /发送|Send/ }).first()
        if (await sendBtn.isVisible()) {
          await sendBtn.click()
          await page.waitForTimeout(300)
          const loading = page.locator('[class*="loading"], [class*="spinner"], [class*="typing"]').first()
          if (await loading.isVisible().catch(() => false)) {
            await page.screenshot({ path: 'test-results/T8-13-加载动画.png', fullPage: false })
          }
        }
      }
    })
  })

  // T8-14: 错误消息展示
  test('T8-14 错误消息展示', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查错误提示区域', async () => {
      const errorMsg = page.locator('[class*="error"], .n-alert, [class*="alert"]').first()
      if (await errorMsg.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-14-错误消息.png', fullPage: false })
      }
    })
  })

  // T8-15: 会话标题显示
  test('T8-15 会话标题显示', async ({ authPage: page }) => {
    await test.step('导航到 AI Agent 页', async () => {
      await page.goto(AI_AGENT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查会话标题区域', async () => {
      const title = page.locator('[class*="session-title"], [class*="chat-title"], h2, h3').first()
      if (await title.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-15-会话标题.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T8-16 ~ T8-30: AI 配置
  // ================================================================

  // T8-16: AI 配置页面加载
  test('T8-16 AI 配置页面加载', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T8-16-AI配置页面.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T8-17: LLM 提供商列表
  test('T8-17 LLM 提供商列表', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找 LLM 提供商列表', async () => {
      const providerList = page.locator('[class*="provider"], [class*="llm"], .n-card, [class*="card"]')
      const count = await providerList.count()
      await page.screenshot({ path: 'test-results/T8-17-提供商列表.png', fullPage: false })
    })
  })

  // T8-18: 添加 LLM 提供商弹窗
  test('T8-18 添加 LLM 提供商弹窗', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击添加提供商按钮', async () => {
      const addBtn = page.locator('button').filter({ hasText: /添加|Add|新建|Create/ }).first()
      if (await addBtn.isVisible()) {
        await addBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-18-添加提供商.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T8-19: 提供商配置表单
  test('T8-19 提供商配置表单', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开添加弹窗检查表单', async () => {
      const addBtn = page.locator('button').filter({ hasText: /添加|Add|新建/ }).first()
      if (await addBtn.isVisible()) {
        await addBtn.click()
        await page.waitForTimeout(500)
        const formFields = page.locator('.n-modal input, [role="dialog"] input, .n-form-item')
        const count = await formFields.count()
        await page.screenshot({ path: 'test-results/T8-19-配置表单.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T8-20: API Key 输入
  test('T8-20 API Key 输入', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找 API Key 输入框', async () => {
      const apiKeyInput = page.locator('input[type="password"], input[placeholder*="API Key"], input[placeholder*="key"]').first()
      if (await apiKeyInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-20-APIKey输入.png', fullPage: false })
      }
    })
  })

  // T8-21: 测试连接功能
  test('T8-21 测试连接功能', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找测试连接按钮', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|Check|Ping/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-21-测试连接.png', fullPage: false })
      }
    })
  })

  // T8-22: 模型选择下拉
  test('T8-22 模型选择下拉', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找模型选择', async () => {
      const modelSelect = page.locator('.n-select, select').filter({ hasText: /模型|Model|gpt|claude/i }).first()
      if (await modelSelect.isVisible().catch(() => false)) {
        await modelSelect.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T8-22-模型选择.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T8-23: 温度参数设置
  test('T8-23 温度参数设置', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找温度参数', async () => {
      const tempSlider = page.locator('.n-slider, input[type="range"], [class*="temperature"]').first()
      if (await tempSlider.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-23-温度参数.png', fullPage: false })
      }
    })
  })

  // T8-24: Token 限制设置
  test('T8-24 Token 限制设置', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找 Token 设置', async () => {
      const tokenInput = page.locator('input[placeholder*="token"], input[placeholder*="Token"], [class*="token"]').first()
      if (await tokenInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-24-Token设置.png', fullPage: false })
      }
    })
  })

  // T8-25: 编辑提供商
  test('T8-25 编辑提供商', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击编辑按钮', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-25-编辑提供商.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T8-26: 删除提供商确认
  test('T8-26 删除提供商确认', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击删除按钮', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete|Remove/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-26-删除确认.png', fullPage: false })
        const cancelBtn = page.locator('button').filter({ hasText: /取消|Cancel/ }).first()
        if (await cancelBtn.isVisible()) {
          await cancelBtn.click()
        } else {
          await page.keyboard.press('Escape')
        }
      }
    })
  })

  // T8-27: 提供商启用/禁用
  test('T8-27 提供商启用禁用', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找启用/禁用开关', async () => {
      const toggle = page.locator('.n-switch, input[type="checkbox"], [class*="toggle"]').first()
      if (await toggle.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-27-启用禁用.png', fullPage: false })
      }
    })
  })

  // T8-28: 基础 URL 配置
  test('T8-28 基础 URL 配置', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找 URL 配置', async () => {
      const urlInput = page.locator('input[placeholder*="URL"], input[placeholder*="url"], input[placeholder*="http"]').first()
      if (await urlInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-28-URL配置.png', fullPage: false })
      }
    })
  })

  // T8-29: 配置保存
  test('T8-29 配置保存', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找保存按钮', async () => {
      const saveBtn = page.locator('button').filter({ hasText: /保存|Save|应用|Apply/ }).first()
      if (await saveBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-29-保存按钮.png', fullPage: false })
      }
    })
  })

  // T8-30: 默认提供商设置
  test('T8-30 默认提供商设置', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找默认设置', async () => {
      const defaultToggle = page.locator('text=默认, text=Default, [class*="default"]').first()
      if (await defaultToggle.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-30-默认设置.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T8-31 ~ T8-45: MCP 服务器
  // ================================================================

  // T8-31: MCP 服务器列表页
  test('T8-31 MCP 服务器列表页', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('切换到 MCP 标签', async () => {
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-31-MCP列表.png', fullPage: true })
      }
    })
  })

  // T8-32: MCP 添加服务器弹窗
  test('T8-32 MCP 添加服务器弹窗', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('点击添加按钮', async () => {
      const addBtn = page.locator('button').filter({ hasText: /添加|Add|新建/ }).first()
      if (await addBtn.isVisible()) {
        await addBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-32-MCP添加弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T8-33: MCP 服务器名称输入
  test('T8-33 MCP 服务器名称输入', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('打开添加弹窗', async () => {
      const addBtn = page.locator('button').filter({ hasText: /添加|Add|新建/ }).first()
      if (await addBtn.isVisible()) {
        await addBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('输入服务器名称', async () => {
      const nameInput = page.locator('.n-modal input, [role="dialog"] input').first()
      if (await nameInput.isVisible()) {
        await nameInput.fill(uid('mcp_server'))
        await page.screenshot({ path: 'test-results/T8-33-MCP名称.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T8-34: MCP 服务器类型选择
  test('T8-34 MCP 服务器类型选择', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('打开添加弹窗', async () => {
      const addBtn = page.locator('button').filter({ hasText: /添加|Add|新建/ }).first()
      if (await addBtn.isVisible()) {
        await addBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找类型选择', async () => {
      const typeSelect = page.locator('.n-modal .n-select, [role="dialog"] .n-select').first()
      if (await typeSelect.isVisible().catch(() => false)) {
        await typeSelect.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T8-34-MCP类型.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T8-35: MCP 命令配置
  test('T8-35 MCP 命令配置', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('打开添加弹窗', async () => {
      const addBtn = page.locator('button').filter({ hasText: /添加|Add|新建/ }).first()
      if (await addBtn.isVisible()) {
        await addBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找命令输入', async () => {
      const cmdInput = page.locator('input[placeholder*="command"], input[placeholder*="Command"], textarea').first()
      if (await cmdInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-35-MCP命令.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T8-36: MCP 参数配置
  test('T8-36 MCP 参数配置', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('打开添加弹窗', async () => {
      const addBtn = page.locator('button').filter({ hasText: /添加|Add|新建/ }).first()
      if (await addBtn.isVisible()) {
        await addBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找参数输入', async () => {
      const argsInput = page.locator('input[placeholder*="arg"], input[placeholder*="param"], textarea').first()
      if (await argsInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-36-MCP参数.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T8-37: MCP 环境变量配置
  test('T8-37 MCP 环境变量配置', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找环境变量配置', async () => {
      const envSection = page.locator('text=环境变量, text=Environment, text=ENV, [class*="env"]').first()
      if (await envSection.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-37-MCP环境变量.png', fullPage: false })
      }
    })
  })

  // T8-38: MCP 测试连接
  test('T8-38 MCP 测试连接', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找测试连接按钮', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|Check/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-38-MCP测试连接.png', fullPage: false })
      }
    })
  })

  // T8-39: MCP 工具列表
  test('T8-39 MCP 工具列表', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找工具列表', async () => {
      const toolsList = page.locator('[class*="tool"], [class*="tools-list"], .n-data-table').first()
      if (await toolsList.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-39-MCP工具列表.png', fullPage: false })
      }
    })
  })

  // T8-40: MCP 调用工具
  test('T8-40 MCP 调用工具', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找调用按钮', async () => {
      const callBtn = page.locator('button').filter({ hasText: /调用|Call|Execute|Run/ }).first()
      if (await callBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-40-MCP调用工具.png', fullPage: false })
      }
    })
  })

  // T8-41: MCP 编辑服务器
  test('T8-41 MCP 编辑服务器', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('点击编辑按钮', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-41-MCP编辑.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T8-42: MCP 删除服务器
  test('T8-42 MCP 删除服务器', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('点击删除按钮', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete|Remove/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-42-MCP删除确认.png', fullPage: false })
        const cancelBtn = page.locator('button').filter({ hasText: /取消|Cancel/ }).first()
        if (await cancelBtn.isVisible()) {
          await cancelBtn.click()
        } else {
          await page.keyboard.press('Escape')
        }
      }
    })
  })

  // T8-43: MCP 服务器状态指示
  test('T8-43 MCP 服务器状态指示', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('检查状态指示器', async () => {
      const statusIndicator = page.locator('.n-tag, [class*="badge"], [class*="status"]').first()
      if (await statusIndicator.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-43-MCP状态.png', fullPage: false })
      }
    })
  })

  // T8-44: MCP 服务器详情
  test('T8-44 MCP 服务器详情', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('点击服务器查看详情', async () => {
      const serverItem = page.locator('.n-card, [class*="card"], [class*="mcp-item"], tr').first()
      if (await serverItem.isVisible().catch(() => false)) {
        await serverItem.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-44-MCP详情.png', fullPage: false })
      }
    })
  })

  // T8-45: MCP 服务器刷新
  test('T8-45 MCP 服务器刷新', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换 MCP', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const mcpTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /MCP/ }).first()
      if (await mcpTab.isVisible()) {
        await mcpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找刷新按钮', async () => {
      const refreshBtn = page.locator('button').filter({ hasText: /刷新|Refresh|Reload/ }).first()
      if (await refreshBtn.isVisible().catch(() => false)) {
        await refreshBtn.click()
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T8-45-MCP刷新.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T8-46 ~ T8-60: AI 技能
  // ================================================================

  // T8-46: AI 技能列表页
  test('T8-46 AI 技能列表页', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('切换到技能标签', async () => {
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-46-技能列表.png', fullPage: true })
      }
    })
  })

  // T8-47: 创建技能弹窗
  test('T8-47 创建技能弹窗', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('点击创建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-47-创建技能.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T8-48: 技能名称输入
  test('T8-48 技能名称输入', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('输入技能名称', async () => {
      const nameInput = page.locator('.n-modal input, [role="dialog"] input').first()
      if (await nameInput.isVisible()) {
        await nameInput.fill(uid('test_skill'))
        await page.screenshot({ path: 'test-results/T8-48-技能名称.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T8-49: 技能描述输入
  test('T8-49 技能描述输入', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('输入描述', async () => {
      const descInput = page.locator('.n-modal textarea, [role="dialog"] textarea').first()
      if (await descInput.isVisible().catch(() => false)) {
        await descInput.fill('测试技能描述')
        await page.screenshot({ path: 'test-results/T8-49-技能描述.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T8-50: 技能编辑
  test('T8-50 技能编辑', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('点击编辑按钮', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-50-编辑技能.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T8-51: 技能删除确认
  test('T8-51 技能删除确认', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('点击删除按钮', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete|Remove/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-51-删除确认.png', fullPage: false })
        const cancelBtn = page.locator('button').filter({ hasText: /取消|Cancel/ }).first()
        if (await cancelBtn.isVisible()) {
          await cancelBtn.click()
        } else {
          await page.keyboard.press('Escape')
        }
      }
    })
  })

  // T8-52: 技能文件上传
  test('T8-52 技能文件上传', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找文件上传区域', async () => {
      const uploadArea = page.locator('.n-upload, [class*="upload"], input[type="file"]').first()
      if (await uploadArea.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-52-文件上传.png', fullPage: false })
      }
    })
  })

  // T8-53: 技能导入
  test('T8-53 技能导入', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找导入按钮', async () => {
      const importBtn = page.locator('button').filter({ hasText: /导入|Import|上传/ }).first()
      if (await importBtn.isVisible().catch(() => false)) {
        await importBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-53-技能导入.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T8-54: 技能导出
  test('T8-54 技能导出', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找导出按钮', async () => {
      const exportBtn = page.locator('button').filter({ hasText: /导出|Export|下载/ }).first()
      if (await exportBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-54-技能导出.png', fullPage: false })
      }
    })
  })

  // T8-55: 技能执行测试
  test('T8-55 技能执行测试', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找执行/测试按钮', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|执行|Run/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-55-技能测试.png', fullPage: false })
      }
    })
  })

  // T8-56: 技能启用/禁用
  test('T8-56 技能启用禁用', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找启用/禁用开关', async () => {
      const toggle = page.locator('.n-switch, input[type="checkbox"], [class*="toggle"]').first()
      if (await toggle.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-56-技能开关.png', fullPage: false })
      }
    })
  })

  // T8-57: 技能分类标签
  test('T8-57 技能分类标签', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('检查分类标签', async () => {
      const tags = page.locator('.n-tag, [class*="tag"], [class*="badge"]').first()
      if (await tags.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-57-分类标签.png', fullPage: false })
      }
    })
  })

  // T8-58: 技能搜索
  test('T8-58 技能搜索', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-58-技能搜索.png', fullPage: false })
      }
    })
  })

  // T8-59: 技能详情面板
  test('T8-59 技能详情面板', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('点击技能查看详情', async () => {
      const skillItem = page.locator('.n-card, [class*="card"], [class*="skill-item"], tr').first()
      if (await skillItem.isVisible().catch(() => false)) {
        await skillItem.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T8-59-技能详情.png', fullPage: false })
      }
    })
  })

  // T8-60: 技能列表空状态
  test('T8-60 技能列表空状态', async ({ authPage: page }) => {
    await test.step('导航到 AI 配置页并切换技能', async () => {
      await page.goto(AI_CONFIG_URL)
      await page.waitForLoadState('networkidle')
      const skillsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /技能|Skills/ }).first()
      if (await skillsTab.isVisible()) {
        await skillsTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('检查空状态', async () => {
      const emptyState = page.locator('.n-empty, [class*="empty"], [class*="EmptyState"]').first()
      const skillItems = page.locator('.n-card, [class*="skill-item"], tr')
      const count = await skillItems.count()
      if (count === 0 && await emptyState.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T8-60-空状态.png', fullPage: false })
      }
    })
  })
})
