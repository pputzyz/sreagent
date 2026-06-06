import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

const API_BASE = '/api/v1'

// ---------------------------------------------------------------------------
// SET-1: SMTP 配置 — 获取结构 → 更新配置 → 验证保存
// ---------------------------------------------------------------------------
test('SET-1 SMTP 配置管理', async ({ authPage: page }) => {
  let originalConfig: any

  try {
    // ---- 1. 获取当前 SMTP 配置 ----
    await test.step('获取 SMTP 配置', async () => {
      const resp = await API.get(page, `${API_BASE}/settings/smtp`)
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      originalConfig = { ...resp.data }
      await page.screenshot({ path: 'test-results/SET-1-01-SMTP配置.png', fullPage: false })
    })

    // ---- 2. 验证配置结构 ----
    await test.step('验证 SMTP 配置结构', async () => {
      const resp = await API.get(page, `${API_BASE}/settings/smtp`)
      const cfg = resp.data
      // 必须包含的字段
      expect(cfg).toHaveProperty('smtp_host')
      expect(cfg).toHaveProperty('smtp_port')
      expect(cfg).toHaveProperty('smtp_tls')
      expect(cfg).toHaveProperty('username')
      expect(cfg).toHaveProperty('from')
      expect(cfg).toHaveProperty('enabled')
      // 类型检查
      expect(typeof cfg.smtp_port).toBe('number')
      expect(typeof cfg.smtp_tls).toBe('boolean')
      expect(typeof cfg.enabled).toBe('boolean')
      await page.screenshot({ path: 'test-results/SET-1-02-结构验证.png', fullPage: false })
    })

    // ---- 3. 更新 SMTP 配置 ----
    await test.step('更新 SMTP 配置', async () => {
      const resp = await API.put(page, `${API_BASE}/settings/smtp`, {
        smtp_host: 'smtp.test.example.com',
        smtp_port: 587,
        smtp_tls: true,
        username: 'test@test.example.com',
        from: 'noreply@test.example.com',
        enabled: false,
      })
      expect(resp.code).toBe(0)
      await page.screenshot({ path: 'test-results/SET-1-03-更新SMTP.png', fullPage: false })
    })

    // ---- 4. 验证配置已保存 ----
    await test.step('验证 SMTP 配置已保存', async () => {
      const resp = await API.get(page, `${API_BASE}/settings/smtp`)
      expect(resp.code).toBe(0)
      expect(resp.data.smtp_host).toBe('smtp.test.example.com')
      expect(resp.data.smtp_port).toBe(587)
      expect(resp.data.smtp_tls).toBe(true)
      expect(resp.data.from).toBe('noreply@test.example.com')
      expect(resp.data.enabled).toBe(false)
      await page.screenshot({ path: 'test-results/SET-1-04-验证保存.png', fullPage: false })
    })
  } finally {
    // 恢复原始配置
    if (originalConfig) {
      try {
        await API.put(page, `${API_BASE}/settings/smtp`, {
          smtp_host: originalConfig.smtp_host,
          smtp_port: originalConfig.smtp_port,
          smtp_tls: originalConfig.smtp_tls,
          username: originalConfig.username,
          from: originalConfig.from,
          enabled: originalConfig.enabled,
        })
      } catch { /* ignore restore errors */ }
    }
  }
})

// ---------------------------------------------------------------------------
// SET-2: 安全配置 — 获取结构 → 更新 JWT 过期时间 → 验证保存
// ---------------------------------------------------------------------------
test('SET-2 安全配置管理', async ({ authPage: page }) => {
  let originalConfig: any

  try {
    // ---- 1. 获取当前安全配置 ----
    await test.step('获取安全配置', async () => {
      const resp = await API.get(page, `${API_BASE}/settings/security`)
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      originalConfig = { ...resp.data }
      await page.screenshot({ path: 'test-results/SET-2-01-安全配置.png', fullPage: false })
    })

    // ---- 2. 验证配置结构 ----
    await test.step('验证安全配置结构', async () => {
      const resp = await API.get(page, `${API_BASE}/settings/security`)
      const cfg = resp.data
      expect(cfg).toHaveProperty('jwt_expire_seconds')
      expect(typeof cfg.jwt_expire_seconds).toBe('number')
      expect(cfg.jwt_expire_seconds).toBeGreaterThan(0)
      await page.screenshot({ path: 'test-results/SET-2-02-结构验证.png', fullPage: false })
    })

    // ---- 3. 更新 JWT 过期时间 ----
    await test.step('更新 JWT 过期时间', async () => {
      const resp = await API.put(page, `${API_BASE}/settings/security`, {
        jwt_expire_seconds: 7200,
      })
      expect(resp.code).toBe(0)
      await page.screenshot({ path: 'test-results/SET-2-03-更新JWT.png', fullPage: false })
    })

    // ---- 4. 验证配置已保存 ----
    await test.step('验证 JWT 过期时间已保存', async () => {
      const resp = await API.get(page, `${API_BASE}/settings/security`)
      expect(resp.code).toBe(0)
      expect(resp.data.jwt_expire_seconds).toBe(7200)
      await page.screenshot({ path: 'test-results/SET-2-04-验证保存.png', fullPage: false })
    })
  } finally {
    // 恢复原始配置
    if (originalConfig) {
      try {
        await API.put(page, `${API_BASE}/settings/security`, {
          jwt_expire_seconds: originalConfig.jwt_expire_seconds,
        })
      } catch { /* ignore restore errors */ }
    }
  }
})

