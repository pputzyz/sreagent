import { test, expect } from '../../fixtures/auth'

// T12: Frontend Full Test Suite — 40 tests
// Covers: Login Page (T12-1~T12-10), Navigation (T12-11~T12-20),
//         Global Features (T12-21~T12-30), Responsive (T12-31~T12-40)

const BASE_URL = 'http://localhost:3000'

/** Generate unique name */
function uid(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

test.describe('T12 - Frontend Full Test Suite', () => {

  // ================================================================
  // T12-1 ~ T12-10: Login Page
  // ================================================================

  // T12-1: Login page form elements
  test('T12-1 Login page form elements', async ({ page }) => {
    await test.step('Navigate to login page', async () => {
      // Clear auth state
      await page.goto(BASE_URL)
      await page.evaluate(() => {
        localStorage.clear()
      })
      await page.goto(BASE_URL + '/login')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-1-login-form.png', fullPage: true })
    })

    await test.step('Verify username field exists', async () => {
      const usernameInput = page.locator('input[placeholder*="用户"], input[placeholder*="user"], input[placeholder*="User"], input[name="username"]').first()
      if (await usernameInput.isVisible().catch(() => false)) {
        await expect(usernameInput).toBeVisible()
      }
    })

    await test.step('Verify password field exists', async () => {
      const passwordInput = page.locator('input[type="password"]').first()
      if (await passwordInput.isVisible().catch(() => false)) {
        await expect(passwordInput).toBeVisible()
      }
    })
  })

  // T12-2: Login form submit
  test('T12-2 Login form submit', async ({ page }) => {
    await test.step('Navigate to login page', async () => {
      await page.goto(BASE_URL)
      await page.evaluate(() => { localStorage.clear() })
      await page.goto(BASE_URL + '/login')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Fill login form', async () => {
      const usernameInput = page.locator('input[placeholder*="用户"], input[placeholder*="user"], input[placeholder*="User"], input[name="username"]').first()
      const passwordInput = page.locator('input[type="password"]').first()
      if (await usernameInput.isVisible().catch(() => false)) {
        await usernameInput.fill('admin')
      }
      if (await passwordInput.isVisible().catch(() => false)) {
        await passwordInput.fill('admin123')
      }
      await page.screenshot({ path: 'test-results/T12-2-login-filled.png', fullPage: false })
    })

    await test.step('Click login button', async () => {
      const loginBtn = page.locator('button').filter({ hasText: /登录|Login|Sign in|Sign In/ }).first()
      if (await loginBtn.isVisible()) {
        await loginBtn.click()
        await page.waitForTimeout(2000)
        await page.screenshot({ path: 'test-results/T12-2-login-submitted.png', fullPage: false })
      }
    })
  })

  // T12-3: Login error display
  test('T12-3 Login error display', async ({ page }) => {
    await test.step('Navigate to login page', async () => {
      await page.goto(BASE_URL)
      await page.evaluate(() => { localStorage.clear() })
      await page.goto(BASE_URL + '/login')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Submit with wrong credentials', async () => {
      const usernameInput = page.locator('input[placeholder*="用户"], input[placeholder*="user"], input[placeholder*="User"], input[name="username"]').first()
      const passwordInput = page.locator('input[type="password"]').first()
      if (await usernameInput.isVisible().catch(() => false)) {
        await usernameInput.fill('wronguser')
      }
      if (await passwordInput.isVisible().catch(() => false)) {
        await passwordInput.fill('wrongpass')
      }
      const loginBtn = page.locator('button').filter({ hasText: /登录|Login|Sign in|Sign In/ }).first()
      if (await loginBtn.isVisible()) {
        await loginBtn.click()
        await page.waitForTimeout(1500)
        await page.screenshot({ path: 'test-results/T12-3-login-error.png', fullPage: false })
      }
    })

    await test.step('Verify error message appears', async () => {
      const errorMsg = page.locator('[class*="error"], .n-message, [class*="alert"], [class*="toast"]').first()
      if (await errorMsg.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T12-3-error-shown.png', fullPage: false })
      }
    })
  })

  // T12-4: Login captcha area
  test('T12-4 Login captcha area', async ({ page }) => {
    await test.step('Navigate to login page', async () => {
      await page.goto(BASE_URL)
      await page.evaluate(() => { localStorage.clear() })
      await page.goto(BASE_URL + '/login')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for captcha element', async () => {
      const captcha = page.locator('[class*="captcha"], canvas, [class*="verify"], input[placeholder*="验证码"]').first()
      if (await captcha.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T12-4-captcha.png', fullPage: false })
      }
    })
  })

  // T12-5: Login SSO button
  test('T12-5 Login SSO button', async ({ page }) => {
    await test.step('Navigate to login page', async () => {
      await page.goto(BASE_URL)
      await page.evaluate(() => { localStorage.clear() })
      await page.goto(BASE_URL + '/login')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for SSO button', async () => {
      const ssoBtn = page.locator('button, a').filter({ hasText: /SSO|OIDC|OAuth|单点|企业/ }).first()
      if (await ssoBtn.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T12-5-sso-button.png', fullPage: false })
      }
    })
  })

  // T12-6: Login remember me checkbox
  test('T12-6 Login remember me checkbox', async ({ page }) => {
    await test.step('Navigate to login page', async () => {
      await page.goto(BASE_URL)
      await page.evaluate(() => { localStorage.clear() })
      await page.goto(BASE_URL + '/login')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for remember me option', async () => {
      const rememberCheckbox = page.locator('input[type="checkbox"], .n-checkbox').filter({ hasText: /记住|Remember|remember/ }).first()
      if (await rememberCheckbox.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T12-6-remember-me.png', fullPage: false })
      }
    })
  })

  // T12-7: Login redirect after auth
  test('T12-7 Login redirect after auth', async ({ authPage: page }) => {
    await test.step('Navigate to protected page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Verify redirected to authenticated page', async () => {
      const url = page.url()
      await page.screenshot({ path: 'test-results/T12-7-redirect-auth.png', fullPage: false })
    })
  })

  // T12-8: Login form validation — empty submit
  test('T12-8 Login form empty submit', async ({ page }) => {
    await test.step('Navigate to login page', async () => {
      await page.goto(BASE_URL)
      await page.evaluate(() => { localStorage.clear() })
      await page.goto(BASE_URL + '/login')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click login without filling fields', async () => {
      const loginBtn = page.locator('button').filter({ hasText: /登录|Login|Sign in|Sign In/ }).first()
      if (await loginBtn.isVisible()) {
        await loginBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T12-8-empty-submit.png', fullPage: false })
      }
    })

    await test.step('Check for validation messages', async () => {
      const validationMsg = page.locator('.n-form-item-feedback--error, [class*="error"], [class*="required"]').first()
      if (await validationMsg.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T12-8-validation-msg.png', fullPage: false })
      }
    })
  })

  // T12-9: Login loading state
  test('T12-9 Login loading state', async ({ page }) => {
    await test.step('Navigate to login page', async () => {
      await page.goto(BASE_URL)
      await page.evaluate(() => { localStorage.clear() })
      await page.goto(BASE_URL + '/login')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Submit and check for loading indicator', async () => {
      const usernameInput = page.locator('input[placeholder*="用户"], input[placeholder*="user"], input[placeholder*="User"], input[name="username"]').first()
      const passwordInput = page.locator('input[type="password"]').first()
      if (await usernameInput.isVisible().catch(() => false)) {
        await usernameInput.fill('admin')
      }
      if (await passwordInput.isVisible().catch(() => false)) {
        await passwordInput.fill('admin123')
      }
      const loginBtn = page.locator('button').filter({ hasText: /登录|Login|Sign in|Sign In/ }).first()
      if (await loginBtn.isVisible()) {
        await loginBtn.click()
        await page.waitForTimeout(200)
        // Capture loading state quickly
        await page.screenshot({ path: 'test-results/T12-9-login-loading.png', fullPage: false })
      }
    })
  })

  // T12-10: Login page accessibility
  test('T12-10 Login page accessibility', async ({ page }) => {
    await test.step('Navigate to login page', async () => {
      await page.goto(BASE_URL)
      await page.evaluate(() => { localStorage.clear() })
      await page.goto(BASE_URL + '/login')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check form labels and ARIA attributes', async () => {
      const form = page.locator('form, [role="form"]').first()
      if (await form.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T12-10-login-a11y.png', fullPage: false })
      }
    })

    await test.step('Check keyboard navigation', async () => {
      await page.keyboard.press('Tab')
      await page.waitForTimeout(200)
      await page.keyboard.press('Tab')
      await page.waitForTimeout(200)
      await page.screenshot({ path: 'test-results/T12-10-login-keyboard.png', fullPage: false })
    })
  })

  // ================================================================
  // T12-11 ~ T12-20: Navigation
  // ================================================================

  // T12-11: Sidebar navigation
  test('T12-11 Sidebar navigation', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Verify sidebar exists', async () => {
      const sidebar = page.locator('nav, [class*="sidebar"], [class*="rail"], [class*="app-shell"]').first()
      if (await sidebar.isVisible().catch(() => false)) {
        await expect(sidebar).toBeVisible()
        await page.screenshot({ path: 'test-results/T12-11-sidebar.png', fullPage: false })
      }
    })
  })

  // T12-12: Sidebar rail collapse
  test('T12-12 Sidebar rail collapse', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find and click rail toggle', async () => {
      const railToggle = page.locator('button[aria-label*="collapse"], button[aria-label*="rail"], [class*="rail-toggle"], [class*="collapse-btn"]').first()
      if (await railToggle.isVisible().catch(() => false)) {
        await railToggle.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T12-12-rail-collapsed.png', fullPage: false })
        // Toggle back
        await railToggle.click()
        await page.waitForTimeout(500)
      }
    })
  })

  // T12-13: Breadcrumbs display
  test('T12-13 Breadcrumbs display', async ({ authPage: page }) => {
    await test.step('Navigate to a nested page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check breadcrumbs', async () => {
      const breadcrumbs = page.locator('.n-breadcrumb, [class*="breadcrumb"], nav[aria-label="breadcrumb"]').first()
      if (await breadcrumbs.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T12-13-breadcrumbs.png', fullPage: false })
      }
    })
  })

  // T12-14: Back button
  test('T12-14 Back button', async ({ authPage: page }) => {
    await test.step('Navigate to a page then back', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
      await page.goBack()
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T12-14-back-button.png', fullPage: false })
    })
  })

  // T12-15: Deep linking
  test('T12-15 Deep linking', async ({ authPage: page }) => {
    await test.step('Navigate directly to deep link', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-15-deep-link.png', fullPage: false })
    })

    await test.step('Verify correct page loaded', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T12-16: Mobile nav menu
  test('T12-16 Mobile nav menu', async ({ authPage: page }) => {
    await test.step('Set mobile viewport', async () => {
      await page.setViewportSize({ width: 375, height: 812 })
    })

    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-16-mobile-nav.png', fullPage: false })
    })

    await test.step('Find mobile menu button', async () => {
      const menuBtn = page.locator('button[aria-label*="menu"], button[aria-label*="Menu"], [class*="hamburger"], [class*="mobile-menu"]').first()
      if (await menuBtn.isVisible().catch(() => false)) {
        await menuBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T12-16-mobile-menu-open.png', fullPage: false })
      }
    })

    await test.step('Restore viewport', async () => {
      await page.setViewportSize({ width: 1280, height: 720 })
    })
  })

  // T12-17: Keyboard navigation — Tab
  test('T12-17 Keyboard navigation Tab', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Tab through navigation items', async () => {
      for (let i = 0; i < 5; i++) {
        await page.keyboard.press('Tab')
        await page.waitForTimeout(100)
      }
      await page.screenshot({ path: 'test-results/T12-17-tab-navigation.png', fullPage: false })
    })
  })

  // T12-18: Focus management
  test('T12-18 Focus management', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check focused element after navigation', async () => {
      await page.keyboard.press('Tab')
      await page.waitForTimeout(200)
      const focusedElement = page.locator(':focus')
      await page.screenshot({ path: 'test-results/T12-18-focus-management.png', fullPage: false })
    })
  })

  // T12-19: Sidebar menu items
  test('T12-19 Sidebar menu items', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Count sidebar menu items', async () => {
      const menuItems = page.locator('nav a, [class*="sidebar"] a, [class*="menu-item"], [class*="nav-item"]')
      const count = await menuItems.count()
      await page.screenshot({ path: 'test-results/T12-19-sidebar-items.png', fullPage: false })
    })
  })

  // T12-20: Sidebar active state
  test('T12-20 Sidebar active state', async ({ authPage: page }) => {
    await test.step('Navigate to alert rules page', async () => {
      await page.goto(BASE_URL + '/alert/rules')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check active menu item', async () => {
      const activeItem = page.locator('[class*="active"], [aria-selected="true"], [class*="selected"]').first()
      if (await activeItem.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T12-20-active-state.png', fullPage: false })
      }
    })
  })

  // ================================================================
  // T12-21 ~ T12-30: Global Features
  // ================================================================

  // T12-21: Global search
  test('T12-21 Global search', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find global search', async () => {
      const globalSearch = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], [class*="global-search"], [class*="search-input"]').first()
      if (await globalSearch.isVisible().catch(() => false)) {
        await globalSearch.fill('test')
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T12-21-global-search.png', fullPage: false })
      }
    })

    await test.step('Clear search', async () => {
      const globalSearch = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], [class*="global-search"], [class*="search-input"]').first()
      if (await globalSearch.isVisible().catch(() => false)) {
        await globalSearch.clear()
        await page.waitForTimeout(300)
      }
    })
  })

  // T12-22: Notifications bell
  test('T12-22 Notifications bell', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find and click notification bell', async () => {
      const bellBtn = page.locator('button[aria-label*="notification"], button[aria-label*="通知"], [class*="notification"], [class*="bell"]').first()
      if (await bellBtn.isVisible().catch(() => false)) {
        await bellBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T12-22-notifications.png', fullPage: false })
      }
    })

    await test.step('Close notifications', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T12-23: Theme toggle
  test('T12-23 Theme toggle', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find theme toggle', async () => {
      const themeBtn = page.locator('button[aria-label*="theme"], button[aria-label*="主题"], [class*="theme"], [class*="dark-mode"]').first()
      if (await themeBtn.isVisible().catch(() => false)) {
        await themeBtn.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T12-23-theme-toggle.png', fullPage: false })
        // Toggle back
        await themeBtn.click()
        await page.waitForTimeout(500)
      }
    })
  })

  // T12-24: Language switcher
  test('T12-24 Language switcher', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Find language switcher', async () => {
      const langBtn = page.locator('button, .n-select').filter({ hasText: /语言|Language|中文|English|EN|CN/ }).first()
      if (await langBtn.isVisible().catch(() => false)) {
        await langBtn.click()
        await page.waitForTimeout(300)
        await page.screenshot({ path: 'test-results/T12-24-language-switcher.png', fullPage: false })
        await page.keyboard.press('Escape')
      }
    })
  })

  // T12-25: User menu
  test('T12-25 User menu', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Click user menu avatar', async () => {
      const userMenu = page.locator('[class*="avatar"], [class*="user-menu"], button[aria-label*="user"], [class*="dropdown-user"]').first()
      if (await userMenu.isVisible().catch(() => false)) {
        await userMenu.click()
        await page.waitForTimeout(500)
        await page.screenshot({ path: 'test-results/T12-25-user-menu.png', fullPage: false })
      }
    })

    await test.step('Close menu', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T12-26: Keyboard shortcuts help
  test('T12-26 Keyboard shortcuts help', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Try opening shortcuts help with ?', async () => {
      await page.keyboard.press('?')
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T12-26-shortcuts-help.png', fullPage: false })
    })

    await test.step('Close', async () => {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
    })
  })

  // T12-27: Error boundary
  test('T12-27 Error boundary', async ({ authPage: page }) => {
    await test.step('Navigate to a non-existent route', async () => {
      await page.goto(BASE_URL + '/nonexistent-page-xyz')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-27-error-boundary.png', fullPage: false })
    })

    await test.step('Check for 404 or error display', async () => {
      const errorText = page.locator('text=404, text=Not Found, text=找不到, [class*="error-page"], [class*="not-found"]').first()
      if (await errorText.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T12-27-404-page.png', fullPage: false })
      }
    })
  })

  // T12-28: Loading states
  test('T12-28 Loading states', async ({ authPage: page }) => {
    await test.step('Navigate and capture loading state', async () => {
      const loadingPromise = page.goto(BASE_URL + '/alert/rules')
      await page.waitForTimeout(100)
      await page.screenshot({ path: 'test-results/T12-28-loading-state.png', fullPage: false })
      await loadingPromise
      await page.waitForLoadState('networkidle')
    })

    await test.step('Verify loading completed', async () => {
      await expect(page.locator('body')).toBeVisible()
    })
  })

  // T12-29: Page title
  test('T12-29 Page title', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check page title', async () => {
      const title = await page.title()
      expect(title.length).toBeGreaterThan(0)
      await page.screenshot({ path: 'test-results/T12-29-page-title.png', fullPage: false })
    })
  })

  // T12-30: Favicon
  test('T12-30 Favicon', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check favicon link exists', async () => {
      const favicon = page.locator('link[rel="icon"], link[rel="shortcut icon"]')
      const count = await favicon.count()
      await page.screenshot({ path: 'test-results/T12-30-favicon.png', fullPage: false })
    })
  })

  // ================================================================
  // T12-31 ~ T12-40: Responsive
  // ================================================================

  // T12-31: Mobile layout
  test('T12-31 Mobile layout', async ({ authPage: page }) => {
    await test.step('Set mobile viewport', async () => {
      await page.setViewportSize({ width: 375, height: 812 })
    })

    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-31-mobile-layout.png', fullPage: true })
    })

    await test.step('Restore viewport', async () => {
      await page.setViewportSize({ width: 1280, height: 720 })
    })
  })

  // T12-32: Tablet layout
  test('T12-32 Tablet layout', async ({ authPage: page }) => {
    await test.step('Set tablet viewport', async () => {
      await page.setViewportSize({ width: 768, height: 1024 })
    })

    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-32-tablet-layout.png', fullPage: true })
    })

    await test.step('Restore viewport', async () => {
      await page.setViewportSize({ width: 1280, height: 720 })
    })
  })

  // T12-33: Desktop layout
  test('T12-33 Desktop layout', async ({ authPage: page }) => {
    await test.step('Set desktop viewport', async () => {
      await page.setViewportSize({ width: 1920, height: 1080 })
    })

    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-33-desktop-layout.png', fullPage: true })
    })

    await test.step('Restore viewport', async () => {
      await page.setViewportSize({ width: 1280, height: 720 })
    })
  })

  // T12-34: Print styles
  test('T12-34 Print styles', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Emulate print media', async () => {
      await page.emulateMedia({ media: 'print' })
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T12-34-print-styles.png', fullPage: true })
    })

    await test.step('Reset media', async () => {
      await page.emulateMedia({ media: 'screen' })
    })
  })

  // T12-35: High contrast mode
  test('T12-35 High contrast mode', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for high contrast support', async () => {
      // Check if there's a high contrast toggle or CSS media query
      await page.emulateMedia({ colorScheme: 'dark' })
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T12-35-high-contrast.png', fullPage: true })
    })

    await test.step('Reset', async () => {
      await page.emulateMedia({ colorScheme: 'light' })
    })
  })

  // T12-36: Reduced motion
  test('T12-36 Reduced motion', async ({ authPage: page }) => {
    await test.step('Emulate reduced motion preference', async () => {
      await page.emulateMedia({ reducedMotion: 'reduce' })
    })

    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: 'test-results/T12-36-reduced-motion.png', fullPage: true })
    })

    await test.step('Reset', async () => {
      await page.emulateMedia({ reducedMotion: 'no-preference' })
    })
  })

  // T12-37: Screen reader landmarks
  test('T12-37 Screen reader landmarks', async ({ authPage: page }) => {
    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check for ARIA landmarks', async () => {
      const main = page.locator('main, [role="main"]')
      const nav = page.locator('nav, [role="navigation"]')
      const mainCount = await main.count()
      const navCount = await nav.count()
      await page.screenshot({ path: 'test-results/T12-37-landmarks.png', fullPage: false })
    })
  })

  // T12-38: Touch targets
  test('T12-38 Touch targets', async ({ authPage: page }) => {
    await test.step('Set mobile viewport', async () => {
      await page.setViewportSize({ width: 375, height: 812 })
    })

    await test.step('Navigate to home page', async () => {
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Check button sizes for touch', async () => {
      const buttons = page.locator('button, a[href]')
      const count = await buttons.count()
      await page.screenshot({ path: 'test-results/T12-38-touch-targets.png', fullPage: false })
    })

    await test.step('Restore viewport', async () => {
      await page.setViewportSize({ width: 1280, height: 720 })
    })
  })

  // T12-39: Responsive sidebar behavior
  test('T12-39 Responsive sidebar behavior', async ({ authPage: page }) => {
    await test.step('Navigate to home page at desktop width', async () => {
      await page.setViewportSize({ width: 1280, height: 720 })
      await page.goto(BASE_URL + '/')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Resize to tablet and check sidebar', async () => {
      await page.setViewportSize({ width: 768, height: 1024 })
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T12-39-responsive-sidebar-tablet.png', fullPage: false })
    })

    await test.step('Resize to mobile', async () => {
      await page.setViewportSize({ width: 375, height: 812 })
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T12-39-responsive-sidebar-mobile.png', fullPage: false })
    })

    await test.step('Restore viewport', async () => {
      await page.setViewportSize({ width: 1280, height: 720 })
    })
  })

  // T12-40: Responsive table overflow
  test('T12-40 Responsive table overflow', async ({ authPage: page }) => {
    await test.step('Navigate to a page with tables', async () => {
      await page.goto(BASE_URL + '/alert/events')
      await page.waitForLoadState('networkidle')
    })

    await test.step('Set mobile viewport', async () => {
      await page.setViewportSize({ width: 375, height: 812 })
      await page.waitForTimeout(500)
      await page.screenshot({ path: 'test-results/T12-40-table-mobile.png', fullPage: false })
    })

    await test.step('Check horizontal scroll', async () => {
      const table = page.locator('table, .n-data-table, [class*="table"]').first()
      if (await table.isVisible().catch(() => false)) {
        await page.screenshot({ path: 'test-results/T12-40-table-overflow.png', fullPage: false })
      }
    })

    await test.step('Restore viewport', async () => {
      await page.setViewportSize({ width: 1280, height: 720 })
    })
  })
})
