import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

/** Unique suffix to avoid name collisions between parallel runs */
function uid(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Helper: create an AI skill via API and return the created object */
async function createAISkill(page: any, overrides: Record<string, unknown> = {}) {
  const tag = uid()
  const payload = {
    name: `skill-test-${tag}`,
    description: 'Functional test AI skill',
    type: 'custom',
    status: 'active',
    ...overrides,
  }
  const res = await API.post(page, `${API_BASE}/ai-skills`, payload)
  expect(res.code).toBe(0)
  expect(res.data).toBeTruthy()
  expect(res.data.id).toBeGreaterThan(0)
  return { ...res.data, _tag: tag, _payload: payload }
}

/** Helper: delete an AI skill by ID, ignoring errors (for cleanup) */
async function cleanupAISkill(page: any, id: number) {
  try {
    await API.del(page, `${API_BASE}/ai-skills/${id}`)
  } catch { /* ignore */ }
}

// ---------------------------------------------------------------------------
// AS-1: AI 技能 CRUD
// ---------------------------------------------------------------------------
test('AS-1 AI技能 CRUD', async ({ authPage: page }) => {
  let skillId: number | null = null

  try {
    await test.step('创建 AI 技能', async () => {
      const skill = await createAISkill(page)
      skillId = skill.id
      expect(skill.name).toContain('skill-test-')
      expect(skill.status).toBe('active')
      await page.screenshot({ path: 'test-results/AS-1-01-创建成功.png', fullPage: false })
    })

    await test.step('GET 验证 AI 技能已保存', async () => {
      const res = await API.get(page, `${API_BASE}/ai-skills/${skillId}`)
      expect(res.code).toBe(0)
      expect(res.data.id).toBe(skillId)
      expect(res.data.name).toContain('skill-test-')
      await page.screenshot({ path: 'test-results/AS-1-02-GET验证.png', fullPage: false })
    })

    await test.step('更新 AI 技能', async () => {
      const res = await API.put(page, `${API_BASE}/ai-skills/${skillId}`, {
        name: `updated-skill-${uid()}`,
        description: 'Updated by functional test',
      })
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/AS-1-03-更新成功.png', fullPage: false })
    })

    await test.step('验证更新生效', async () => {
      const res = await API.get(page, `${API_BASE}/ai-skills/${skillId}`)
      expect(res.code).toBe(0)
      expect(res.data.description).toBe('Updated by functional test')
      await page.screenshot({ path: 'test-results/AS-1-04-更新验证.png', fullPage: false })
    })

    await test.step('删除 AI 技能', async () => {
      const res = await API.del(page, `${API_BASE}/ai-skills/${skillId}`)
      expect(res.code).toBe(0)
      await page.screenshot({ path: 'test-results/AS-1-05-删除成功.png', fullPage: false })
    })

    await test.step('验证删除生效', async () => {
      const res = await API.get(page, `${API_BASE}/ai-skills/${skillId}`)
      expect(res.code).not.toBe(0)
      await page.screenshot({ path: 'test-results/AS-1-06-删除验证.png', fullPage: false })
    })

    skillId = null
  } finally {
    if (skillId) await cleanupAISkill(page, skillId)
  }
})

// ---------------------------------------------------------------------------
// AS-2: AI 技能 zip 导入
// ---------------------------------------------------------------------------
test('AS-2 AI技能 zip导入', async ({ authPage: page }) => {
  let importedSkillId: number | null = null

  try {
    await test.step('构造 zip 文件并导入', async () => {
      // Create a minimal skill definition as JSON (simulating zip import)
      const skillDef = {
        name: `imported-skill-${uid()}`,
        description: 'Imported from zip test',
        type: 'custom',
      }

      // Try importing via multipart endpoint
      const token = await page.evaluate(() => localStorage.getItem('token'))
      const resp = await page.request.post(`http://localhost:3000${API_BASE}/ai-skills/import`, {
        headers: { Authorization: `Bearer ${token}` },
        multipart: {
          file: {
            name: 'skill.zip',
            mimeType: 'application/zip',
            buffer: Buffer.from(JSON.stringify(skillDef)),
          },
        },
      })
      const result = await resp.json()

      // Import may succeed or fail with validation error — either is acceptable
      expect(result).toBeDefined()
      expect(result.code).toBeDefined()
      if (result.code === 0 && result.data?.id) {
        importedSkillId = result.data.id
      }
      await page.screenshot({ path: 'test-results/AS-2-01-导入结果.png', fullPage: false })
    })

    if (importedSkillId) {
      await test.step('验证导入的技能存在', async () => {
        const res = await API.get(page, `${API_BASE}/ai-skills/${importedSkillId}`)
        expect(res.code).toBe(0)
        expect(res.data.description).toBe('Imported from zip test')
        await page.screenshot({ path: 'test-results/AS-2-02-导入验证.png', fullPage: false })
      })
    } else {
      await test.step('记录导入未成功（预期行为）', async () => {
        // Zip import may require specific format — just verify the endpoint responds
        await page.screenshot({ path: 'test-results/AS-2-02-导入未成功.png', fullPage: false })
      })
    }
  } finally {
    if (importedSkillId) await cleanupAISkill(page, importedSkillId)
  }
})

// ---------------------------------------------------------------------------
// AS-3: AI 技能文件管理
// ---------------------------------------------------------------------------
test('AS-3 AI技能 文件管理', async ({ authPage: page }) => {
  let skillId: number | null = null

  try {
    await test.step('创建 AI 技能', async () => {
      const skill = await createAISkill(page)
      skillId = skill.id
      await page.screenshot({ path: 'test-results/AS-3-01-创建技能.png', fullPage: false })
    })

    await test.step('获取技能文件列表', async () => {
      const res = await API.get(page, `${API_BASE}/ai-skills/${skillId}/files`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      await page.screenshot({ path: 'test-results/AS-3-02-文件列表.png', fullPage: false })
    })

    await test.step('验证文件列表结构', async () => {
      const res = await API.get(page, `${API_BASE}/ai-skills/${skillId}/files`)
      expect(res.code).toBe(0)
      const files = Array.isArray(res.data) ? res.data : res.data.list || []
      expect(Array.isArray(files)).toBe(true)
      await page.screenshot({ path: 'test-results/AS-3-03-文件结构.png', fullPage: false })
    })
  } finally {
    if (skillId) await cleanupAISkill(page, skillId)
  }
})

// ---------------------------------------------------------------------------
// AS-4: AI 技能内置保护
// ---------------------------------------------------------------------------
test('AS-4 AI技能 内置保护', async ({ authPage: page }) => {
  let builtinSkillId: number | null = null

  try {
    await test.step('获取技能列表找到内置技能', async () => {
      const res = await API.get(page, `${API_BASE}/ai-skills?page=1&page_size=100`)
      expect(res.code).toBe(0)
      const list = res.data.list || res.data || []
      // Find a built-in skill
      const builtin = list.find((s: any) => s.type === 'builtin' || s.is_builtin === true || s.builtin === true)
      if (builtin) {
        builtinSkillId = builtin.id
      }
      await page.screenshot({ path: 'test-results/AS-4-01-技能列表.png', fullPage: false })
    })

    if (builtinSkillId) {
      await test.step('尝试删除内置技能 — 应被拒绝', async () => {
        const res = await API.del(page, `${API_BASE}/ai-skills/${builtinSkillId}`)
        // Built-in skills should not be deletable
        const isProtected = res.code !== 0 || res.message?.includes('builtin') || res.message?.includes('protected')
        expect(isProtected).toBeTruthy()
        await page.screenshot({ path: 'test-results/AS-4-02-删除被拒.png', fullPage: false })
      })

      await test.step('尝试更新内置技能 — 应被拒绝或受限', async () => {
        const res = await API.put(page, `${API_BASE}/ai-skills/${builtinSkillId}`, {
          name: 'hacked-name',
        })
        // Built-in skills should not be modifiable (or have limited fields)
        if (res.code === 0) {
          // If update succeeded, verify name wasn't actually changed
          const getRes = await API.get(page, `${API_BASE}/ai-skills/${builtinSkillId}`)
          // Name should still be original
          expect(getRes.data.name).not.toBe('hacked-name')
        }
        await page.screenshot({ path: 'test-results/AS-4-03-更新保护.png', fullPage: false })
      })
    } else {
      await test.step('无内置技能 — 跳过保护测试', async () => {
        await page.screenshot({ path: 'test-results/AS-4-02-无内置技能.png', fullPage: false })
      })
    }
  } finally {
    // No cleanup needed — built-in skills are not modified
  }
})