// ---------------------------------------------------------------------------
// SET-3: AI 配置 — 获取 → 验证 enabled → 更新 model → 验证保存
// ---------------------------------------------------------------------------
test('SET-3 AI 配置管理', async ({ authPage: page }) => {
  let originalConfig: any

  try {
    // ---- 1. 获取当前 AI 配置 ----
    await test.step('获取 AI 配置', async () => {
      const resp = await API.get(page, `${API_BASE}/ai/config`)
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      originalConfig = { ...resp.data }
      await page.screenshot({ path: 'test-results/SET-3-01-AI配置.png', fullPage: false })
    })

    // ---- 2. 验证 AI 配置结构与 enabled 状态 ----
    await test.step('验证 AI 配置结构', async () => {
      const resp = await API.get(page, `${API_BASE}/ai/config`)
      const cfg = resp.data
      // 必须包含的字段
      expect(cfg).toHaveProperty('provider')
      expect(cfg).toHaveProperty('model')
      expect(cfg).toHaveProperty('enabled')
      expect(cfg).toHaveProperty('temperature')
      expect(cfg).toHaveProperty('max_tokens')
      // 类型检查
      expect(typeof cfg.provider).toBe('string')
      expect(typeof cfg.model).toBe('string')
      expect(typeof cfg.enabled).toBe('boolean')
      expect(typeof cfg.temperature).toBe('number')
      expect(typeof cfg.max_tokens).toBe('number')
      await page.screenshot({ path: 'test-results/SET-3-02-结构验证.png', fullPage: false })
    })

    // ---- 3. 更新 model 名称 ----
    await test.step('更新 AI model', async () => {
      const resp = await API.put(page, `${API_BASE}/ai/config`, {
        model: 'gpt-4o-mini-test',
      })
      expect(resp.code).toBe(0)
      await page.screenshot({ path: 'test-results/SET-3-03-更新Model.png', fullPage: false })
    })

    // ---- 4. 验证 model 已保存 ----
    await test.step('验证 AI model 已保存', async () => {
      const resp = await API.get(page, `${API_BASE}/ai/config`)
      expect(resp.code).toBe(0)
      expect(resp.data.model).toBe('gpt-4o-mini-test')
      // 其他字段应保持不变
      expect(resp.data.provider).toBe(originalConfig.provider)
      expect(resp.data.enabled).toBe(originalConfig.enabled)
      await page.screenshot({ path: 'test-results/SET-3-04-验证保存.png', fullPage: false })
    })
  } finally {
    // 恢复原始配置
    if (originalConfig) {
      try {
        await API.put(page, `${API_BASE}/ai/config`, {
          model: originalConfig.model,
        })
      } catch { /* ignore restore errors */ }
    }
  }
})

// ---------------------------------------------------------------------------
// SET-4: 站点信息 — 获取结构 → 更新 site_name → 验证保存
// ---------------------------------------------------------------------------
test('SET-4 站点信息管理', async ({ authPage: page }) => {
  let originalConfig: any

  try {
    // ---- 1. 获取当前站点信息 ----
    await test.step('获取站点信息', async () => {
      const resp = await API.get(page, `${API_BASE}/settings/site-info`)
      expect(resp.code).toBe(0)
      expect(resp.data).toBeDefined()
      originalConfig = { ...resp.data }
      await page.screenshot({ path: 'test-results/SET-4-01-站点信息.png', fullPage: false })
    })

    // ---- 2. 验证站点信息结构 ----
    await test.step('验证站点信息结构', async () => {
      const resp = await API.get(page, `${API_BASE}/settings/site-info`)
      const cfg = resp.data
      // 必须包含的字段
      expect(cfg).toHaveProperty('site_name')
      expect(cfg).toHaveProperty('logo_url')
      expect(cfg).toHaveProperty('favicon_url')
      expect(cfg).toHaveProperty('login_title')
      expect(cfg).toHaveProperty('login_subtitle')
      expect(cfg).toHaveProperty('footer_text')
      // 类型检查
      expect(typeof cfg.site_name).toBe('string')
      expect(typeof cfg.logo_url).toBe('string')
      expect(typeof cfg.favicon_url).toBe('string')
      expect(typeof cfg.login_title).toBe('string')
      await page.screenshot({ path: 'test-results/SET-4-02-结构验证.png', fullPage: false })
    })

    // ---- 3. 更新站点名称 ----
    await test.step('更新站点名称', async () => {
      const newSiteName = `SRE Agent Test ${Date.now()}`
      const resp = await API.put(page, `${API_BASE}/settings/site-info`, {
        ...originalConfig,
        site_name: newSiteName,
      })
      expect(resp.code).toBe(0)
      await page.screenshot({ path: 'test-results/SET-4-03-更新站点名.png', fullPage: false })
    })

    // ---- 4. 验证站点名称已保存 ----
    await test.step('验证站点名称已保存', async () => {
      const resp = await API.get(page, `${API_BASE}/settings/site-info`)
      expect(resp.code).toBe(0)
      expect(resp.data.site_name).toContain('SRE Agent Test')
      // 其他字段应保持不变
      expect(resp.data.login_title).toBe(originalConfig.login_title)
      await page.screenshot({ path: 'test-results/SET-4-04-验证保存.png', fullPage: false })
    })
  } finally {
    // 恢复原始配置
    if (originalConfig) {
      try {
        await API.put(page, `${API_BASE}/settings/site-info`, {
          site_name: originalConfig.site_name,
          logo_url: originalConfig.logo_url,
          favicon_url: originalConfig.favicon_url,
          login_title: originalConfig.login_title,
          login_subtitle: originalConfig.login_subtitle,
          footer_text: originalConfig.footer_text,
          custom_css: originalConfig.custom_css || '',
        })
      } catch { /* ignore restore errors */ }
    }
  }
})
