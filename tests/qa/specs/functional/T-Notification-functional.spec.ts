import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

test.describe('通知功能测试', () => {

  // NOTIF-1: List notify media -> verify structure -> create webhook media -> verify saved -> delete -> verify gone
  test('NOTIF-1 通知媒体 CRUD', async ({ authPage: page }) => {
    let createdMediaId: number | undefined
    const mediaName = `QA-Test-Webhook-${Date.now()}`

    await test.step('获取通知媒体列表', async () => {
      const res = await API.get(page, '/api/v1/notify-media?page=1&page_size=100')
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      expect(res.data.list).toBeDefined()
      expect(Array.isArray(res.data.list)).toBe(true)
      await page.screenshot({ path: 'test-results/NOTIF-1-媒体列表.png', fullPage: true })
    })

    try {
      await test.step('创建 webhook 类型通知媒体', async () => {
        const res = await API.post(page, '/api/v1/notify-media', {
          name: mediaName,
          type: 'webhook',
          config: JSON.stringify({
            url: 'https://httpbin.org/post',
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
          }),
        })
        expect(res.code).toBe(0)
        expect(res.data).toBeDefined()
        expect(res.data.id).toBeGreaterThan(0)
        expect(res.data.name).toBe(mediaName)
        createdMediaId = res.data.id
        await page.screenshot({ path: 'test-results/NOTIF-1-创建成功.png', fullPage: false })
      })

      await test.step('验证媒体已保存', async () => {
        const res = await API.get(page, `/api/v1/notify-media/${createdMediaId}`)
        expect(res.code).toBe(0)
        expect(res.data).toBeDefined()
        expect(res.data.id).toBe(createdMediaId)
        expect(res.data.name).toBe(mediaName)
        expect(res.data.type).toBe('webhook')
        await page.screenshot({ path: 'test-results/NOTIF-1-验证已保存.png', fullPage: false })
      })
    } finally {
      await test.step('清理: 删除创建的媒体', async () => {
        if (createdMediaId) {
          const res = await API.del(page, `/api/v1/notify-media/${createdMediaId}`)
          expect(res.code).toBe(0)

          // 验证已删除
          const verifyRes = await API.get(page, `/api/v1/notify-media/${createdMediaId}`)
          expect(verifyRes.code).not.toBe(0)
          await page.screenshot({ path: 'test-results/NOTIF-1-清理完成.png', fullPage: false })
        }
      })
    }
  })

  // NOTIF-2: List message templates -> verify structure -> create template -> verify saved
  test('NOTIF-2 消息模板 CRUD', async ({ authPage: page }) => {
    let createdTemplateId: number | undefined
    const templateName = `QA-Test-Template-${Date.now()}`

    await test.step('获取消息模板列表', async () => {
      const res = await API.get(page, '/api/v1/message-templates?page=1&page_size=100')
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      expect(res.data.list).toBeDefined()
      expect(Array.isArray(res.data.list)).toBe(true)
      await page.screenshot({ path: 'test-results/NOTIF-2-模板列表.png', fullPage: true })
    })

    try {
      await test.step('创建消息模板', async () => {
        const res = await API.post(page, '/api/v1/message-templates', {
          name: templateName,
          type: 'webhook',
          content: '{"alert": "{{ .AlertName }}", "severity": "{{ .Severity }}", "status": "{{ .Status }}"}',
          description: 'QA 自动化测试模板',
        })
        expect(res.code).toBe(0)
        expect(res.data).toBeDefined()
        expect(res.data.id).toBeGreaterThan(0)
        expect(res.data.name).toBe(templateName)
        createdTemplateId = res.data.id
        await page.screenshot({ path: 'test-results/NOTIF-2-创建成功.png', fullPage: false })
      })

      await test.step('验证模板已保存', async () => {
        const res = await API.get(page, `/api/v1/message-templates/${createdTemplateId}`)
        expect(res.code).toBe(0)
        expect(res.data).toBeDefined()
        expect(res.data.id).toBe(createdTemplateId)
        expect(res.data.name).toBe(templateName)
        await page.screenshot({ path: 'test-results/NOTIF-2-验证已保存.png', fullPage: false })
      })
    } finally {
      await test.step('清理: 删除创建的模板', async () => {
        if (createdTemplateId) {
          const res = await API.del(page, `/api/v1/message-templates/${createdTemplateId}`)
          expect(res.code).toBe(0)
          await page.screenshot({ path: 'test-results/NOTIF-2-清理完成.png', fullPage: false })
        }
      })
    }
  })

  // NOTIF-3: List notify rules -> verify structure -> create rule with conditions -> verify saved
  test('NOTIF-3 通知规则 CRUD', async ({ authPage: page }) => {
    let createdRuleId: number | undefined
    const ruleName = `QA-Test-Rule-${Date.now()}`

    await test.step('获取通知规则列表', async () => {
      const res = await API.get(page, '/api/v1/notify-rules?page=1&page_size=100')
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      expect(res.data.list).toBeDefined()
      expect(Array.isArray(res.data.list)).toBe(true)
      await page.screenshot({ path: 'test-results/NOTIF-3-规则列表.png', fullPage: true })
    })

    try {
      await test.step('创建通知规则', async () => {
        const res = await API.post(page, '/api/v1/notify-rules', {
          name: ruleName,
          severity: ['critical', 'warning'],
          labels_match: { env: 'test' },
          notify_media_ids: [],
          is_enabled: true,
        })
        expect(res.code).toBe(0)
        expect(res.data).toBeDefined()
        expect(res.data.id).toBeGreaterThan(0)
        expect(res.data.name).toBe(ruleName)
        createdRuleId = res.data.id
        await page.screenshot({ path: 'test-results/NOTIF-3-创建成功.png', fullPage: false })
      })

      await test.step('验证规则已保存', async () => {
        const res = await API.get(page, `/api/v1/notify-rules/${createdRuleId}`)
        expect(res.code).toBe(0)
        expect(res.data).toBeDefined()
        expect(res.data.id).toBe(createdRuleId)
        expect(res.data.name).toBe(ruleName)
        await page.screenshot({ path: 'test-results/NOTIF-3-验证已保存.png', fullPage: false })
      })
    } finally {
      await test.step('清理: 删除创建的规则', async () => {
        if (createdRuleId) {
          const res = await API.del(page, `/api/v1/notify-rules/${createdRuleId}`)
          expect(res.code).toBe(0)
          await page.screenshot({ path: 'test-results/NOTIF-3-清理完成.png', fullPage: false })
        }
      })
    }
  })
})
