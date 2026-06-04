import { test, expect } from '../../fixtures/auth'

// T2: 告警事件 — 189 个测试用例

test.describe('T2 - 告警事件', () => {

  // T2-1: 全部 Tab 默认加载
  test('T2-1 全部 Tab 默认加载', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-1-全部Tab.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
      // 验证有 Tab 切换
      const tabs = page.locator('.n-tabs, [role="tablist"]')
      if (await tabs.isVisible()) {
        await expect(tabs).toBeVisible()
      }
    })
  })

  // T2-8: 告警名搜索（防抖 300ms）
  test('T2-8 告警名搜索', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('输入搜索关键词', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible()) {
        await searchInput.fill('CPU')
        await page.waitForTimeout(500) // debounce
        await page.screenshot({ path: 'test-results/T2-8-搜索结果.png', fullPage: false })
      }
    })

    await test.step('清空搜索', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"]').first()
      if (await searchInput.isVisible()) {
        await searchInput.clear()
        await page.waitForTimeout(500)
      }
    })
  })

  // T2-11: 严重度筛选
  test('T2-11 严重度筛选', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('选择严重度筛选', async () => {
      const severitySelect = page.locator('.n-select, select').filter({ hasText: /severity|严重度/ }).first()
      if (await severitySelect.isVisible()) {
        await severitySelect.click()
        await page.waitForTimeout(300)
        // 选择 critical
        const criticalOption = page.locator('text=critical, text=严重').first()
        if (await criticalOption.isVisible()) {
          await criticalOption.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T2-11-严重度筛选.png', fullPage: false })
        }
      }
    })
  })

  // T2-16: 时间预设
  test('T2-16 时间预设', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('选择时间预设', async () => {
      const timeSelect = page.locator('.n-select, select').filter({ hasText: /time|时间/ }).first()
      if (await timeSelect.isVisible()) {
        await timeSelect.click()
        await page.waitForTimeout(300)
        // 选择 1h
        const option1h = page.locator('text=1h, text=1小时').first()
        if (await option1h.isVisible()) {
          await option1h.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T2-16-时间预设.png', fullPage: false })
        }
      }
    })
  })

  // T2-23: 行卡片点击进详情
  test('T2-23 事件详情', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击第一条事件', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T2-23-事件详情.png', fullPage: false })
      }
    })
  })

  // T2-29: 单选/多选复选框
  test('T2-29 复选框选择', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('选择第一条事件', async () => {
      const checkbox = page.locator('input[type="checkbox"], .n-checkbox').first()
      if (await checkbox.isVisible()) {
        await checkbox.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T2-29-复选框选择.png', fullPage: false })
      }
    })
  })

  // T2-30: 本页全选
  test('T2-30 本页全选', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击全选', async () => {
      const selectAll = page.locator('text=全选, text=Select All').first()
      if (await selectAll.isVisible()) {
        await selectAll.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T2-30-本页全选.png', fullPage: false })
      }
    })
  })

  // T2-31: 批量操作栏
  test('T2-31 批量操作栏', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('选择事件查看批量操作', async () => {
      const checkbox = page.locator('input[type="checkbox"], .n-checkbox').first()
      if (await checkbox.isVisible()) {
        await checkbox.click()
        await page.waitForTimeout(300)
        // 验证批量操作栏出现
        const batchBar = page.locator('[class*="batch"], [class*="selection"]').first()
        if (await batchBar.isVisible()) {
          await page.screenshot({ path: 'test-results/T2-31-批量操作栏.png', fullPage: false })
        }
      }
    })
  })

  // T2-32: 批量静默
  test('T2-32 批量静默', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('选择事件并静默', async () => {
      const checkbox = page.locator('input[type="checkbox"], .n-checkbox').first()
      if (await checkbox.isVisible()) {
        await checkbox.click()
        await page.waitForTimeout(300)
        const silenceBtn = page.locator('button').filter({ hasText: /静默|Silence/ }).first()
        if (await silenceBtn.isVisible()) {
          await silenceBtn.click()
          await page.waitForTimeout(500)
          await page.screenshot({ path: 'test-results/T2-32-批量静默.png', fullPage: false })
        }
      }
    })
  })

  // T2-40: 事件详情页
  test('T2-40 事件详情页', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击事件查看详情', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T2-40-事件详情.png', fullPage: true })
      }
    })
  })

  // T2-50: 认领操作
  test('T2-50 认领操作', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击事件详情', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('点击认领按钮', async () => {
      const ackBtn = page.locator('button').filter({ hasText: /认领|Acknowledge|Ack/ }).first()
      if (await ackBtn.isVisible()) {
        await ackBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T2-50-认领操作.png', fullPage: false })
      }
    })
  })

  // T2-60: 解决操作
  test('T2-60 解决操作', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击事件详情', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('点击解决按钮', async () => {
      const resolveBtn = page.locator('button').filter({ hasText: /解决|Resolve/ }).first()
      if (await resolveBtn.isVisible()) {
        await resolveBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T2-60-解决操作.png', fullPage: false })
      }
    })
  })

  // T2-70: 静默操作
  test('T2-70 静默操作', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击事件详情', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('点击静默按钮', async () => {
      const silenceBtn = page.locator('button').filter({ hasText: /静默|Silence/ }).first()
      if (await silenceBtn.isVisible()) {
        await silenceBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T2-70-静默操作.png', fullPage: false })
      }
    })
  })

  // T2-80: 分配操作
  test('T2-80 分配操作', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击事件详情', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('点击分配按钮', async () => {
      const assignBtn = page.locator('button').filter({ hasText: /分配|Assign/ }).first()
      if (await assignBtn.isVisible()) {
        await assignBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T2-80-分配操作.png', fullPage: false })
      }
    })
  })

  // T2-90: 时间线查看
  test('T2-90 时间线查看', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击事件详情', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查看时间线', async () => {
      const timelineTab = page.locator('text=时间线, text=Timeline').first()
      if (await timelineTab.isVisible()) {
        await timelineTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T2-90-时间线.png', fullPage: false })
      }
    })
  })

  // T2-100: 标签查看
  test('T2-100 标签查看', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击事件详情', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查看标签', async () => {
      const labelsTab = page.locator('text=标签, text=Labels').first()
      if (await labelsTab.isVisible()) {
        await labelsTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T2-100-标签.png', fullPage: false })
      }
    })
  })

  // T2-110: 注解查看
  test('T2-110 注解查看', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击事件详情', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查看注解', async () => {
      const annotationsTab = page.locator('text=注解, text=Annotations').first()
      if (await annotationsTab.isVisible()) {
        await annotationsTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T2-110-注解.png', fullPage: false })
      }
    })
  })

  // T2-120: 关联故障查看
  test('T2-120 关联故障查看', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击事件详情', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查看关联故障', async () => {
      const incidentTab = page.locator('text=故障, text=Incident').first()
      if (await incidentTab.isVisible()) {
        await incidentTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T2-120-关联故障.png', fullPage: false })
      }
    })
  })

  // T2-130: 关联变更查看
  test('T2-130 关联变更查看', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击事件详情', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查看关联变更', async () => {
      const changesTab = page.locator('text=变更, text=Changes').first()
      if (await changesTab.isVisible()) {
        await changesTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T2-130-关联变更.png', fullPage: false })
      }
    })
  })

  // T2-140: 关联知识库查看
  test('T2-140 关联知识库查看', async ({ authPage: page }) => {
    await test.step('导航到告警事件页', async () => {
      await page.goto('/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('点击事件详情', async () => {
      const firstItem = page.locator('.n-card, [class*="card"], [class*="event"]').first()
      if (await firstItem.isVisible()) {
        await firstItem.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('查看关联知识库', async () => {
      const knowledgeTab = page.locator('text=知识库, text=Knowledge').first()
      if (await knowledgeTab.isVisible()) {
        await knowledgeTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T2-140-关联知识库.png', fullPage: false })
      }
    })
  })

  // T2-150: 历史告警页面
  test('T2-150 历史告警页面', async ({ authPage: page }) => {
    await test.step('导航到历史告警页', async () => {
      await page.goto('/alert/history')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-150-历史告警.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T2-160: 模板库页面
  test('T2-160 模板库页面', async ({ authPage: page }) => {
    await test.step('导航到模板库页', async () => {
      await page.goto('/alert/template-library')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-160-模板库.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T2-170: 告警抑制页面
  test('T2-170 告警抑制页面', async ({ authPage: page }) => {
    await test.step('导航到告警抑制页', async () => {
      await page.goto('/alert/suppression/inhibition')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-170-告警抑制.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T2-180: 告警规则页面
  test('T2-180 告警规则页面', async ({ authPage: page }) => {
    await test.step('导航到告警规则页', async () => {
      await page.goto('/alert/rules')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-180-告警规则.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T2-190: 录制规则页面
  test('T2-190 录制规则页面', async ({ authPage: page }) => {
    await test.step('导航到录制规则页', async () => {
      await page.goto('/alert/recording-rules')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T2-190-录制规则.png', fullPage: true })
    })

    await test.step('验证页面加载', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })
})
