import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

// ---------------------------------------------------------------------------
// UP-1 用户偏好 get
// ---------------------------------------------------------------------------
test('UP-1 用户偏好get', async ({ authPage: page }) => {
  // ---- 1. 获取用户偏好 ----
  await test.step('获取用户偏好', async () => {
    const res = await API.get(page, `${API_BASE}/me/preferences`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    await page.screenshot({ path: 'test-results/UP-1-01-获取偏好.png', fullPage: false })
  })

  // ---- 2. 验证偏好结构 ----
  await test.step('验证偏好结构', async () => {
    const res = await API.get(page, `${API_BASE}/me/preferences`)
    expect(res.code).toBe(0)
    const prefs = res.data
    // 偏好应为对象
    expect(typeof prefs).toBe('object')
    // 可能包含的常见偏好字段
    // language, theme, timezone 等
    await page.screenshot({ path: 'test-results/UP-1-02-偏好结构.png', fullPage: false })
  })

  // ---- 3. 再次获取确认一致性 ----
  await test.step('再次获取确认一致性', async () => {
    const res1 = await API.get(page, `${API_BASE}/me/preferences`)
    const res2 = await API.get(page, `${API_BASE}/me/preferences`)
    expect(res1.code).toBe(0)
    expect(res2.code).toBe(0)
    // 两次获取应返回相同数据
    expect(JSON.stringify(res1.data)).toBe(JSON.stringify(res2.data))
    await page.screenshot({ path: 'test-results/UP-1-03-一致性验证.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// UP-2 用户偏好 update
// ---------------------------------------------------------------------------
test('UP-2 用户偏好update', async ({ authPage: page }) => {
  let originalPrefs: any = null
  const tag = uid()

  try {
    // ---- 1. 获取当前偏好 ----
    await test.step('获取当前偏好', async () => {
      const res = await API.get(page, `${API_BASE}/me/preferences`)
      expect(res.code).toBe(0)
      originalPrefs = { ...res.data }
      await page.screenshot({ path: 'test-results/UP-2-01-获取当前偏好.png', fullPage: false })
    })

    // ---- 2. 更新偏好 ----
    await test.step('更新偏好', async () => {
      const res = await API.put(page, `${API_BASE}/me/preferences`, {
        ...originalPrefs,
        language: 'en-US',
        theme: 'dark',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/UP-2-02-更新偏好.png', fullPage: false })
    })

    // ---- 3. 验证更新 ----
    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/me/preferences`)
      expect(res.code).toBe(0)
      expect(res.data.language).toBe('en-US')
      expect(res.data.theme).toBe('dark')
      await page.screenshot({ path: 'test-results/UP-2-03-验证更新.png', fullPage: false })
    })

    // ---- 4. 再次更新 ----
    await test.step('再次更新偏好', async () => {
      const res = await API.put(page, `${API_BASE}/me/preferences`, {
        ...originalPrefs,
        language: 'zh-CN',
        theme: 'light',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/UP-2-04-再次更新.png', fullPage: false })
    })

    // ---- 5. 验证再次更新 ----
    await test.step('验证再次更新', async () => {
      const res = await API.get(page, `${API_BASE}/me/preferences`)
      expect(res.code).toBe(0)
      expect(res.data.language).toBe('zh-CN')
      expect(res.data.theme).toBe('light')
      await page.screenshot({ path: 'test-results/UP-2-05-再次验证.png', fullPage: false })
    })
  } finally {
    // 恢复原始偏好
    if (originalPrefs) {
      try {
        await API.put(page, `${API_BASE}/me/preferences`, originalPrefs)
      } catch { /* ignore */ }
    }
  }
})
