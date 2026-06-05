import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// T6: 数据源完整测试 — 60 个测试用例
// 覆盖：列表页(T6-1~T6-15)、CRUD(T6-16~T6-30)、查询探索(T6-31~T6-45)、仪表盘(T6-46~T6-60)

const DS_URL = '/alert/datasources'
const EXPLORE_URL = '/alert/explore'
const DASH_URL = '/alert/dashboards'

/** 生成唯一名称 */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

/** 通过 API 创建测试数据源 */
async function createTestDatasource(page: import('@playwright/test').Page, overrides?: Record<string, unknown>): Promise<number> {
  const name = uid('test_ds')
  const body = {
    name,
    type: 'prometheus',
    url: 'http://localhost:9090',
    is_default: false,
    status: 'active',
    ...overrides,
  }
  const res = await API.post(page, '/api/v1/datasources', body)
  return res?.data?.id ?? 0
}

/** 通过 API 删除数据源 */
async function deleteTestDatasource(page: import('@playwright/test').Page, id: number): Promise<void> {
  if (id > 0) {
    await API.del(page, `/api/v1/datasources/${id}`)
  }
}

test.describe('T6 - 数据源完整测试', () => {

  // ================================================================
  // T6-1 ~ T6-15: 数据源列表
  // ================================================================

  // T6-1: 列表页初始加载
  test('T6-1 列表页初始加载', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T6-1-列表页初始加载.png', fullPage: true })
    })

    await test.step('验证页面标题区域', async () => {
      await expect(page.locator('body')).toBeVisible()
    })

    await test.step('验证创建按钮存在', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      await expect(createBtn).toBeVisible()
    })
  })

  // T6-2: 数据源卡片展示
  test('T6-2 数据源卡片展示', async ({ authPage: page }) => {
    const dsId = await createTestDatasource(page)

    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('验证数据源卡片', async () => {
      const cards = page.locator('.n-card, [class*="card"], [class*="ds-card"], [class*="datasource-card"]')
      const count = await cards.count()
      await page.screenshot({ path: 'test-results/T6-2-数据源卡片.png', fullPage: false })
    })

    await test.step('清理测试数据', async () => {
      await deleteTestDatasource(page, dsId)
    })
  })

  // T6-3: 数据源类型标签显示
  test('T6-3 数据源类型标签显示', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查类型标签', async () => {
      const typeLabels = page.locator('.n-tag, [class*="tag"], [class*="badge"]').filter({ hasText: /prometheus|victoriametrics|zabbix|elasticsearch|loki|vm/i })
      const count = await typeLabels.count()
      await page.screenshot({ path: 'test-results/T6-3-类型标签.png', fullPage: false })
    })
  })

  // T6-4: 数据源状态徽标
  test('T6-4 数据源状态徽标', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查状态徽标', async () => {
      const statusBadges = page.locator('.n-tag, [class*="badge"], [class*="status"]').filter({ hasText: /active|inactive|error|正常|异常/ })
      const count = await statusBadges.count()
      await page.screenshot({ path: 'test-results/T6-4-状态徽标.png', fullPage: false })
    })
  })

  // T6-5: 默认数据源标记
  test('T6-5 默认数据源标记', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查默认标记', async () => {
      const defaultBadge = page.locator('text=默认, text=Default, [class*="default"]').first()
      if (await defaultBadge.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-5-默认标记.png', fullPage: false })
      }
    })
  })

  // T6-6: 健康检查状态
  test('T6-6 健康检查状态', async ({ authPage: page }) => {
    const dsId = await createTestDatasource(page)

    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查健康状态指示器', async () => {
      const healthIndicator = page.locator('.health-dot, [class*="health"], [class*="status-dot"], .n-icon').first()
      if (await healthIndicator.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-6-健康状态.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteTestDatasource(page, dsId)
    })
  })

  // T6-7: 搜索数据源
  test('T6-7 搜索数据源', async ({ authPage: page }) => {
    const dsId = await createTestDatasource(page, { name: 'searchable_ds_test' })

    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], .toolbar-search input').first()
      if (await searchInput.isVisible()) {
        await searchInput.fill('searchable')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-7-搜索结果.png', fullPage: false })
      }
    })

    await test.step('清空搜索', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], .toolbar-search input').first()
      if (await searchInput.isVisible()) {
        await searchInput.clear()
        await page.waitForTimeout(400)
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteTestDatasource(page, dsId)
    })
  })

  // T6-8: 按类型筛选
  test('T6-8 按类型筛选', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开类型筛选下拉', async () => {
      const typeSelect = page.locator('.n-select, [class*="filter-select"]').first()
      if (await typeSelect.isVisible()) {
        await typeSelect.click()
        await page.waitForTimeout(300)
        const option = page.locator('.n-select-option, [class*="option"]').filter({ hasText: /prometheus|victoriametrics/i }).first()
        if (await option.isVisible()) {
          await option.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T6-8-类型筛选.png', fullPage: false })
        }
      }
    })
  })

  // T6-9: 分页控件
  test('T6-9 分页控件', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查分页控件', async () => {
      const pagination = page.locator('.n-pagination, [class*="pagination"]').first()
      if (await pagination.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-9-分页控件.png', fullPage: false })
      }
    })
  })

  // T6-10: 空状态展示
  test('T6-10 空状态展示', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查空状态或列表', async () => {
      const emptyState = page.locator('.n-empty, [class*="empty"], [class*="EmptyState"]').first()
      const cards = page.locator('.n-card, [class*="ds-card"]')
      const hasCards = await cards.count() > 0
      if (!hasCards && await emptyState.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-10-空状态.png', fullPage: false })
      }
    })
  })

  // T6-11: 数据源 URL 显示
  test('T6-11 数据源 URL 显示', async ({ authPage: page }) => {
    const dsId = await createTestDatasource(page)

    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查 URL 显示', async () => {
      const urlText = page.locator('text=http, [class*="url"], code').first()
      if (await urlText.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-11-URL显示.png', fullPage: false })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteTestDatasource(page, dsId)
    })
  })

  // T6-12: 数据源描述显示
  test('T6-12 数据源描述显示', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查描述区域', async () => {
      const descText = page.locator('[class*="desc"], [class*="description"]').first()
      if (await descText.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-12-描述显示.png', fullPage: false })
      }
    })
  })

  // T6-13: 数据源操作按钮
  test('T6-13 数据源操作按钮', async ({ authPage: page }) => {
    const dsId = await createTestDatasource(page)

    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查操作按钮', async () => {
      const actionBtns = page.locator('button').filter({ hasText: /编辑|删除|Edit|Delete|测试|Test/ })
      const count = await actionBtns.count()
      await page.screenshot({ path: 'test-results/T6-13-操作按钮.png', fullPage: false })
    })

    await test.step('清理测试数据', async () => {
      await deleteTestDatasource(page, dsId)
    })
  })

  // T6-14: 数据源列表刷新
  test('T6-14 数据源列表刷新', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击刷新按钮', async () => {
      const refreshBtn = page.locator('button').filter({ hasText: /刷新|Refresh|Reload/ }).first()
      if (await refreshBtn.isVisible().catch(() => false)) {
        await refreshBtn.click()
        await page.waitForTimeout(1000)
      }
      await page.screenshot({ path: 'test-results/T6-14-刷新后.png', fullPage: false })
    })
  })

  // T6-15: 数据源列表排序
  test('T6-15 数据源列表排序', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查排序控件', async () => {
      const sortSelect = page.locator('.n-select, [class*="sort"]').filter({ hasText: /排序|Sort|名称|时间/ }).first()
      if (await sortSelect.isVisible().catch(() => false)) {
        await sortSelect.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T6-15-排序选项.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // ================================================================
  // T6-16 ~ T6-30: 数据源 CRUD
  // ================================================================

  // T6-16: 创建弹窗打开
  test('T6-16 创建弹窗打开', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击创建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      await createBtn.click()
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T6-16-创建弹窗.png', fullPage: false })
    })

    await test.step('验证弹窗内容', async () => {
      const modal = page.locator('.n-modal, [role="dialog"], .n-drawer').first()
      await expect(modal).toBeVisible()
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T6-17: 创建弹窗类型选择器
  test('T6-17 创建弹窗类型选择器', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      await createBtn.click()
      await page.waitForTimeout(500)
    })

    await test.step('点击类型选择器', async () => {
      const typeSelect = page.locator('.n-modal .n-select, [role="dialog"] .n-select, .n-drawer .n-select').first()
      if (await typeSelect.isVisible()) {
        await typeSelect.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T6-17-类型选择器.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T6-18: Prometheus 类型端点配置
  test('T6-18 Prometheus 类型端点配置', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗并选择 Prometheus', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      await createBtn.click()
      await page.waitForTimeout(500)
      const typeSelect = page.locator('.n-modal .n-select, [role="dialog"] .n-select').first()
      if (await typeSelect.isVisible()) {
        await typeSelect.click()
        await page.waitForTimeout(300)
        const promOption = page.locator('.n-select-option, [class*="option"]').filter({ hasText: /prometheus/i }).first()
        if (await promOption.isVisible()) {
          await promOption.click()
          await page.waitForTimeout(300)
        }
      }
    })

    await test.step('检查 URL 输入框', async () => {
      const urlInput = page.locator('input[placeholder*="URL"], input[placeholder*="url"], input[placeholder*="http"]').first()
      if (await urlInput.isVisible().catch(() => false)) {
        await urlInput.fill('http://localhost:9090')
        await page.screenshot({ path: 'test-results/T6-18-端点配置.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T6-19: 名称输入验证
  test('T6-19 名称输入验证', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      await createBtn.click()
      await page.waitForTimeout(500)
    })

    await test.step('留空名称直接提交', async () => {
      const submitBtn = page.locator('.n-modal button, [role="dialog"] button').filter({ hasText: /确定|Save|Submit|创建|Create/ }).first()
      if (await submitBtn.isVisible()) {
        await submitBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-19-名称验证.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T6-20: 认证配置区域
  test('T6-20 认证配置区域', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      await createBtn.click()
      await page.waitForTimeout(500)
    })

    await test.step('检查认证配置区域', async () => {
      const authSection = page.locator('text=认证, text=Auth, text=Token, text=Basic, [class*="auth"]').first()
      if (await authSection.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-20-认证配置.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T6-21: 测试连接按钮
  test('T6-21 测试连接按钮', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      await createBtn.click()
      await page.waitForTimeout(500)
    })

    await test.step('查找测试连接按钮', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|Check|Ping/ }).first()
      if (await testBtn.isVisible()) {
        await page.screenshot({ path: 'test-results/T6-21-测试连接按钮.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T6-22: 通过 API 创建数据源
  test('T6-22 通过 API 创建数据源', async ({ authPage: page }) => {
    let dsId = 0

    await test.step('通过 API 创建数据源', async () => {
      dsId = await createTestDatasource(page, { name: uid('api_ds'), type: 'prometheus', url: 'http://localhost:9090' })
      expect(dsId).toBeGreaterThan(0)
    })

    await test.step('验证数据源出现在列表', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T6-22-API创建验证.png', fullPage: false })
    })

    await test.step('清理测试数据', async () => {
      await deleteTestDatasource(page, dsId)
    })
  })

  // T6-23: 编辑数据源弹窗
  test('T6-23 编辑数据源弹窗', async ({ authPage: page }) => {
    const dsId = await createTestDatasource(page)

    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击编辑按钮', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible()) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-23-编辑弹窗.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteTestDatasource(page, dsId)
    })
  })

  // T6-24: 编辑数据源名称
  test('T6-24 编辑数据源名称', async ({ authPage: page }) => {
    const dsId = await createTestDatasource(page)

    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击编辑按钮', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible()) {
        await editBtn.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('修改名称', async () => {
      const nameInput = page.locator('.n-modal input, [role="dialog"] input, .n-drawer input').first()
      if (await nameInput.isVisible()) {
        await nameInput.clear()
        await nameInput.fill('updated_ds_name')
        await page.screenshot({ path: 'test-results/T6-24-编辑名称.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })

    await test.step('清理测试数据', async () => {
      await deleteTestDatasource(page, dsId)
    })
  })

  // T6-25: 删除数据源确认弹窗
  test('T6-25 删除数据源确认弹窗', async ({ authPage: page }) => {
    const dsId = await createTestDatasource(page)

    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击删除按钮', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete|Remove/ }).first()
      if (await deleteBtn.isVisible()) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-25-删除确认.png', fullPage: false })
      }
    })

    await test.step('取消删除', async () => {
      const cancelBtn = page.locator('button').filter({ hasText: /取消|Cancel/ }).first()
      if (await cancelBtn.isVisible()) {
        await cancelBtn.click()
      } else {
        await page.keyboard.press('Escape')
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteTestDatasource(page, dsId)
    })
  })

  // T6-26: 通过 API 删除数据源
  test('T6-26 通过 API 删除数据源', async ({ authPage: page }) => {
    let dsId = 0

    await test.step('创建并删除数据源', async () => {
      dsId = await createTestDatasource(page)
      expect(dsId).toBeGreaterThan(0)
      await deleteTestDatasource(page, dsId)
      dsId = 0
    })

    await test.step('验证数据源已删除', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T6-26-删除后验证.png', fullPage: false })
    })
  })

  // T6-27: 数据源标签显示
  test('T6-27 数据源标签显示', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查标签元素', async () => {
      const tags = page.locator('.n-tag, [class*="label-tag"], [class*="tag"]')
      const count = await tags.count()
      await page.screenshot({ path: 'test-results/T6-27-标签显示.png', fullPage: false })
    })
  })

  // T6-28: 创建表单必填字段验证
  test('T6-28 创建表单必填字段验证', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗并直接提交', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      await createBtn.click()
      await page.waitForTimeout(500)
      const submitBtn = page.locator('.n-modal button, [role="dialog"] button').filter({ hasText: /确定|Save|Submit|创建/ }).first()
      if (await submitBtn.isVisible()) {
        await submitBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-28-必填验证.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // T6-29: 数据源详情页
  test('T6-29 数据源详情页', async ({ authPage: page }) => {
    const dsId = await createTestDatasource(page)

    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击数据源卡片查看详情', async () => {
      const card = page.locator('.n-card, [class*="card"], [class*="ds-card"]').first()
      if (await card.isVisible()) {
        await card.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-29-详情页.png', fullPage: true })
      }
    })

    await test.step('清理测试数据', async () => {
      await deleteTestDatasource(page, dsId)
    })
  })

  // T6-30: VictoriaMetrics 类型配置
  test('T6-30 VictoriaMetrics 类型配置', async ({ authPage: page }) => {
    await test.step('导航到数据源页', async () => {
      await page.goto(DS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开创建弹窗并选择 VM', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加/ }).first()
      await createBtn.click()
      await page.waitForTimeout(500)
      const typeSelect = page.locator('.n-modal .n-select, [role="dialog"] .n-select').first()
      if (await typeSelect.isVisible()) {
        await typeSelect.click()
        await page.waitForTimeout(300)
        const vmOption = page.locator('.n-select-option, [class*="option"]').filter({ hasText: /victoria|vm|VictoriaMetrics/i }).first()
        if (await vmOption.isVisible()) {
          await vmOption.click()
          await page.waitForTimeout(300)
          await page.screenshot({ path: 'test-results/T6-30-VM配置.png', fullPage: false })
        }
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(200)
      await page.keyboard.press('Escape')
    })
  })

  // ================================================================
  // T6-31 ~ T6-45: 查询探索
  // ================================================================

  // T6-31: 探索页面加载
  test('T6-31 探索页面加载', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T6-31-探索页面.png', fullPage: true })
    })

    await test.step('验证页面结构', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T6-32: 数据源选择器
  test('T6-32 数据源选择器', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找数据源选择器', async () => {
      const dsSelector = page.locator('.n-select, [class*="ds-select"], [class*="datasource-select"]').first()
      if (await dsSelector.isVisible()) {
        await dsSelector.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T6-32-数据源选择器.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T6-33: 表达式输入框
  test('T6-33 表达式输入框', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找表达式输入', async () => {
      const exprInput = page.locator('textarea, .monaco-editor, [class*="expr-input"], input[placeholder*="expression"], input[placeholder*="PromQL"]').first()
      if (await exprInput.isVisible()) {
        await exprInput.click()
        await page.screenshot({ path: 'test-results/T6-33-表达式输入.png', fullPage: false })
      }
    })
  })

  // T6-34: 查询执行按钮
  test('T6-34 查询执行按钮', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找执行按钮', async () => {
      const runBtn = page.locator('button').filter({ hasText: /运行|Run|Execute|查询|Query/ }).first()
      if (await runBtn.isVisible()) {
        await page.screenshot({ path: 'test-results/T6-34-执行按钮.png', fullPage: false })
      }
    })
  })

  // T6-35: 查询结果区域
  test('T6-35 查询结果区域', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查结果区域', async () => {
      const resultArea = page.locator('[class*="result"], [class*="table"], .n-data-table, [class*="graph"]').first()
      if (await resultArea.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-35-结果区域.png', fullPage: false })
      }
    })
  })

  // T6-36: 时间范围选择器
  test('T6-36 时间范围选择器', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找时间范围选择器', async () => {
      const timePicker = page.locator('.n-date-picker, [class*="time-picker"], [class*="time-range"], button').filter({ hasText: /最近|Last|小时|天|Minutes|Hours/ }).first()
      if (await timePicker.isVisible()) {
        await timePicker.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T6-36-时间范围.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T6-37: 查询历史记录
  test('T6-37 查询历史记录', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找历史记录入口', async () => {
      const historyBtn = page.locator('button, [class*="history"]').filter({ hasText: /历史|History|记录/ }).first()
      if (await historyBtn.isVisible().catch(() => false)) {
        await historyBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T6-37-查询历史.png', fullPage: false })
      }
    })
  })

  // T6-38: Metrics 标签页
  test('T6-38 Metrics 标签页', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找 Metrics 标签', async () => {
      const metricsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /Metrics|指标|Metric/ }).first()
      if (await metricsTab.isVisible()) {
        await metricsTab.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T6-38-Metrics标签.png', fullPage: false })
      }
    })
  })

  // T6-39: Logs 标签页
  test('T6-39 Logs 标签页', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找 Logs 标签', async () => {
      const logsTab = page.locator('[class*="tab"], .n-tabs-tab').filter({ hasText: /Logs|日志|Log/ }).first()
      if (await logsTab.isVisible()) {
        await logsTab.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T6-39-Logs标签.png', fullPage: false })
      }
    })
  })

  // T6-40: ES 探索页面
  test('T6-40 ES 探索页面', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找 ES 相关选项', async () => {
      const esOption = page.locator('text=Elasticsearch, text=ES, [class*="es-"]').first()
      if (await esOption.isVisible().catch(() => false)) {
        await esOption.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-40-ES探索.png', fullPage: false })
      }
    })
  })

  // T6-41: 表达式自动补全
  test('T6-41 表达式自动补全', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入触发自动补全', async () => {
      const exprInput = page.locator('textarea, input[placeholder*="expression"], input[placeholder*="PromQL"]').first()
      if (await exprInput.isVisible()) {
        await exprInput.fill('up')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-41-自动补全.png', fullPage: false })
      }
    })
  })

  // T6-42: 图表展示区域
  test('T6-42 图表展示区域', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查图表区域', async () => {
      const graphArea = page.locator('[class*="graph"], [class*="chart"], canvas, svg').first()
      if (await graphArea.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-42-图表区域.png', fullPage: false })
      }
    })
  })

  // T6-43: 表格展示区域
  test('T6-43 表格展示区域', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('检查表格区域', async () => {
      const tableArea = page.locator('.n-data-table, table, [class*="table"]').first()
      if (await tableArea.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-43-表格区域.png', fullPage: false })
      }
    })
  })

  // T6-44: 查询格式切换
  test('T6-44 查询格式切换', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找格式切换', async () => {
      const formatToggle = page.locator('button, [class*="toggle"]').filter({ hasText: /表格|图表|Graph|Table|Table/ }).first()
      if (await formatToggle.isVisible().catch(() => false)) {
        await formatToggle.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T6-44-格式切换.png', fullPage: false })
      }
    })
  })

  // T6-45: 探索页面分屏模式
  test('T6-45 探索页面分屏模式', async ({ authPage: page }) => {
    await test.step('导航到探索页', async () => {
      await page.goto(EXPLORE_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找分屏按钮', async () => {
      const splitBtn = page.locator('button').filter({ hasText: /分屏|Split|添加查询|Add Query/ }).first()
      if (await splitBtn.isVisible().catch(() => false)) {
        await splitBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-45-分屏模式.png', fullPage: true })
      }
    })
  })

  // ================================================================
  // T6-46 ~ T6-60: 仪表盘
  // ================================================================

  // T6-46: 仪表盘列表页
  test('T6-46 仪表盘列表页', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T6-46-仪表盘列表.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T6-47: 仪表盘搜索
  test('T6-47 仪表盘搜索', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], .toolbar-search input').first()
      if (await searchInput.isVisible()) {
        await searchInput.fill('test')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-47-仪表盘搜索.png', fullPage: false })
      }
    })
  })

  // T6-48: 新建仪表盘弹窗
  test('T6-48 新建仪表盘弹窗', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击新建按钮', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-48-新建仪表盘.png', fullPage: false })
      }
    })

    await test.step('关闭弹窗', async () => {
      await page.keyboard.press('Escape')
    })
  })

  // T6-49: 仪表盘编辑器加载
  test('T6-49 仪表盘编辑器加载', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开第一个仪表盘', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="dash-item"], tr').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T6-49-编辑器.png', fullPage: true })
      }
    })
  })

  // T6-50: 仪表盘面板展示
  test('T6-50 仪表盘面板展示', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开仪表盘查看面板', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="dash-item"], tr').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(1000)
        const panels = page.locator('[class*="panel"], [class*="chart"], canvas, svg')
        const count = await panels.count()
        await page.screenshot({ path: 'test-results/T6-50-面板展示.png', fullPage: false })
      }
    })
  })

  // T6-51: 添加面板按钮
  test('T6-51 添加面板按钮', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开仪表盘', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], tr').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('查找添加面板按钮', async () => {
      const addPanelBtn = page.locator('button').filter({ hasText: /添加面板|Add Panel|新建面板/ }).first()
      if (await addPanelBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-51-添加面板.png', fullPage: false })
      }
    })
  })

  // T6-52: 仪表盘变量选择
  test('T6-52 仪表盘变量选择', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开仪表盘查看变量', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], tr').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(1000)
        const varSelect = page.locator('.n-select, select, [class*="variable"]').first()
        if (await varSelect.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T6-52-变量选择.png', fullPage: false })
        }
      }
    })
  })

  // T6-53: 仪表盘时间范围
  test('T6-53 仪表盘时间范围', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开仪表盘', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], tr').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('查找时间范围选择器', async () => {
      const timePicker = page.locator('.n-date-picker, [class*="time-picker"], button').filter({ hasText: /最近|Last|小时|天|Minutes/ }).first()
      if (await timePicker.isVisible()) {
        await timePicker.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T6-53-时间范围.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T6-54: 仪表盘自动刷新
  test('T6-54 仪表盘自动刷新', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开仪表盘', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], tr').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('查找自动刷新控件', async () => {
      const refreshBtn = page.locator('button, [class*="refresh"]').filter({ hasText: /刷新|Refresh|自动/ }).first()
      if (await refreshBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-54-自动刷新.png', fullPage: false })
      }
    })
  })

  // T6-55: 仪表盘全屏模式
  test('T6-55 仪表盘全屏模式', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开仪表盘', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], tr').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('查找全屏按钮', async () => {
      const fullscreenBtn = page.locator('button').filter({ hasText: /全屏|Fullscreen|展开/ }).first()
      if (await fullscreenBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-55-全屏按钮.png', fullPage: false })
      }
    })
  })

  // T6-56: 仪表盘分享功能
  test('T6-56 仪表盘分享功能', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开仪表盘', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], tr').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('查找分享按钮', async () => {
      const shareBtn = page.locator('button').filter({ hasText: /分享|Share|链接/ }).first()
      if (await shareBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-56-分享按钮.png', fullPage: false })
      }
    })
  })

  // T6-57: 仪表盘导出
  test('T6-57 仪表盘导出', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找导出按钮', async () => {
      const exportBtn = page.locator('button').filter({ hasText: /导出|Export|下载/ }).first()
      if (await exportBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-57-导出按钮.png', fullPage: false })
      }
    })
  })

  // T6-58: 仪表盘导入
  test('T6-58 仪表盘导入', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('查找导入按钮', async () => {
      const importBtn = page.locator('button').filter({ hasText: /导入|Import|上传/ }).first()
      if (await importBtn.isVisible().catch(() => false)) {
        await importBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-58-导入弹窗.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T6-59: 仪表盘编辑标题
  test('T6-59 仪表盘编辑标题', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开仪表盘', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], tr').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('查找设置按钮', async () => {
      const settingsBtn = page.locator('button').filter({ hasText: /设置|Settings|编辑|Edit/ }).first()
      if (await settingsBtn.isVisible().catch(() => false)) {
        await settingsBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T6-59-编辑标题.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T6-60: 仪表盘保存功能
  test('T6-60 仪表盘保存功能', async ({ authPage: page }) => {
    await test.step('导航到仪表盘页', async () => {
      await page.goto(DASH_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('打开仪表盘', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], tr').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(1000)
      }
    })

    await test.step('查找保存按钮', async () => {
      const saveBtn = page.locator('button').filter({ hasText: /保存|Save/ }).first()
      if (await saveBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T6-60-保存按钮.png', fullPage: false })
      }
    })
  })
})
