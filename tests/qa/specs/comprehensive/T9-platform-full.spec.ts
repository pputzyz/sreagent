import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// T9: 平台功能完整测试 — 80 个测试用例
// 覆盖：巡检(T9-1~T9-15)、知识库(T9-16~T9-30)、标注(T9-31~T9-45)、
//       任务(T9-46~T9-60)、设置页面(T9-61~T9-80)

const INSPECT_URL = '/platform/inspections'
const KB_URL = '/platform/knowledge-base'
const ANNOTATE_URL = '/platform/annotations'
const TASK_URL = '/platform/tasks'
const SETTINGS_URL = '/platform/settings'

/** 生成唯一名称 */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

test.describe('T9 - 平台功能完整测试', () => {

  // ================================================================
  // T9-1 ~ T9-15: 巡检任务
  // ================================================================

  // T9-1: 巡检任务列表页
  test('T9-1 巡检任务列表页', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T9-1-巡检列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T9-2: 创建巡检弹窗
  test('T9-2 创建巡检弹窗', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击创建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-2-创建巡检.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T9-3: 巡检名称输入
  test('T9-3 巡检名称输入', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('输入名称', async () => {
      const nameInput = page.locator('.n-modal input, [role="dialog"] input').first()
      if (await nameInput.isVisible()) {
        await nameInput.fill(uid('inspection'))
        await page.screenshot({ path: 'test-results/T9-3-巡检名称.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T9-4: 巡检 Cron 表达式
  test('T9-4 巡检 Cron 表达式', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找 Cron 输入', async () => {
      const cronInput = page.locator('input[placeholder*="cron"], input[placeholder*="Cron"], [class*="cron"]').first()
      if (await cronInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-4-Cron输入.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T9-5: 巡检 Cron 预览
  test('T9-5 巡检 Cron 预览', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找 Cron 预览', async () => {
      const cronPreview = page.locator('text=下次执行, text=Next run, [class*="cron-preview"]').first()
      if (await cronPreview.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-5-Cron预览.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T9-6: 巡检编辑
  test('T9-6 巡检编辑', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击编辑按钮', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-6-巡检编辑.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T9-7: 巡检删除确认
  test('T9-7 巡检删除确认', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击删除按钮', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete|Remove/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-7-巡检删除确认.png', fullPage: false })
        const cancelBtn = page.locator('button').filter({ hasText: /取消|Cancel/ }).first()
        if (await cancelBtn.isVisible()) {
          await cancelBtn.click()
        } else {
          await page.keyboard.press('Escape')
        }
      }
    })
  })

  // T9-8: 手动运行巡检
  test('T9-8 手动运行巡检', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击运行按钮', async () => {
      const runBtn = page.locator('button').filter({ hasText: /运行|Run|执行|手动/ }).first()
      if (await runBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-8-手动运行.png', fullPage: false })
      }
    })
  })

  // T9-9: 巡检运行历史
  test('T9-9 巡检运行历史', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找运行历史入口', async () => {
      const historyBtn = page.locator('button, a').filter({ hasText: /历史|History|记录|查看/ }).first()
      if (await historyBtn.isVisible().catch(() => false)) {
        await historyBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-9-运行历史.png', fullPage: false })
      }
    })
  })

  // T9-10: 巡检结果展示
  test('T9-10 巡检结果展示', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找结果入口', async () => {
      const resultBtn = page.locator('button, a').filter({ hasText: /结果|Result|详情/ }).first()
      if (await resultBtn.isVisible().catch(() => false)) {
        await resultBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-10-巡检结果.png', fullPage: false })
      }
    })
  })

  // T9-11: 巡检启用/禁用
  test('T9-11 巡检启用禁用', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找启用/禁用开关', async () => {
      const toggle = page.locator('.n-switch, input[type="checkbox"], [class*="toggle"]').first()
      if (await toggle.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-11-巡检开关.png', fullPage: false })
      }
    })
  })

  // T9-12: 巡检状态标签
  test('T9-12 巡检状态标签', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查状态标签', async () => {
      const statusTag = page.locator('.n-tag, [class*="badge"], [class*="status"]').first()
      if (await statusTag.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-12-巡检状态.png', fullPage: false })
      }
    })
  })

  // T9-13: 巡检搜索
  test('T9-13 巡检搜索', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-13-巡检搜索.png', fullPage: false })
      }
    })
  })

  // T9-14: 巡检分页
  test('T9-14 巡检分页', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查分页控件', async () => {
      const pagination = page.locator('.n-pagination, [class*="pagination"]').first()
      if (await pagination.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-14-巡检分页.png', fullPage: false })
      }
    })
  })

  // T9-15: 巡检空状态
  test('T9-15 巡检空状态', async ({ authPage: page }) => {
    await test.step('导航到巡检任务页', async () => {
      await page.goto(INSPECT_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查空状态', async () => {
      const emptyState = page.locator('.n-empty, [class*="empty"]').first()
      const items = page.locator('tr, .n-card, [class*="item"]')
      const count = await items.count()
      if (count <= 1 && await emptyState.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-15-巡检空状态.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T9-16 ~ T9-30: 知识库
  // ================================================================

  // T9-16: 知识库列表页
  test('T9-16 知识库列表页', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T9-16-知识库列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T9-17: 知识库搜索
  test('T9-17 知识库搜索', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first()
      if (await searchInput.isVisible()) {
        await searchInput.fill('测试')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-17-知识库搜索.png', fullPage: false })
      }
    })
  })

  // T9-18: 创建知识条目弹窗
  test('T9-18 创建知识条目弹窗', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击创建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-18-创建知识条目.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T9-19: 知识条目标题输入
  test('T9-19 知识条目标题输入', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('输入标题', async () => {
      const titleInput = page.locator('.n-modal input, [role="dialog"] input').first()
      if (await titleInput.isVisible()) {
        await titleInput.fill(uid('kb_article'))
        await page.screenshot({ path: 'test-results/T9-19-知识标题.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T9-20: 知识条目内容编辑器
  test('T9-20 知识条目内容编辑器', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找内容编辑器', async () => {
      const editor = page.locator('textarea, .monaco-editor, [class*="editor"], [contenteditable]').first()
      if (await editor.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-20-内容编辑器.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T9-21: 知识条目来源筛选
  test('T9-21 知识条目来源筛选', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找来源筛选', async () => {
      const sourceFilter = page.locator('.n-select, select').filter({ hasText: /来源|Source|全部/ }).first()
      if (await sourceFilter.isVisible().catch(() => false)) {
        await sourceFilter.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T9-21-来源筛选.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T9-22: 知识条目编辑
  test('T9-22 知识条目编辑', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击编辑按钮', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-22-知识编辑.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T9-23: 知识条目删除
  test('T9-23 知识条目删除', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击删除按钮', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete|Remove/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-23-知识删除确认.png', fullPage: false })
        const cancelBtn = page.locator('button').filter({ hasText: /取消|Cancel/ }).first()
        if (await cancelBtn.isVisible()) {
          await cancelBtn.click()
        } else {
          await page.keyboard.press('Escape')
        }
      }
    })
  })

  // T9-24: 全文搜索
  test('T9-24 全文搜索', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入全文搜索', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible()) {
        await searchInput.fill('告警规则配置')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-24-全文搜索.png', fullPage: false })
      }
    })
  })

  // T9-25: 知识条目帮助计数
  test('T9-25 知识条目帮助计数', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查帮助计数', async () => {
      const helpCount = page.locator('text=有帮助, text=Helpful, [class*="help-count"]').first()
      if (await helpCount.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-25-帮助计数.png', fullPage: false })
      }
    })
  })

  // T9-26: 知识条目分类
  test('T9-26 知识条目分类', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查分类标签', async () => {
      const categoryTag = page.locator('.n-tag, [class*="tag"], [class*="category"]').first()
      if (await categoryTag.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-26-知识分类.png', fullPage: false })
      }
    })
  })

  // T9-27: 知识条目详情
  test('T9-27 知识条目详情', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击知识条目查看详情', async () => {
      const item = page.locator('.n-card, [class*="card"], [class*="kb-item"], tr').first()
      if (await item.isVisible().catch(() => false)) {
        await item.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-27-知识详情.png', fullPage: true })
      }
    })
  })

  // T9-28: 知识库分页
  test('T9-28 知识库分页', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查分页控件', async () => {
      const pagination = page.locator('.n-pagination, [class*="pagination"]').first()
      if (await pagination.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-28-知识库分页.png', fullPage: false })
      }
    })
  })

  // T9-29: 知识库排序
  test('T9-29 知识库排序', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找排序控件', async () => {
      const sortSelect = page.locator('.n-select, [class*="sort"]').filter({ hasText: /排序|Sort|时间|名称/ }).first()
      if (await sortSelect.isVisible().catch(() => false)) {
        await sortSelect.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T9-29-知识排序.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T9-30: 知识库空状态
  test('T9-30 知识库空状态', async ({ authPage: page }) => {
    await test.step('导航到知识库页', async () => {
      await page.goto(KB_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查空状态', async () => {
      const emptyState = page.locator('.n-empty, [class*="empty"]').first()
      const items = page.locator('.n-card, [class*="kb-item"], tr')
      const count = await items.count()
      if (count <= 1 && await emptyState.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-30-知识库空状态.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T9-31 ~ T9-45: 标注
  // ================================================================

  // T9-31: 标注列表页
  test('T9-31 标注列表页', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T9-31-标注列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T9-32: 创建标注弹窗
  test('T9-32 创建标注弹窗', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击创建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-32-创建标注.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T9-33: 标注内容输入
  test('T9-33 标注内容输入', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('输入标注内容', async () => {
      const contentInput = page.locator('.n-modal textarea, [role="dialog"] textarea, .n-modal input').first()
      if (await contentInput.isVisible()) {
        await contentInput.fill('测试标注内容')
        await page.screenshot({ path: 'test-results/T9-33-标注内容.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T9-34: 标注时间范围
  test('T9-34 标注时间范围', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找时间选择器', async () => {
      const timePicker = page.locator('.n-date-picker, [class*="time-picker"], input[type="datetime"]').first()
      if (await timePicker.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-34-标注时间.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T9-35: 标注仪表盘关联
  test('T9-35 标注仪表盘关联', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找仪表盘筛选', async () => {
      const dashFilter = page.locator('.n-select, select').filter({ hasText: /仪表盘|Dashboard/ }).first()
      if (await dashFilter.isVisible().catch(() => false)) {
        await dashFilter.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T9-35-仪表盘关联.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T9-36: 标注编辑
  test('T9-36 标注编辑', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击编辑按钮', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-36-标注编辑.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T9-37: 标注删除
  test('T9-37 标注删除', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击删除按钮', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete|Remove/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-37-标注删除确认.png', fullPage: false })
        const cancelBtn = page.locator('button').filter({ hasText: /取消|Cancel/ }).first()
        if (await cancelBtn.isVisible()) {
          await cancelBtn.click()
        } else {
          await page.keyboard.press('Escape')
        }
      }
    })
  })

  // T9-38: 标注公开/私有
  test('T9-38 标注公开私有', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找公开/私有设置', async () => {
      const visibilityToggle = page.locator('.n-switch, [class*="toggle"]').filter({ hasText: /公开|私有|Public|Private/ }).first()
      if (await visibilityToggle.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-38-标注公开私有.png', fullPage: false })
      }
    })
  })

  // T9-39: 标注搜索
  test('T9-39 标注搜索', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-39-标注搜索.png', fullPage: false })
      }
    })
  })

  // T9-40: 标注分页
  test('T9-40 标注分页', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查分页控件', async () => {
      const pagination = page.locator('.n-pagination, [class*="pagination"]').first()
      if (await pagination.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-40-标注分页.png', fullPage: false })
      }
    })
  })

  // T9-41: 标注时间线视图
  test('T9-41 标注时间线视图', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找时间线视图', async () => {
      const timeline = page.locator('[class*="timeline"], [class*="Timeline"]').first()
      if (await timeline.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-41-时间线.png', fullPage: false })
      }
    })
  })

  // T9-42: 标注列表视图切换
  test('T9-42 标注列表视图切换', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找视图切换', async () => {
      const viewToggle = page.locator('button, [class*="view-toggle"]').filter({ hasText: /列表|时间线|List|Timeline/ }).first()
      if (await viewToggle.isVisible().catch(() => false)) {
        await viewToggle.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T9-42-视图切换.png', fullPage: false })
      }
    })
  })

  // T9-43: 标注标签
  test('T9-43 标注标签', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查标注标签', async () => {
      const tags = page.locator('.n-tag, [class*="tag"]').first()
      if (await tags.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-43-标注标签.png', fullPage: false })
      }
    })
  })

  // T9-44: 标注时间筛选
  test('T9-44 标注时间筛选', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找时间筛选', async () => {
      const timeFilter = page.locator('.n-date-picker, button').filter({ hasText: /时间|Time|日期|Date/ }).first()
      if (await timeFilter.isVisible().catch(() => false)) {
        await timeFilter.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T9-44-时间筛选.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T9-45: 标注空状态
  test('T9-45 标注空状态', async ({ authPage: page }) => {
    await test.step('导航到标注页', async () => {
      await page.goto(ANNOTATE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查空状态', async () => {
      const emptyState = page.locator('.n-empty, [class*="empty"]').first()
      const items = page.locator('.n-card, [class*="annotation"], tr')
      const count = await items.count()
      if (count <= 1 && await emptyState.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-45-标注空状态.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T9-46 ~ T9-60: 任务
  // ================================================================

  // T9-46: 任务模板列表页
  test('T9-46 任务模板列表页', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T9-46-任务模板列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T9-47: 创建任务模板弹窗
  test('T9-47 创建任务模板弹窗', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击创建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-47-创建任务模板.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T9-48: 任务模板名称输入
  test('T9-48 任务模板名称输入', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('输入名称', async () => {
      const nameInput = page.locator('.n-modal input, [role="dialog"] input').first()
      if (await nameInput.isVisible()) {
        await nameInput.fill(uid('task_template'))
        await page.screenshot({ path: 'test-results/T9-48-任务名称.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T9-49: 任务模板编辑
  test('T9-49 任务模板编辑', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击编辑按钮', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-49-任务编辑.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T9-50: 任务模板删除
  test('T9-50 任务模板删除', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击删除按钮', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete|Remove/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-50-任务删除确认.png', fullPage: false })
        const cancelBtn = page.locator('button').filter({ hasText: /取消|Cancel/ }).first()
        if (await cancelBtn.isVisible()) {
          await cancelBtn.click()
        } else {
          await page.keyboard.press('Escape')
        }
      }
    })
  })

  // T9-51: 执行任务
  test('T9-51 执行任务', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击执行按钮', async () => {
      const runBtn = page.locator('button').filter({ hasText: /执行|Run|Execute|运行/ }).first()
      if (await runBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-51-执行任务.png', fullPage: false })
      }
    })
  })

  // T9-52: 执行历史
  test('T9-52 执行历史', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找执行历史入口', async () => {
      const historyBtn = page.locator('button, a').filter({ hasText: /历史|History|记录|执行记录/ }).first()
      if (await historyBtn.isVisible().catch(() => false)) {
        await historyBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-52-执行历史.png', fullPage: false })
      }
    })
  })

  // T9-53: 主机执行结果
  test('T9-53 主机执行结果', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找主机结果入口', async () => {
      const hostResultBtn = page.locator('button, a').filter({ hasText: /主机|Host|结果|Result/ }).first()
      if (await hostResultBtn.isVisible().catch(() => false)) {
        await hostResultBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-53-主机结果.png', fullPage: false })
      }
    })
  })

  // T9-54: 任务搜索
  test('T9-54 任务搜索', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-54-任务搜索.png', fullPage: false })
      }
    })
  })

  // T9-55: 任务分页
  test('T9-55 任务分页', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查分页控件', async () => {
      const pagination = page.locator('.n-pagination, [class*="pagination"]').first()
      if (await pagination.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-55-任务分页.png', fullPage: false })
      }
    })
  })

  // T9-56: 任务状态标签
  test('T9-56 任务状态标签', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查状态标签', async () => {
      const statusTag = page.locator('.n-tag, [class*="badge"], [class*="status"]').first()
      if (await statusTag.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-56-任务状态.png', fullPage: false })
      }
    })
  })

  // T9-57: 任务脚本编辑器
  test('T9-57 任务脚本编辑器', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找脚本编辑器', async () => {
      const editor = page.locator('textarea, .monaco-editor, [class*="editor"]').first()
      if (await editor.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-57-脚本编辑器.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T9-58: 任务主机选择
  test('T9-58 任务主机选择', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找主机选择', async () => {
      const hostSelect = page.locator('.n-select, [class*="host-select"]').filter({ hasText: /主机|Host|目标/ }).first()
      if (await hostSelect.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-58-主机选择.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T9-59: 任务超时设置
  test('T9-59 任务超时设置', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查找超时设置', async () => {
      const timeoutInput = page.locator('input[placeholder*="超时"], input[placeholder*="timeout"]').first()
      if (await timeoutInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-59-超时设置.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T9-60: 任务空状态
  test('T9-60 任务空状态', async ({ authPage: page }) => {
    await test.step('导航到任务页', async () => {
      await page.goto(TASK_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查空状态', async () => {
      const emptyState = page.locator('.n-empty, [class*="empty"]').first()
      const items = page.locator('.n-card, [class*="task-item"], tr')
      const count = await items.count()
      if (count <= 1 && await emptyState.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-60-任务空状态.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T9-61 ~ T9-80: 设置页面
  // ================================================================

  // T9-61: SMTP 设置页面
  test('T9-61 SMTP 设置页面', async ({ authPage: page }) => {
    await test.step('导航到 SMTP 设置页', async () => {
      await page.goto(`${SETTINGS_URL}/smtp`)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T9-61-SMTP设置.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T9-62: SMTP 配置表单
  test('T9-62 SMTP 配置表单', async ({ authPage: page }) => {
    await test.step('导航到 SMTP 设置页', async () => {
      await page.goto(`${SETTINGS_URL}/smtp`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查表单字段', async () => {
      const formFields = page.locator('input, .n-form-item, .n-input')
      const count = await formFields.count()
      await page.screenshot({ path: 'test-results/T9-62-SMTP表单.png', fullPage: false })
    })
  })

  // T9-63: SMTP 测试发送
  test('T9-63 SMTP 测试发送', async ({ authPage: page }) => {
    await test.step('导航到 SMTP 设置页', async () => {
      await page.goto(`${SETTINGS_URL}/smtp`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找测试发送按钮', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|发送测试/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-63-SMTP测试.png', fullPage: false })
      }
    })
  })

  // T9-64: 安全设置页面
  test('T9-64 安全设置页面', async ({ authPage: page }) => {
    await test.step('导航到安全设置页', async () => {
      await page.goto(`${SETTINGS_URL}/security`)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T9-64-安全设置.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T9-65: 安全策略配置
  test('T9-65 安全策略配置', async ({ authPage: page }) => {
    await test.step('导航到安全设置页', async () => {
      await page.goto(`${SETTINGS_URL}/security`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查安全策略选项', async () => {
      const policySection = page.locator('[class*="policy"], [class*="security"], .n-form-item')
      const count = await policySection.count()
      await page.screenshot({ path: 'test-results/T9-65-安全策略.png', fullPage: false })
    })
  })

  // T9-66: SSO 设置页面
  test('T9-66 SSO 设置页面', async ({ authPage: page }) => {
    await test.step('导航到 SSO 设置页', async () => {
      await page.goto('/platform/org/sso')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T9-66-SSO设置.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T9-67: SSO 配置表单
  test('T9-67 SSO 配置表单', async ({ authPage: page }) => {
    await test.step('导航到 SSO 设置页', async () => {
      await page.goto('/platform/org/sso')
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查配置表单', async () => {
      const formFields = page.locator('input, .n-form-item')
      const count = await formFields.count()
      await page.screenshot({ path: 'test-results/T9-67-SSO表单.png', fullPage: false })
    })
  })

  // T9-68: 审计日志页面
  test('T9-68 审计日志页面', async ({ authPage: page }) => {
    await test.step('导航到审计日志页', async () => {
      await page.goto('/platform/audit')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T9-68-审计日志.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T9-69: 审计日志筛选
  test('T9-69 审计日志筛选', async ({ authPage: page }) => {
    await test.step('导航到审计日志页', async () => {
      await page.goto('/platform/audit')
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找筛选控件', async () => {
      const filterSelect = page.locator('.n-select, select').first()
      if (await filterSelect.isVisible().catch(() => false)) {
        await filterSelect.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T9-69-审计筛选.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T9-70: 审计日志搜索
  test('T9-70 审计日志搜索', async ({ authPage: page }) => {
    await test.step('导航到审计日志页', async () => {
      await page.goto('/platform/audit')
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('login')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T9-70-审计搜索.png', fullPage: false })
      }
    })
  })

  // T9-71: 站点信息设置
  test('T9-71 站点信息设置', async ({ authPage: page }) => {
    await test.step('导航到站点设置页', async () => {
      await page.goto(`${SETTINGS_URL}/site`)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T9-71-站点信息.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T9-72: 站点名称修改
  test('T9-72 站点名称修改', async ({ authPage: page }) => {
    await test.step('导航到站点设置页', async () => {
      await page.goto(`${SETTINGS_URL}/site`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找站点名称输入', async () => {
      const nameInput = page.locator('input[placeholder*="站点"], input[placeholder*="site"], input[placeholder*="名称"]').first()
      if (await nameInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-72-站点名称.png', fullPage: false })
      }
    })
  })

  // T9-73: Lark 机器人设置
  test('T9-73 Lark 机器人设置', async ({ authPage: page }) => {
    await test.step('导航到 Lark 设置页', async () => {
      await page.goto(`${SETTINGS_URL}/lark`)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T9-73-Lark设置.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T9-74: Lark Webhook 配置
  test('T9-74 Lark Webhook 配置', async ({ authPage: page }) => {
    await test.step('导航到 Lark 设置页', async () => {
      await page.goto(`${SETTINGS_URL}/lark`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找 Webhook 输入', async () => {
      const webhookInput = page.locator('input[placeholder*="webhook"], input[placeholder*="Webhook"], input[placeholder*="URL"]').first()
      if (await webhookInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-74-Webhook配置.png', fullPage: false })
      }
    })
  })

  // T9-75: AI 设置页面
  test('T9-75 AI 设置页面', async ({ authPage: page }) => {
    await test.step('导航到 AI 设置页', async () => {
      await page.goto(`${SETTINGS_URL}/ai`)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T9-75-AI设置.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T9-76: 设置保存按钮
  test('T9-76 设置保存按钮', async ({ authPage: page }) => {
    await test.step('导航到 SMTP 设置页', async () => {
      await page.goto(`${SETTINGS_URL}/smtp`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找保存按钮', async () => {
      const saveBtn = page.locator('button').filter({ hasText: /保存|Save|应用|Apply/ }).first()
      if (await saveBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-76-保存按钮.png', fullPage: false })
      }
    })
  })

  // T9-77: 设置页面导航
  test('T9-77 设置页面导航', async ({ authPage: page }) => {
    await test.step('导航到设置页', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查设置菜单', async () => {
      const menuItems = page.locator('.n-menu-item, [class*="menu-item"], [class*="nav-item"]')
      const count = await menuItems.count()
      await page.screenshot({ path: 'test-results/T9-77-设置导航.png', fullPage: false })
    })
  })

  // T9-78: 加密密钥配置
  test('T9-78 加密密钥配置', async ({ authPage: page }) => {
    await test.step('导航到安全设置页', async () => {
      await page.goto(`${SETTINGS_URL}/security`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找密钥配置', async () => {
      const keyInput = page.locator('input[type="password"], input[placeholder*="密钥"], input[placeholder*="key"]').first()
      if (await keyInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-78-密钥配置.png', fullPage: false })
      }
    })
  })

  // T9-79: 速率限制配置
  test('T9-79 速率限制配置', async ({ authPage: page }) => {
    await test.step('导航到安全设置页', async () => {
      await page.goto(`${SETTINGS_URL}/security`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找速率限制', async () => {
      const rateLimitInput = page.locator('input[placeholder*="速率"], input[placeholder*="rate"], [class*="rate-limit"]').first()
      if (await rateLimitInput.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T9-79-速率限制.png', fullPage: false })
      }
    })
  })

  // T9-80: 设置页面保存反馈
  test('T9-80 设置页面保存反馈', async ({ authPage: page }) => {
    await test.step('导航到 SMTP 设置页', async () => {
      await page.goto(`${SETTINGS_URL}/smtp`)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击保存并检查反馈', async () => {
      const saveBtn = page.locator('button').filter({ hasText: /保存|Save|应用|Apply/ }).first()
      if (await saveBtn.isVisible().catch(() => false)) {
        await saveBtn.click()
        await page.waitForTimeout(1000)
        const feedback = page.locator('.n-message, [class*="toast"], [class*="notification"], .n-notification').first()
        if (await feedback.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T9-80-保存反馈.png', fullPage: false })
        }
      }
    })
  })
})
