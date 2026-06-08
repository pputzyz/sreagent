import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

// ---------------------------------------------------------------------------
// LR-1: 标签注册表 keys 查询
// ---------------------------------------------------------------------------
test('LR-1 标签注册表 keys查询', async ({ authPage: page }) => {
  await test.step('获取标签键列表', async () => {
    const res = await API.get(page, `${API_BASE}/label-registry/keys`)
    expect(res.code).toBe(0)
    expect(res.data).toBeDefined()
    await page.screenshot({ path: 'test-results/LR-1-01-标签键列表.png', fullPage: false })
  })

  await test.step('验证标签键列表结构', async () => {
    const res = await API.get(page, `${API_BASE}/label-registry/keys`)
    expect(res.code).toBe(0)
    const keys = Array.isArray(res.data) ? res.data : res.data.list || []
    expect(Array.isArray(keys)).toBe(true)
    await page.screenshot({ path: 'test-results/LR-1-02-键结构.png', fullPage: false })
  })

  await test.step('带搜索条件查询标签键', async () => {
    const res = await API.get(page, `${API_BASE}/label-registry/keys?keyword=env`)
    expect(res.code).toBe(0)
    await page.screenshot({ path: 'test-results/LR-1-03-搜索标签键.png', fullPage: false })
  })
})

// ---------------------------------------------------------------------------
// LR-2: 标签注册表 values 查询
// ---------------------------------------------------------------------------
test('LR-2 标签注册表 values查询', async ({ authPage: page }) => {
  let labelKey: string | undefined

  await test.step('获取标签键列表', async () => {
    const res = await API.get(page, `${API_BASE}/label-registry/keys`)
    expect(res.code).toBe(0)
    const keys = Array.isArray(res.data) ? res.data : res.data.list || []
    if (keys.length > 0) {
      labelKey = typeof keys[0] === 'string' ? keys[0] : keys[0].key || keys[0].name
    }
    await page.screenshot({ path: 'test-results/LR-2-01-获取标签键.png', fullPage: false })
  })

  if (labelKey) {
    await test.step('查询标签值', async () => {
      const res = await API.get(page, `${API_BASE}/label-registry/values?key=${encodeURIComponent(labelKey)}`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/LR-2-02-标签值列表.png', fullPage: false })
    })

    await test.step('验证标签值结构', async () => {
      const res = await API.get(page, `${API_BASE}/label-registry/values?key=${encodeURIComponent(labelKey)}`)
      expect(res.code).toBe(0)
      const values = Array.isArray(res.data) ? res.data : res.data.list || []
      expect(Array.isArray(values)).toBe(true)
      await page.screenshot({ path: 'test-results/LR-2-03-值结构.png', fullPage: false })
    })
  } else {
    await test.step('无标签键 — 跳过值查询', async () => {
      await page.screenshot({ path: 'test-results/LR-2-02-无标签键.png', fullPage: false })
    })
  }
})
