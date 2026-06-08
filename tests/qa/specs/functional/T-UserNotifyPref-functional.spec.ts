import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

// ---------------------------------------------------------------------------
// UNP-1 用户通知偏好 CRUD
// ---------------------------------------------------------------------------
test('UNP-1 用户通知偏好CRUD', async ({ authPage: page }) => {
  let configId: number | null = null

  try {
    // ---- 1. 获取当前通知偏好列表 ----
    await test.step('获取当前通知偏好', async () => {
      const res = await API.get(page, `${API_BASE}/me/notify-configs`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      expect(Array.isArray(res.data.list || res.data)).toBe(true)
      await page.screenshot({ path: 'test-results/UNP-1-01-获取偏好.png', fullPage: false })
    })

    // ---- 2. 创建通知偏好 ----
    await test.step('创建通知偏好', async () => {
      const res = await API.post(page, `${API_BASE}/me/notify-configs`, {
        media_type: 'webhook',
        target: `https://example.com/hook-${uid()}`,
        enabled: true,
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      expect(res.data.id).toBeGreaterThan(0)
      configId = res.data.id
      await page.screenshot({ path: 'test-results/UNP-1-02-创建偏好.png', fullPage: false })
    })

    // ---- 3. 验证创建成功 ----
    await test.step('验证创建成功', async () => {
      const res = await API.get(page, `${API_BASE}/me/notify-configs`)
      expect(res.code).toBe(0)
      const list = res.data.list || res.data || []
      const found = list.find((c: any) => c.id === configId)
      expect(found).toBeTruthy()
      expect(found.media_type).toBe('webhook')
      expect(found.enabled).toBe(true)
      await page.screenshot({ path: 'test-results/UNP-1-03-验证创建.png', fullPage: false })
    })

    // ---- 4. 更新通知偏好 ----
    await test.step('更新通知偏好', async () => {
      const res = await API.put(page, `${API_BASE}/me/notify-configs/${configId}`, {
        enabled: false,
        target: `https://example.com/updated-hook-${uid()}`,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/UNP-1-04-更新偏好.png', fullPage: false })
    })

    // ---- 5. 验证更新 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/me/notify-configs`)
      expect(res.code).toBe(0)
      const list = res.data.list || res.data || []
      const found = list.find((c: any) => c.id === configId)
      expect(found).toBeTruthy()
      expect(found.enabled).toBe(false)
      await page.screenshot({ path: 'test-results/UNP-1-05-验证更新.png', fullPage: false })
    })
  } finally {
    // cleanup: delete the config
    if (configId) {
      try {
        await API.del(page, `${API_BASE}/me/notify-configs/${configId}`)
      } catch { /* ignore */ }
    }
  }
})

// ---------------------------------------------------------------------------
// UNP-2 用户通知偏好按媒体类型
// ---------------------------------------------------------------------------
test('UNP-2 用户通知偏好按媒体类型', async ({ authPage: page }) => {
  const configIds: number[] = []

  try {
    // ---- 1. 创建多个不同类型的偏好 ----
    await test.step('创建多种类型偏好', async () => {
      const types = ['email', 'webhook']
      for (const type of types) {
        const res = await API.post(page, `${API_BASE}/me/notify-configs`, {
          media_type: type,
          target: `${type}-${uid()}@example.com`,
          enabled: true,
        })
        expect(res.code).toBe(0)
        configIds.push(res.data.id)
      }
      await page.screenshot({ path: 'test-results/UNP-2-01-创建多种类型.png', fullPage: false })
    })

    // ---- 2. 按媒体类型筛选 ----
    await test.step('按媒体类型筛选', async () => {
      const res = await API.get(page, `${API_BASE}/me/notify-configs?media_type=email`)
      expect(res.code).toBe(0)
      const list = res.data.list || res.data || []
      // 所有返回的配置应该是 email 类型
      for (const cfg of list) {
        expect(cfg.media_type).toBe('email')
      }
      await page.screenshot({ path: 'test-results/UNP-2-02-类型筛选.png', fullPage: false })
    })

    // ---- 3. 按 webhook 类型筛选 ----
    await test.step('按 webhook 类型筛选', async () => {
      const res = await API.get(page, `${API_BASE}/me/notify-configs?media_type=webhook`)
      expect(res.code).toBe(0)
      const list = res.data.list || res.data || []
      for (const cfg of list) {
        expect(cfg.media_type).toBe('webhook')
      }
      await page.screenshot({ path: 'test-results/UNP-2-03-webhook筛选.png', fullPage: false })
    })
  } finally {
    for (const id of configIds) {
      try {
        await API.del(page, `${API_BASE}/me/notify-configs/${id}`)
      } catch { /* ignore */ }
    }
  }
})

// ---------------------------------------------------------------------------
// UNP-3 用户通知偏好删除
// ---------------------------------------------------------------------------
test('UNP-3 用户通知偏好删除', async ({ authPage: page }) => {
  let configId: number | null = null

  try {
    // ---- 1. 创建偏好 ----
    await test.step('创建偏好', async () => {
      const res = await API.post(page, `${API_BASE}/me/notify-configs`, {
        media_type: 'email',
        target: `delete-test-${uid()}@example.com`,
        enabled: true,
      })
      expect(res.code).toBe(0)
      configId = res.data.id
      await page.screenshot({ path: 'test-results/UNP-3-01-创建偏好.png', fullPage: false })
    })

    // ---- 2. 删除偏好 ----
    await test.step('删除偏好', async () => {
      const res = await API.del(page, `${API_BASE}/me/notify-configs/${configId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/UNP-3-02-删除偏好.png', fullPage: false })
    })

    // ---- 3. 验证已删除 ----
    await test.step('验证已删除', async () => {
      const res = await API.get(page, `${API_BASE}/me/notify-configs`)
      expect(res.code).toBe(0)
      const list = res.data.list || res.data || []
      const found = list.find((c: any) => c.id === configId)
      expect(found).toBeFalsy()
      await page.screenshot({ path: 'test-results/UNP-3-03-验证删除.png', fullPage: false })
    })

    configId = null
  } finally {
    if (configId) {
      try {
        await API.del(page, `${API_BASE}/me/notify-configs/${configId}`)
      } catch { /* ignore */ }
    }
  }
})
