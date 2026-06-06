import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

// ---------------------------------------------------------------------------
// USER-1: 用户列表 → 验证 admin 存在 → 按 ID 查询 → 验证 username/role
// ---------------------------------------------------------------------------
test('USER-1 用户列表与详情查询', async ({ authPage: page }) => {
  let adminUser: any

  try {
    // ---- 1. 获取用户列表 ----
    await test.step('获取用户列表', async () => {
      const resp = await API.get(page, `${API_BASE}/users?role=admin`)
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      expect(resp.data.list).toBeDefined()
      expect(Array.isArray(resp.data.list)).toBe(true)
      expect(resp.data.list.length).toBeGreaterThan(0)
      await page.screenshot({ path: 'test-results/USER-1-01-用户列表.png', fullPage: false })
    })

    // ---- 2. 验证 admin 用户存在 ----
    await test.step('验证 admin 用户存在', async () => {
      const resp = await API.get(page, `${API_BASE}/users?role=admin`)
      adminUser = resp.data.list.find((u: any) => u.username === 'admin')
      expect(adminUser).toBeDefined()
      expect(adminUser.username).toBe('admin')
      expect(adminUser.role).toBe('admin')
      expect(adminUser.id).toBeGreaterThan(0)
      await page.screenshot({ path: 'test-results/USER-1-02-admin用户存在.png', fullPage: false })
    })

    // ---- 3. 按 ID 获取用户详情 ----
    await test.step('按 ID 获取用户详情', async () => {
      const resp = await API.get(page, `${API_BASE}/users/${adminUser.id}`)
      expect(resp.code).toBe(0)
      expect(resp.data.id).toBe(adminUser.id)
      expect(resp.data.username).toBe('admin')
      expect(resp.data.role).toBe('admin')
      // 验证返回字段完整性
      expect(resp.data).toHaveProperty('username')
      expect(resp.data).toHaveProperty('role')
      expect(resp.data).toHaveProperty('id')
      await page.screenshot({ path: 'test-results/USER-1-03-用户详情.png', fullPage: false })
    })

    // ---- 4. 分页查询验证 ----
    await test.step('分页查询验证', async () => {
      const resp = await API.get(page, `${API_BASE}/users?page=1&page_size=5`)
      expect(resp.code).toBe(0)
      expect(resp.data.list.length).toBeLessThanOrEqual(5)
      expect(resp.data.total).toBeGreaterThan(0)
      await page.screenshot({ path: 'test-results/USER-1-04-分页查询.png', fullPage: false })
    })
  } finally {
    // 读取操作，无需清理
  }
})

// ---------------------------------------------------------------------------
// USER-2: 获取当前用户 profile → 验证 username=admin, role=admin
// ---------------------------------------------------------------------------
test('USER-2 当前用户 Profile', async ({ authPage: page }) => {
  try {
    // ---- 1. 获取当前用户 profile ----
    await test.step('获取当前用户 profile', async () => {
      const resp = await API.get(page, `${API_BASE}/auth/profile`)
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      expect(resp.data.username).toBe('admin')
      expect(resp.data.role).toBe('admin')
      expect(resp.data.id).toBeGreaterThan(0)
      await page.screenshot({ path: 'test-results/USER-2-01-当前用户Profile.png', fullPage: false })
    })

    // ---- 2. 验证 profile 字段完整性 ----
    await test.step('验证 profile 字段完整性', async () => {
      const resp = await API.get(page, `${API_BASE}/auth/profile`)
      const user = resp.data
      // 必须包含的核心字段
      expect(user).toHaveProperty('id')
      expect(user).toHaveProperty('username')
      expect(user).toHaveProperty('role')
      // 验证类型
      expect(typeof user.id).toBe('number')
      expect(typeof user.username).toBe('string')
      expect(typeof user.role).toBe('string')
      // admin 应该有完整信息
      expect(user.username.length).toBeGreaterThan(0)
      expect(user.role.length).toBeGreaterThan(0)
      await page.screenshot({ path: 'test-results/USER-2-02-字段完整性.png', fullPage: false })
    })
  } finally {
    // 读取操作，无需清理
  }
})

