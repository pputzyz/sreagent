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
  const mediaType = `test_webhook_${uid()}`

  try {
    // ---- 1. 获取当前通知偏好列表 ----
    await test.step('获取当前通知偏好', async () => {
      const res = await API.get(page, `${API_BASE}/me/notify-configs`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      // List returns array directly
      expect(Array.isArray(res.data)).toBe(true)
      await page.screenshot({ path: 'test-results/UNP-1-01-获取偏好.png', fullPage: false })
    })

    // ---- 2. 创建通知偏好 (PUT upsert) ----
    await test.step('创建通知偏好', async () => {
      const res = await API.put(page, `${API_BASE}/me/notify-configs`, {
        media_type: mediaType,
        config: JSON.stringify({ url: `https://example.com/hook-${uid()}` }),
        is_enabled: true,
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeTruthy()
      expect(res.data.id).toBeGreaterThan(0)
      await page.screenshot({ path: 'test-results/UNP-1-02-创建偏好.png', fullPage: false })
    })

    // ---- 3. 验证创建成功 ----
    await test.step('验证创建成功', async () => {
      const res = await API.get(page, `${API_BASE}/me/notify-configs`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      const found = list.find((c: any) => c.media_type === mediaType)
      expect(found).toBeTruthy()
      expect(found.is_enabled).toBe(true)
      await page.screenshot({ path: 'test-results/UNP-1-03-验证创建.png', fullPage: false })
    })

    // ---- 4. 更新通知偏好 (PUT upsert with same media_type) ----
    await test.step('更新通知偏好', async () => {
      const res = await API.put(page, `${API_BASE}/me/notify-configs`, {
        media_type: mediaType,
        config: JSON.stringify({ url: `https://example.com/updated-hook-${uid()}` }),
        is_enabled: false,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/UNP-1-04-更新偏好.png', fullPage: false })
    })

    // ---- 5. 验证更新 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/me/notify-configs`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      const found = list.find((c: any) => c.media_type === mediaType)
      expect(found).toBeTruthy()
      expect(found.is_enabled).toBe(false)
      await page.screenshot({ path: 'test-results/UNP-1-05-验证更新.png', fullPage: false })
    })
  } finally {
    // cleanup: delete the config by media_type
    try {
      await API.del(page, `${API_BASE}/me/notify-configs/${mediaType}`)
    } catch { /* ignore */ }
  }
})

// ---------------------------------------------------------------------------
// UNP-2 用户通知偏好按媒体类型
// ---------------------------------------------------------------------------
test('UNP-2 用户通知偏好按媒体类型', async ({ authPage: page }) => {
  const mediaTypes = [`email_test_${uid()}`, `webhook_test_${uid()}`]

  try {
    // ---- 1. 创建多个不同类型的偏好 ----
    await test.step('创建多种类型偏好', async () => {
      for (const type of mediaTypes) {
        const res = await API.put(page, `${API_BASE}/me/notify-configs`, {
          media_type: type,
          config: JSON.stringify({ url: `${type}@example.com` }),
          is_enabled: true,
        })
        expect(res.code).toBe(0)
      }
      await page.screenshot({ path: 'test-results/UNP-2-01-创建多种类型.png', fullPage: false })
    })

    // ---- 2. 验证列表包含所有类型 ----
    await test.step('验证列表包含所有类型', async () => {
      const res = await API.get(page, `${API_BASE}/me/notify-configs`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      for (const type of mediaTypes) {
        const found = list.find((c: any) => c.media_type === type)
        expect(found).toBeTruthy()
      }
      await page.screenshot({ path: 'test-results/UNP-2-02-类型验证.png', fullPage: false })
    })
  } finally {
    for (const type of mediaTypes) {
      try {
        await API.del(page, `${API_BASE}/me/notify-configs/${type}`)
      } catch { /* ignore */ }
    }
  }
})

// ---------------------------------------------------------------------------
// UNP-3 用户通知偏好删除
// ---------------------------------------------------------------------------
test('UNP-3 用户通知偏好删除', async ({ authPage: page }) => {
  const mediaType = `delete_test_${uid()}`

  try {
    // ---- 1. 创建偏好 ----
    await test.step('创建偏好', async () => {
      const res = await API.put(page, `${API_BASE}/me/notify-configs`, {
        media_type: mediaType,
        config: JSON.stringify({ url: `https://example.com/delete-test` }),
        is_enabled: true,
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/UNP-3-01-创建偏好.png', fullPage: false })
    })

    // ---- 2. 删除偏好 (DELETE by mediaType) ----
    await test.step('删除偏好', async () => {
      const res = await API.del(page, `${API_BASE}/me/notify-configs/${mediaType}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/UNP-3-02-删除偏好.png', fullPage: false })
    })

    // ---- 3. 验证已删除 ----
    await test.step('验证已删除', async () => {
      const res = await API.get(page, `${API_BASE}/me/notify-configs`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      const found = list.find((c: any) => c.media_type === mediaType)
      expect(found).toBeFalsy()
      await page.screenshot({ path: 'test-results/UNP-3-03-验证删除.png', fullPage: false })
    })
  } finally {
    try {
      await API.del(page, `${API_BASE}/me/notify-configs/${mediaType}`)
    } catch { /* ignore */ }
  }
})
