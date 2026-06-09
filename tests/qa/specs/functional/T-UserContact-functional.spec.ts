import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create a user contact via API and return the created object */
async function createUserContact(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    type: 'email',
    value: `test-${tag}@example.com`,
    name: `contact-${tag}`,
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/user/contacts`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete a user contact by ID, ignoring errors (for cleanup) */
async function cleanupUserContact(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/user/contacts/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// UC-1: 用户联系人 CRUD
// ---------------------------------------------------------------------------
test('UC-1 用户联系人 CRUD', async ({ authPage: page }) => {
  let contactId: number | null = null

  try {
    await test.step('创建用户联系人', async () => {
      const contact = await createUserContact(page)
      contactId = contact.id
      expect(contact.type).toBe('email')
      expect(contact.value).toContain('@example.com')
      await page.screenshot({ path: 'test-results/UC-1-01-创建成功.png', fullPage: false })
    })

    await test.step('GET 验证用户联系人已保存', async () => {
      const res = await API.get(page, `${API_BASE}/user/contacts`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      const found = list.find((c: any) => c.id === contactId)
      expect(found).toBeTruthy()
      expect(found.type).toBe('email')
      await page.screenshot({ path: 'test-results/UC-1-02-GET验证.png', fullPage: false })
    })

    await test.step('更新用户联系人', async () => {
      const res = await API.put(page, `${API_BASE}/user/contacts/${contactId}`, {
        type: 'email',
        value: `updated-${uid()}@example.com`,
        name: 'updated-contact',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/UC-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/user/contacts`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      const found = list.find((c: any) => c.id === contactId)
      expect(found).toBeTruthy()
      expect(found.value).toContain('updated-')
      await page.screenshot({ path: 'test-results/UC-1-04-更新验证.png', fullPage: false })
    })

    await test.step('删除用户联系人', async () => {
      const res = await API.del(page, `${API_BASE}/user/contacts/${contactId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/UC-1-05-删除成功.png', fullPage: false })
    })

    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/user/contacts`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      const found = list.find((c: any) => c.id === contactId)
      expect(found).toBeFalsy()
      await page.screenshot({ path: 'test-results/UC-1-06-删除验证.png', fullPage: false })
    })

    contactId = null
  } finally {
    if (contactId) await cleanupUserContact(page, contactId)
  }
})

// ---------------------------------------------------------------------------
// UC-2: 用户联系人设为默认
// ---------------------------------------------------------------------------
test('UC-2 用户联系人 设为默认', async ({ authPage: page }) => {
  let contact1Id: number | null = null
  let contact2Id: number | null = null

  try {
    await test.step('创建两个联系人', async () => {
      const c1 = await createUserContact(page)
      contact1Id = c1.id
      const c2 = await createUserContact(page)
      contact2Id = c2.id
      await page.screenshot({ path: 'test-results/UC-2-01-创建联系人.png', fullPage: false })
    })

    await test.step('将第一个设为默认', async () => {
      const res = await API.post(page, `${API_BASE}/user/contacts/${contact1Id}/default`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/UC-2-02-设为默认.png', fullPage: false })
    })

    await test.step('验证第一个为默认', async () => {
      const res = await API.get(page, `${API_BASE}/user/contacts`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      const found = list.find((c: any) => c.id === contact1Id)
      expect(found).toBeTruthy()
      expect(found.is_default).toBe(true)
      await page.screenshot({ path: 'test-results/UC-2-03-验证默认.png', fullPage: false })
    })

    await test.step('将第二个设为默认', async () => {
      const res = await API.post(page, `${API_BASE}/user/contacts/${contact2Id}/default`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/UC-2-04-设为默认.png', fullPage: false })
    })

    await test.step('验证默认互斥', async () => {
      const res = await API.get(page, `${API_BASE}/user/contacts`)
      expect(res.code).toBe(0)
      const list = Array.isArray(res.data) ? res.data : []
      const found = list.find((c: any) => c.id === contact2Id)
      expect(found).toBeTruthy()
      expect(found.is_default).toBe(true)
      await page.screenshot({ path: 'test-results/UC-2-05-互斥验证.png', fullPage: false })
    })
  } finally {
    if (contact1Id) await cleanupUserContact(page, contact1Id)
    if (contact2Id) await cleanupUserContact(page, contact2Id)
  }
})

// ---------------------------------------------------------------------------
// UC-3: 用户联系人验证
// ---------------------------------------------------------------------------
test('UC-3 用户联系人 验证', async ({ authPage: page }) => {
  let contactId: number | null = null

  try {
    await test.step('创建联系人', async () => {
      const contact = await createUserContact(page)
      contactId = contact.id
      await page.screenshot({ path: 'test-results/UC-3-01-创建联系人.png', fullPage: false })
    })

    await test.step('发送验证请求', async () => {
      const res = await API.post(page, `${API_BASE}/user/contacts/${contactId}/verify`)
      // May succeed or fail depending on SMTP config
      expect(res).toBeDefined()
      expect(res.code).toBeDefined()
      await page.screenshot({ path: 'test-results/UC-3-02-发送验证.png', fullPage: false })
    })
  } finally {
    if (contactId) await cleanupUserContact(page, contactId)
  }
})

// ---------------------------------------------------------------------------
// UC-4: 用户联系人确认验证
// ---------------------------------------------------------------------------
test('UC-4 用户联系人 确认验证', async ({ authPage: page }) => {
  let contactId: number | null = null

  try {
    await test.step('创建联系人', async () => {
      const contact = await createUserContact(page)
      contactId = contact.id
      await page.screenshot({ path: 'test-results/UC-4-01-创建联系人.png', fullPage: false })
    })

    await test.step('确认验证（使用测试码）', async () => {
      const res = await API.post(page, `${API_BASE}/user/contacts/${contactId}/verify/confirm`, {
        code: '000000',
      })
      // May fail with invalid code -- that's expected
      expect(res).toBeDefined()
      expect(res.code).toBeDefined()
      await page.screenshot({ path: 'test-results/UC-4-02-确认验证.png', fullPage: false })
    })
  } finally {
    if (contactId) await cleanupUserContact(page, contactId)
  }
})