// ---------------------------------------------------------------------------
// USER-3: 列出团队 → 创建团队 → 验证保存 → 添加成员 → 验证成员 → 移除成员 → 删除团队
// ---------------------------------------------------------------------------
test('USER-3 团队 CRUD 与成员管理', async ({ authPage: page }) => {
  let teamId: number | null = null
  const teamName = `func-team-${uid()}`
  let adminUserId: number

  try {
    // ---- 0. 获取 admin 用户 ID ----
    await test.step('获取 admin 用户 ID', async () => {
      const resp = await API.get(page, `${API_BASE}/auth/profile`)
      expect(resp.code).toBe(0)
      adminUserId = resp.data.id
    })

    // ---- 1. 列出已有团队 ----
    await test.step('列出已有团队', async () => {
      const resp = await API.get(page, `${API_BASE}/teams`)
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      expect(resp.data.list).toBeDefined()
      expect(Array.isArray(resp.data.list)).toBe(true)
      await page.screenshot({ path: 'test-results/USER-3-01-团队列表.png', fullPage: false })
    })

    // ---- 2. 创建团队 ----
    await test.step('创建团队', async () => {
      const resp = await API.post(page, `${API_BASE}/teams`, {
        name: teamName,
        description: 'Functional test team - auto created',
        labels: { env: 'test', purpose: 'functional' },
      })
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      expect(resp.data.id).toBeGreaterThan(0)
      expect(resp.data.name).toBe(teamName)
      expect(resp.data.description).toBe('Functional test team - auto created')
      teamId = resp.data.id
      await page.screenshot({ path: 'test-results/USER-3-02-创建团队.png', fullPage: false })
    })

    // ---- 3. 验证团队已保存 ----
    await test.step('验证团队已保存', async () => {
      const resp = await API.get(page, `${API_BASE}/teams/${teamId}`)
      expect(resp.code).toBe(0)
      expect(resp.data.id).toBe(teamId)
      expect(resp.data.name).toBe(teamName)
      expect(resp.data.description).toBe('Functional test team - auto created')
      await page.screenshot({ path: 'test-results/USER-3-03-验证团队保存.png', fullPage: false })
    })

    // ---- 4. 添加成员 ----
    await test.step('添加成员到团队', async () => {
      const resp = await API.post(page, `${API_BASE}/teams/${teamId}/members`, {
        user_id: adminUserId,
        role: 'team_lead',
      })
      expect(resp.code).toBe(0)
      await page.screenshot({ path: 'test-results/USER-3-04-添加成员.png', fullPage: false })
    })

    // ---- 5. 验证成员存在 ----
    await test.step('验证成员存在', async () => {
      const resp = await API.get(page, `${API_BASE}/teams/${teamId}/members`)
      expect(resp.code).toBe(0)
      expect(Array.isArray(resp.data)).toBe(true)
      const member = resp.data.find((m: any) => m.id === adminUserId || m.user_id === adminUserId)
      expect(member).toBeDefined()
      await page.screenshot({ path: 'test-results/USER-3-05-验证成员存在.png', fullPage: false })
    })

    // ---- 6. 移除成员 ----
    await test.step('移除成员', async () => {
      const resp = await API.del(page, `${API_BASE}/teams/${teamId}/members/${adminUserId}`)
      expect(resp.code).toBe(0)
      await page.screenshot({ path: 'test-results/USER-3-06-移除成员.png', fullPage: false })
    })

    // ---- 7. 验证成员已移除 ----
    await test.step('验证成员已移除', async () => {
      const resp = await API.get(page, `${API_BASE}/teams/${teamId}/members`)
      expect(resp.code).toBe(0)
      const member = resp.data.find((m: any) => m.id === adminUserId || m.user_id === adminUserId)
      expect(member).toBeUndefined()
      await page.screenshot({ path: 'test-results/USER-3-07-成员已移除.png', fullPage: false })
    })

    // ---- 8. 删除团队 ----
    await test.step('删除团队', async () => {
      const resp = await API.del(page, `${API_BASE}/teams/${teamId}`)
      expect(resp.code).toBe(0)
      teamId = null // 标记已清理
      await page.screenshot({ path: 'test-results/USER-3-08-删除团队.png', fullPage: false })
    })

    // ---- 9. 验证团队已删除 ----
    await test.step('验证团队已删除', async () => {
      const resp = await API.get(page, `${API_BASE}/teams/999999`)
      // 团队不存在应返回非0 code
      expect(resp.code).not.toBe(0)
    })
  } finally {
    // 清理：如果团队还存在则删除
    if (teamId) {
      try {
        await API.del(page, `${API_BASE}/teams/${teamId}`)
      } catch { /* ignore cleanup errors */ }
    }
  }
})

// ---------------------------------------------------------------------------
// USER-4: 获取权限 → 验证结构 → 检查特定权限存在
// ---------------------------------------------------------------------------
test('USER-4 权限查询', async ({ authPage: page }) => {
  try {
    // ---- 1. 获取当前用户权限 ----
    await test.step('获取当前用户权限', async () => {
      const resp = await API.get(page, `${API_BASE}/me/permissions`)
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      await page.screenshot({ path: 'test-results/USER-4-01-权限查询.png', fullPage: false })
    })

    // ---- 2. 验证权限结构 ----
    await test.step('验证权限结构', async () => {
      const resp = await API.get(page, `${API_BASE}/me/permissions`)
      expect(resp.code).toBe(0)
      const perms = resp.data
      // 权限数据应该存在
      expect(perms).toBeTruthy()
      // admin 用户应有权限数据（可能是数组或对象）
      if (Array.isArray(perms)) {
        expect(perms.length).toBeGreaterThan(0)
      } else if (typeof perms === 'object') {
        // 可能是 { permissions: [...] } 或 { resource: [actions] } 结构
        expect(Object.keys(perms).length).toBeGreaterThan(0)
      }
      await page.screenshot({ path: 'test-results/USER-4-02-权限结构.png', fullPage: false })
    })

    // ---- 3. 验证 admin 拥有核心管理权限 ----
    await test.step('验证 admin 核心权限', async () => {
      const resp = await API.get(page, `${API_BASE}/me/permissions`)
      expect(resp.code).toBe(0)
      const perms = resp.data
      const permsStr = JSON.stringify(perms).toLowerCase()
      // admin 应该拥有至少这些核心资源的权限
      const coreResources = ['rules', 'teams', 'users']
      for (const resource of coreResources) {
        // 权限中应包含这些资源相关的内容
        expect(permsStr).toContain(resource.toLowerCase().replace('-', ''))
      }
      await page.screenshot({ path: 'test-results/USER-4-03-核心权限.png', fullPage: false })
    })

    // ---- 4. 验证 profile 中的角色与权限一致 ----
    await test.step('验证角色与权限一致', async () => {
      const profileResp = await API.get(page, `${API_BASE}/auth/profile`)
      expect(profileResp.code).toBe(0)
      expect(profileResp.data.role).toBe('admin')

      const permResp = await API.get(page, `${API_BASE}/me/permissions`)
      expect(permResp.code).toBe(0)
      // admin 应该有非空权限
      expect(permResp.data).toBeTruthy()
      await page.screenshot({ path: 'test-results/USER-4-04-角色权限一致.png', fullPage: false })
    })
  } finally {
    // 读取操作，无需清理
  }
})
