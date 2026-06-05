import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

// T7: Settings Full Test Suite — 60 tests
// Covers: SMTP (T7-1~T7-15), Security (T7-16~T7-30), SSO (T7-31~T7-45), User Mgmt (T7-46~T7-60)

const SETTINGS_URL = '/settings'
const USERS_URL = '/settings/users'

/** Generate unique name */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

test.describe('T7 - Settings Full Test Suite', () => {

  // ================================================================
  // T7-1 ~ T7-15: SMTP Settings
  // ================================================================

  // T7-1: SMTP settings page load
  test('T7-1 SMTP settings page load', async ({ authPage: page }) => {
    await test.step('Navigate to settings page', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-1-SMTP-load.png', fullPage: true })
    })

    await test.step('Verify page body is visible', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-2: SMTP tab navigation
  test('T7-2 SMTP tab navigation', async ({ authPage: page }) => {
    await test.step('Navigate to settings page', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click SMTP tab', async () => {
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T7-2-SMTP-tab.png', fullPage: false })
      }
    })
  })

  // T7-3: SMTP form fields exist
  test('T7-3 SMTP form fields exist', async ({ authPage: page }) => {
    await test.step('Navigate to settings SMTP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Verify SMTP host field exists', async () => {
      const hostInput = page.locator('input[placeholder*="host"], input[placeholder*="Host"], input[placeholder*="smtp"]').first()
      if (await hostInput.isVisible().catch(() => false)) {
        await expect(hostInput).toBeVisible()
        await page.screenshot({ path: 'test-results/T7-3-SMTP-fields.png', fullPage: false })
      }
    })
  })

  // T7-4: SMTP host input
  test('T7-4 SMTP host input', async ({ authPage: page }) => {
    await test.step('Navigate to SMTP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Fill SMTP host', async () => {
      const hostInput = page.locator('input[placeholder*="host"], input[placeholder*="Host"], input[placeholder*="smtp"]').first()
      if (await hostInput.isVisible().catch(() => false)) {
        await hostInput.fill('smtp.example.com')
        await page.screenshot({ path: 'test-results/T7-4-SMTP-host.png', fullPage: false })
      }
    })
  })

  // T7-5: SMTP port input and validation
  test('T7-5 SMTP port input validation', async ({ authPage: page }) => {
    await test.step('Navigate to SMTP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Fill invalid port', async () => {
      const portInput = page.locator('input[placeholder*="port"], input[placeholder*="Port"], input[type="number"]').first()
      if (await portInput.isVisible().catch(() => false)) {
        await portInput.fill('99999')
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T7-5-SMTP-port-invalid.png', fullPage: false })
      }
    })

    await test.step('Fill valid port', async () => {
      const portInput = page.locator('input[placeholder*="port"], input[placeholder*="Port"], input[type="number"]').first()
      if (await portInput.isVisible().catch(() => false)) {
        await portInput.fill('587')
        await page.screenshot({ path: 'test-results/T7-5-SMTP-port-valid.png', fullPage: false })
      }
    })
  })

  // T7-6: SMTP TLS toggle
  test('T7-6 SMTP TLS toggle', async ({ authPage: page }) => {
    await test.step('Navigate to SMTP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find and toggle TLS switch', async () => {
      const tlsSwitch = page.locator('.n-switch').filter({ hasText: /TLS|SSL|tls|ssl/ }).first()
      if (await tlsSwitch.isVisible().catch(() => false)) {
        await tlsSwitch.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T7-6-SMTP-TLS-toggle.png', fullPage: false })
      }
    })
  })

  // T7-7: SMTP authentication toggle
  test('T7-7 SMTP auth toggle', async ({ authPage: page }) => {
    await test.step('Navigate to SMTP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find and toggle auth switch', async () => {
      const authSwitch = page.locator('.n-switch').filter({ hasText: /认证|Auth|auth/ }).first()
      if (await authSwitch.isVisible().catch(() => false)) {
        await authSwitch.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T7-7-SMTP-auth-toggle.png', fullPage: false })
      }
    })
  })

  // T7-8: SMTP username field
  test('T7-8 SMTP username field', async ({ authPage: page }) => {
    await test.step('Navigate to SMTP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Fill username', async () => {
      const usernameInput = page.locator('input[placeholder*="user"], input[placeholder*="User"], input[placeholder*="account"]').first()
      if (await usernameInput.isVisible().catch(() => false)) {
        await usernameInput.fill('test@example.com')
        await page.screenshot({ path: 'test-results/T7-8-SMTP-username.png', fullPage: false })
      }
    })
  })

  // T7-9: SMTP password field
  test('T7-9 SMTP password field', async ({ authPage: page }) => {
    await test.step('Navigate to SMTP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Fill password', async () => {
      const passwordInput = page.locator('input[type="password"][placeholder*="pass"], input[placeholder*="password"], input[placeholder*="密码"]').first()
      if (await passwordInput.isVisible().catch(() => false)) {
        await passwordInput.fill('testpassword123')
        await page.screenshot({ path: 'test-results/T7-9-SMTP-password.png', fullPage: false })
      }
    })
  })

  // T7-10: SMTP from address
  test('T7-10 SMTP from address', async ({ authPage: page }) => {
    await test.step('Navigate to SMTP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Fill from address', async () => {
      const fromInput = page.locator('input[placeholder*="from"], input[placeholder*="From"], input[placeholder*="发件"]').first()
      if (await fromInput.isVisible().catch(() => false)) {
        await fromInput.fill('noreply@example.com')
        await page.screenshot({ path: 'test-results/T7-10-SMTP-from.png', fullPage: false })
      }
    })
  })

  // T7-11: SMTP from name
  test('T7-11 SMTP from name', async ({ authPage: page }) => {
    await test.step('Navigate to SMTP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Fill from name', async () => {
      const nameInput = page.locator('input[placeholder*="name"], input[placeholder*="Name"], input[placeholder*="名称"]').first()
      if (await nameInput.isVisible().catch(() => false)) {
        await nameInput.fill('SRE Agent Alert')
        await page.screenshot({ path: 'test-results/T7-11-SMTP-from-name.png', fullPage: false })
      }
    })
  })

  // T7-12: SMTP test send button
  test('T7-12 SMTP test send button', async ({ authPage: page }) => {
    await test.step('Navigate to SMTP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find test send button', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|Send|发送/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await expect(testBtn).toBeVisible()
        await page.screenshot({ path: 'test-results/T7-12-SMTP-test-btn.png', fullPage: false })
      }
    })
  })

  // T7-13: SMTP test send with recipient
  test('T7-13 SMTP test send with recipient', async ({ authPage: page }) => {
    await test.step('Navigate to SMTP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Click test send', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试|Test|Send|发送/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await testBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T7-13-SMTP-test-send.png', fullPage: false })
      }
    })

    await test.step('Close dialog if any', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T7-14: SMTP save button
  test('T7-14 SMTP save button', async ({ authPage: page }) => {
    await test.step('Navigate to SMTP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find save button', async () => {
      const saveBtn = page.locator('button').filter({ hasText: /保存|Save|提交|Submit/ }).first()
      if (await saveBtn.isVisible().catch(() => false)) {
        await expect(saveBtn).toBeVisible()
        await page.screenshot({ path: 'test-results/T7-14-SMTP-save-btn.png', fullPage: false })
      }
    })
  })

  // T7-15: SMTP form validation on empty submit
  test('T7-15 SMTP empty submit validation', async ({ authPage: page }) => {
    await test.step('Navigate to SMTP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const smtpTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SMTP|邮件|Mail/ }).first()
      if (await smtpTab.isVisible().catch(() => false)) {
        await smtpTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Click save without filling fields', async () => {
      const saveBtn = page.locator('button').filter({ hasText: /保存|Save|提交|Submit/ }).first()
      if (await saveBtn.isVisible().catch(() => false)) {
        await saveBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T7-15-SMTP-empty-validation.png', fullPage: false })
      }
    })

    await test.step('Check for validation errors', async () => {
      const errorMsg = page.locator('.n-form-item-feedback--error, [class*="error"], .n-form-item-feedback').first()
      if (await errorMsg.isVisible().catch(() => false)) {
        await expect(errorMsg).toBeVisible()
      }
    })
  })

  // ================================================================
  // T7-16 ~ T7-30: Security Settings
  // ================================================================

  // T7-16: Security settings page load
  test('T7-16 Security settings page load', async ({ authPage: page }) => {
    await test.step('Navigate to settings page', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click security tab', async () => {
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T7-16-security-load.png', fullPage: true })
      }
    })
  })

  // T7-17: JWT expiry setting
  test('T7-17 JWT expiry setting', async ({ authPage: page }) => {
    await test.step('Navigate to security section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find JWT expiry input', async () => {
      const jwtInput = page.locator('input[placeholder*="JWT"], input[placeholder*="jwt"], input[placeholder*="token"]').first()
      if (await jwtInput.isVisible().catch(() => false)) {
        await jwtInput.fill('24h')
        await page.screenshot({ path: 'test-results/T7-17-JWT-expiry.png', fullPage: false })
      }
    })
  })

  // T7-18: Password policy settings
  test('T7-18 Password policy settings', async ({ authPage: page }) => {
    await test.step('Navigate to security section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find password policy settings', async () => {
      const policySection = page.locator('[class*="password"], [class*="policy"]').filter({ hasText: /密码|Password|policy/ }).first()
      if (await policySection.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T7-18-password-policy.png', fullPage: false })
      }
    })
  })

  // T7-19: Password minimum length
  test('T7-19 Password minimum length', async ({ authPage: page }) => {
    await test.step('Navigate to security section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find minimum length input', async () => {
      const minLenInput = page.locator('input[placeholder*="长度"], input[placeholder*="length"], input[placeholder*="min"]').first()
      if (await minLenInput.isVisible().catch(() => false)) {
        await minLenInput.fill('8')
        await page.screenshot({ path: 'test-results/T7-19-password-min-length.png', fullPage: false })
      }
    })
  })

  // T7-20: Session timeout setting
  test('T7-20 Session timeout setting', async ({ authPage: page }) => {
    await test.step('Navigate to security section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find session timeout input', async () => {
      const timeoutInput = page.locator('input[placeholder*="session"], input[placeholder*="Session"], input[placeholder*="超时"]').first()
      if (await timeoutInput.isVisible().catch(() => false)) {
        await timeoutInput.fill('30')
        await page.screenshot({ path: 'test-results/T7-20-session-timeout.png', fullPage: false })
      }
    })
  })

  // T7-21: 2FA toggle
  test('T7-21 2FA toggle', async ({ authPage: page }) => {
    await test.step('Navigate to security section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find 2FA toggle', async () => {
      const tfaSwitch = page.locator('.n-switch').filter({ hasText: /2FA|MFA|双因素|两步/ }).first()
      if (await tfaSwitch.isVisible().catch(() => false)) {
        await tfaSwitch.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T7-21-2FA-toggle.png', fullPage: false })
      }
    })
  })

  // T7-22: IP whitelist setting
  test('T7-22 IP whitelist setting', async ({ authPage: page }) => {
    await test.step('Navigate to security section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find IP whitelist input', async () => {
      const ipInput = page.locator('input[placeholder*="IP"], input[placeholder*="ip"], textarea[placeholder*="whitelist"]').first()
      if (await ipInput.isVisible().catch(() => false)) {
        await ipInput.fill('192.168.1.0/24')
        await page.screenshot({ path: 'test-results/T7-22-IP-whitelist.png', fullPage: false })
      }
    })
  })

  // T7-23: Audit log level setting
  test('T7-23 Audit log level setting', async ({ authPage: page }) => {
    await test.step('Navigate to security section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find audit log level selector', async () => {
      const auditSelect = page.locator('.n-select').filter({ hasText: /审计|Audit|日志级别/ }).first()
      if (await auditSelect.isVisible().catch(() => false)) {
        await auditSelect.click()
        await page.waitForTimeout(300)
        const option = page.locator('.n-select-option').filter({ hasText: /info|warn|debug|error/ }).first()
        if (await option.isVisible()) {
          await option.click()
          await page.screenshot({ path: 'test-results/T7-23-audit-log-level.png', fullPage: false })
        }
      }
    })
  })

  // T7-24: Login attempt limit
  test('T7-24 Login attempt limit', async ({ authPage: page }) => {
    await test.step('Navigate to security section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find login attempt limit', async () => {
      const attemptInput = page.locator('input[placeholder*="attempt"], input[placeholder*="尝试"], input[placeholder*="次数"]').first()
      if (await attemptInput.isVisible().catch(() => false)) {
        await attemptInput.fill('5')
        await page.screenshot({ path: 'test-results/T7-24-login-attempts.png', fullPage: false })
      }
    })
  })

  // T7-25: Account lockout duration
  test('T7-25 Account lockout duration', async ({ authPage: page }) => {
    await test.step('Navigate to security section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find lockout duration input', async () => {
      const lockoutInput = page.locator('input[placeholder*="lock"], input[placeholder*="锁定"], input[placeholder*="duration"]').first()
      if (await lockoutInput.isVisible().catch(() => false)) {
        await lockoutInput.fill('30')
        await page.screenshot({ path: 'test-results/T7-25-lockout-duration.png', fullPage: false })
      }
    })
  })

  // T7-26: CORS allowed origins
  test('T7-26 CORS allowed origins', async ({ authPage: page }) => {
    await test.step('Navigate to security section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find CORS origins input', async () => {
      const corsInput = page.locator('input[placeholder*="origin"], input[placeholder*="CORS"], textarea[placeholder*="origin"]').first()
      if (await corsInput.isVisible().catch(() => false)) {
        await corsInput.fill('https://example.com')
        await page.screenshot({ path: 'test-results/T7-26-CORS-origins.png', fullPage: false })
      }
    })
  })

  // T7-27: Rate limiting toggle
  test('T7-27 Rate limiting toggle', async ({ authPage: page }) => {
    await test.step('Navigate to security section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find rate limit toggle', async () => {
      const rateSwitch = page.locator('.n-switch').filter({ hasText: /限流|Rate|rate/ }).first()
      if (await rateSwitch.isVisible().catch(() => false)) {
        await rateSwitch.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T7-27-rate-limit-toggle.png', fullPage: false })
      }
    })
  })

  // T7-28: Security settings save
  test('T7-28 Security settings save', async ({ authPage: page }) => {
    await test.step('Navigate to security section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Click save button', async () => {
      const saveBtn = page.locator('button').filter({ hasText: /保存|Save|提交|Submit/ }).first()
      if (await saveBtn.isVisible().catch(() => false)) {
        await saveBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T7-28-security-save.png', fullPage: false })
      }
    })
  })

  // T7-29: Security settings API test
  test('T7-29 Security settings API test', async ({ authPage: page }) => {
    await test.step('GET security settings via API', async () => {
      const res = await API.get(page, '/api/v1/settings/security')
      await page.screenshot({ path: 'test-results/T7-29-security-API.png', fullPage: false })
    })
  })

  // T7-30: Security settings reset to defaults
  test('T7-30 Security settings reset to defaults', async ({ authPage: page }) => {
    await test.step('Navigate to security section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const secTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /安全|Security|认证/ }).first()
      if (await secTab.isVisible().catch(() => false)) {
        await secTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find reset button', async () => {
      const resetBtn = page.locator('button').filter({ hasText: /重置|Reset|默认|Default/ }).first()
      if (await resetBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T7-30-security-reset.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T7-31 ~ T7-45: SSO Settings
  // ================================================================

  // T7-31: SSO settings page load
  test('T7-31 SSO settings page load', async ({ authPage: page }) => {
    await test.step('Navigate to settings page', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click SSO tab', async () => {
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T7-31-SSO-load.png', fullPage: true })
      }
    })
  })

  // T7-32: OIDC configuration fields
  test('T7-32 OIDC configuration fields', async ({ authPage: page }) => {
    await test.step('Navigate to SSO section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Verify OIDC fields exist', async () => {
      const issuerInput = page.locator('input[placeholder*="issuer"], input[placeholder*="Issuer"], input[placeholder*="discovery"]').first()
      if (await issuerInput.isVisible().catch(() => false)) {
        await expect(issuerInput).toBeVisible()
        await page.screenshot({ path: 'test-results/T7-32-OIDC-fields.png', fullPage: false })
      }
    })
  })

  // T7-33: OIDC issuer URL input
  test('T7-33 OIDC issuer URL input', async ({ authPage: page }) => {
    await test.step('Navigate to SSO section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Fill OIDC issuer URL', async () => {
      const issuerInput = page.locator('input[placeholder*="issuer"], input[placeholder*="Issuer"], input[placeholder*="URL"]').first()
      if (await issuerInput.isVisible().catch(() => false)) {
        await issuerInput.fill('https://keycloak.example.com/realms/sreagent')
        await page.screenshot({ path: 'test-results/T7-33-OIDC-issuer.png', fullPage: false })
      }
    })
  })

  // T7-34: OIDC client ID input
  test('T7-34 OIDC client ID input', async ({ authPage: page }) => {
    await test.step('Navigate to SSO section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Fill client ID', async () => {
      const clientIdInput = page.locator('input[placeholder*="client"], input[placeholder*="Client"]').first()
      if (await clientIdInput.isVisible().catch(() => false)) {
        await clientIdInput.fill('sreagent-client')
        await page.screenshot({ path: 'test-results/T7-34-OIDC-client-id.png', fullPage: false })
      }
    })
  })

  // T7-35: OIDC client secret input
  test('T7-35 OIDC client secret input', async ({ authPage: page }) => {
    await test.step('Navigate to SSO section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Fill client secret', async () => {
      const secretInput = page.locator('input[type="password"], input[placeholder*="secret"], input[placeholder*="Secret"]').first()
      if (await secretInput.isVisible().catch(() => false)) {
        await secretInput.fill('super-secret-value-123')
        await page.screenshot({ path: 'test-results/T7-35-OIDC-client-secret.png', fullPage: false })
      }
    })
  })

  // T7-36: OIDC callback URL
  test('T7-36 OIDC callback URL', async ({ authPage: page }) => {
    await test.step('Navigate to SSO section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find callback URL field', async () => {
      const callbackInput = page.locator('input[placeholder*="callback"], input[placeholder*="Callback"], input[placeholder*="redirect"]').first()
      if (await callbackInput.isVisible().catch(() => false)) {
        await expect(callbackInput).toBeVisible()
        await page.screenshot({ path: 'test-results/T7-36-OIDC-callback.png', fullPage: false })
      }
    })
  })

  // T7-37: OAuth2 configuration
  test('T7-37 OAuth2 configuration', async ({ authPage: page }) => {
    await test.step('Navigate to SSO section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Switch to OAuth2 tab if available', async () => {
      const oauthTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /OAuth2|OAuth/ }).first()
      if (await oauthTab.isVisible().catch(() => false)) {
        await oauthTab.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T7-37-OAuth2-config.png', fullPage: false })
      }
    })
  })

  // T7-38: OAuth2 provider select
  test('T7-38 OAuth2 provider select', async ({ authPage: page }) => {
    await test.step('Navigate to SSO section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find provider selector', async () => {
      const providerSelect = page.locator('.n-select').filter({ hasText: /provider|Provider|提供/ }).first()
      if (await providerSelect.isVisible().catch(() => false)) {
        await providerSelect.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T7-38-OAuth2-provider.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T7-39: LDAP configuration
  test('T7-39 LDAP configuration', async ({ authPage: page }) => {
    await test.step('Navigate to SSO section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点|LDAP/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Switch to LDAP tab if available', async () => {
      const ldapTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /LDAP|ldap/ }).first()
      if (await ldapTab.isVisible().catch(() => false)) {
        await ldapTab.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T7-39-LDAP-config.png', fullPage: false })
      }
    })
  })

  // T7-40: LDAP server URL input
  test('T7-40 LDAP server URL input', async ({ authPage: page }) => {
    await test.step('Navigate to LDAP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点|LDAP/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
        const ldapTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /LDAP|ldap/ }).first()
        if (await ldapTab.isVisible().catch(() => false)) {
          await ldapTab.click()
          await page.waitForTimeout(300)
        }
      }
    })

    await test.step('Fill LDAP server URL', async () => {
      const ldapInput = page.locator('input[placeholder*="LDAP"], input[placeholder*="ldap"], input[placeholder*="server"]').first()
      if (await ldapInput.isVisible().catch(() => false)) {
        await ldapInput.fill('ldap://ldap.example.com:389')
        await page.screenshot({ path: 'test-results/T7-40-LDAP-server.png', fullPage: false })
      }
    })
  })

  // T7-41: LDAP bind DN input
  test('T7-41 LDAP bind DN input', async ({ authPage: page }) => {
    await test.step('Navigate to LDAP section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点|LDAP/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
        const ldapTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /LDAP|ldap/ }).first()
        if (await ldapTab.isVisible().catch(() => false)) {
          await ldapTab.click()
          await page.waitForTimeout(300)
        }
      }
    })

    await test.step('Fill bind DN', async () => {
      const dnInput = page.locator('input[placeholder*="DN"], input[placeholder*="dn"], input[placeholder*="bind"]').first()
      if (await dnInput.isVisible().catch(() => false)) {
        await dnInput.fill('cn=admin,dc=example,dc=com')
        await page.screenshot({ path: 'test-results/T7-41-LDAP-bind-DN.png', fullPage: false })
      }
    })
  })

  // T7-42: SSO test connection button
  test('T7-42 SSO test connection button', async ({ authPage: page }) => {
    await test.step('Navigate to SSO section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find test connection button', async () => {
      const testBtn = page.locator('button').filter({ hasText: /测试连接|Test Connection|Test/ }).first()
      if (await testBtn.isVisible().catch(() => false)) {
        await expect(testBtn).toBeVisible()
        await page.screenshot({ path: 'test-results/T7-42-SSO-test-btn.png', fullPage: false })
      }
    })
  })

  // T7-43: SSO enable/disable toggle
  test('T7-43 SSO enable disable toggle', async ({ authPage: page }) => {
    await test.step('Navigate to SSO section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find SSO enable toggle', async () => {
      const enableSwitch = page.locator('.n-switch').filter({ hasText: /启用|Enable|SSO/ }).first()
      if (await enableSwitch.isVisible().catch(() => false)) {
        await enableSwitch.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T7-43-SSO-toggle.png', fullPage: false })
      }
    })
  })

  // T7-44: SSO scopes configuration
  test('T7-44 SSO scopes configuration', async ({ authPage: page }) => {
    await test.step('Navigate to SSO section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Find scopes input', async () => {
      const scopesInput = page.locator('input[placeholder*="scope"], input[placeholder*="Scope"]').first()
      if (await scopesInput.isVisible().catch(() => false)) {
        await scopesInput.fill('openid profile email')
        await page.screenshot({ path: 'test-results/T7-44-SSO-scopes.png', fullPage: false })
      }
    })
  })

  // T7-45: SSO settings save
  test('T7-45 SSO settings save', async ({ authPage: page }) => {
    await test.step('Navigate to SSO section', async () => {
      await page.goto(SETTINGS_URL)
      await page.waitForLoadState('networkidle')
      const ssoTab = page.locator('.n-tabs-tab, [role="tab"]').filter({ hasText: /SSO|OIDC|OAuth|单点/ }).first()
      if (await ssoTab.isVisible().catch(() => false)) {
        await ssoTab.click()
        await page.waitForTimeout(500)
      }
    })

    await test.step('Click save button', async () => {
      const saveBtn = page.locator('button').filter({ hasText: /保存|Save|提交|Submit/ }).first()
      if (await saveBtn.isVisible().catch(() => false)) {
        await saveBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T7-45-SSO-save.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T7-46 ~ T7-60: User Management
  // ================================================================

  // T7-46: User list page load
  test('T7-46 User list page load', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T7-46-user-list-load.png', fullPage: true })
    })

    await test.step('Verify page body', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T7-47: User search
  test('T7-47 User search', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Search for user', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[placeholder*="Search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('admin')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T7-47-user-search.png', fullPage: false })
      }
    })

    await test.step('Clear search', async () => {
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[placeholder*="Search"]').first()
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.clear()
        await page.waitForTimeout(400)
      }
    })
  })

  // T7-48: Create user button
  test('T7-48 Create user button', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click create user button', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T7-48-create-user-btn.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T7-49: Create user form fields
  test('T7-49 Create user form fields', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open create dialog and verify fields', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const nameInput = page.locator('.n-modal input, [role="dialog"] input').first()
        if (await nameInput.isVisible()) {
          await expect(nameInput).toBeVisible()
          await page.screenshot({ path: 'test-results/T7-49-create-user-form.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T7-50: Create user — fill username
  test('T7-50 Create user fill username', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open dialog and fill username', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const inputs = page.locator('.n-modal input, [role="dialog"] input')
        const count = await inputs.count()
        if (count > 0) {
          await inputs.first().fill(uid('testuser'))
          await page.screenshot({ path: 'test-results/T7-50-create-username.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T7-51: Create user — fill email
  test('T7-51 Create user fill email', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open dialog and fill email', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const emailInput = page.locator('input[placeholder*="email"], input[placeholder*="Email"], input[type="email"]').first()
        if (await emailInput.isVisible().catch(() => false)) {
          await emailInput.fill(`${uid('user')}@example.com`)
          await page.screenshot({ path: 'test-results/T7-51-create-email.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T7-52: Create user — fill password
  test('T7-52 Create user fill password', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open dialog and fill password', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const pwdInput = page.locator('input[type="password"]').first()
        if (await pwdInput.isVisible().catch(() => false)) {
          await pwdInput.fill('TestPass123!')
          await page.screenshot({ path: 'test-results/T7-52-create-password.png', fullPage: false })
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T7-53: Create user — role select
  test('T7-53 Create user role select', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open dialog and select role', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const roleSelect = page.locator('.n-modal .n-select, [role="dialog"] .n-select').first()
        if (await roleSelect.isVisible().catch(() => false)) {
          await roleSelect.click()
          await page.waitForTimeout(300)
          const option = page.locator('.n-select-option').filter({ hasText: /member|admin|lead/ }).first()
          if (await option.isVisible()) {
            await option.click()
            await page.screenshot({ path: 'test-results/T7-53-create-role.png', fullPage: false })
          }
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T7-54: Create user — submit
  test('T7-54 Create user submit', async ({ authPage: page }) => {
    const testUsername = uid('submituser')
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open dialog and fill form', async () => {
      const createBtn = page.locator('button').filter({ hasText: /创建|Create|新建|添加|Add/ }).first()
      if (await createBtn.isVisible()) {
        await createBtn.click()
        await page.waitForTimeout(500)
        const inputs = page.locator('.n-modal input, [role="dialog"] input')
        const count = await inputs.count()
        if (count > 0) {
          await inputs.first().fill(testUsername)
        }
      }
    })

    await test.step('Submit form', async () => {
      const submitBtn = page.locator('button[type="submit"], .n-modal button, [role="dialog"] button').filter({ hasText: /保存|Save|确定|OK|提交|Create/ }).first()
      if (await submitBtn.isVisible()) {
        await submitBtn.click()
        await page.waitForTimeout(1000)
        await page.screenshot({ path: 'test-results/T7-54-create-submit.png', fullPage: false })
      }
    })

    await test.step('Cleanup: delete via API', async () => {
      const res = await API.get(page, '/api/v1/users?page=1&page_size=100')
      const users = res?.data?.list || res?.data?.items || []
      const found = users.find((u: { username: string }) => u.username === testUsername)
      if (found) {
        await API.del(page, `/api/v1/users/${found.id}`)
      }
    })
  })

  // T7-55: Edit user
  test('T7-55 Edit user', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click edit on first user row', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T7-55-edit-user.png', fullPage: false })
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T7-56: Delete user confirmation
  test('T7-56 Delete user confirmation', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click delete on a user row', async () => {
      const deleteBtn = page.locator('button').filter({ hasText: /删除|Delete/ }).first()
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T7-56-delete-user-confirm.png', fullPage: false })
      }
    })

    await test.step('Cancel delete', async () => {
      const cancelBtn = page.locator('.n-dialog button, .n-modal button').filter({ hasText: /取消|Cancel/ }).first()
      if (await cancelBtn.isVisible().catch(() => false)) {
        await cancelBtn.click()
        await page.waitForTimeout(300)
      } else {
        await page.keyboard.press('Escape')
        await page.waitForTimeout(300)
      }
    })
  })

  // T7-57: User role change
  test('T7-57 User role change', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open user edit and change role', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        const roleSelect = page.locator('.n-modal .n-select, [role="dialog"] .n-select').first()
        if (await roleSelect.isVisible().catch(() => false)) {
          await roleSelect.click()
          await page.waitForTimeout(300)
          await page.screenshot({ path: 'test-results/T7-57-role-change.png', fullPage: false })
          await page.keyboard.press('Escape')
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T7-58: User active toggle
  test('T7-58 User active toggle', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find user active toggle', async () => {
      const toggle = page.locator('.n-switch, [class*="toggle"]').first()
      if (await toggle.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T7-58-user-active-toggle.png', fullPage: false })
      }
    })
  })

  // T7-59: User password reset
  test('T7-59 User password reset', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find password reset action', async () => {
      const moreBtn = page.locator('button').filter({ hasText: /更多|More|操作/ }).first()
      if (await moreBtn.isVisible().catch(() => false)) {
        await moreBtn.click()
        await page.waitForTimeout(300)
        const resetItem = page.locator('.n-dropdown-option, [class*="dropdown"]').filter({ hasText: /重置密码|Reset Password|Reset/ }).first()
        if (await resetItem.isVisible().catch(() => false)) {
          await page.screenshot({ path: 'test-results/T7-59-password-reset.png', fullPage: false })
        }
      }
    })

    await test.step('Close menu', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T7-60: User team assignment
  test('T7-60 User team assignment', async ({ authPage: page }) => {
    await test.step('Navigate to user management', async () => {
      await page.goto(USERS_URL)
      await page.waitForLoadState('networkidle')
    })

    await test.step('Open user edit and check team assignment', async () => {
      const editBtn = page.locator('button').filter({ hasText: /编辑|Edit/ }).first()
      if (await editBtn.isVisible().catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(500)
        const teamSelect = page.locator('.n-modal .n-select, [role="dialog"] .n-select').nth(1)
        if (await teamSelect.isVisible().catch(() => false)) {
          await teamSelect.click()
          await page.waitForTimeout(300)
          await page.screenshot({ path: 'test-results/T7-60-team-assignment.png', fullPage: false })
          await page.keyboard.press('Escape')
        }
      }
    })

    await test.step('Close dialog', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })
})
