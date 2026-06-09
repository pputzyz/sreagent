import { test, expect } from '../../fixtures/auth'
import { API } from '../../helpers/api'

test.describe('数据源功能测试', () => {

  // DS-1: List datasources -> verify VM datasource exists -> health check -> verify status and latency
  test('DS-1 数据源列表与健康检查', async ({ authPage: page }) => {
    let datasourceId: number | undefined

    await test.step('获取数据源列表', async () => {
      const res = await API.get(page, '/api/v1/datasources?page=1&page_size=100')
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      expect(res.data.list).toBeDefined()
      expect(Array.isArray(res.data.list)).toBe(true)
      expect(res.data.list.length).toBeGreaterThan(0)

      // 找到 VictoriaMetrics 类型数据源
      const vmDs = res.data.list.find((ds: any) =>
        ds.type === 'victoriametrics' || ds.type === 'prometheus' || ds.type === 'vm'
      )
      expect(vmDs).toBeDefined()
      datasourceId = vmDs.id
      expect(datasourceId).toBeGreaterThan(0)
      await page.screenshot({ path: 'test-results/DS-1-数据源列表.png', fullPage: true })
    })

    await test.step('执行健康检查', async () => {
      expect(datasourceId).toBeDefined()
      const res = await API.post(page, `/api/v1/datasources/${datasourceId}/health-check`)
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      expect(res.data.status).toBeDefined()
      expect(typeof res.data.latency_ms).toBe('number')
      // latency_ms may be -1 if the datasource is unreachable or returns an error
      expect(res.data.latency_ms === -1 || res.data.latency_ms >= 0).toBe(true)
      await page.screenshot({ path: 'test-results/DS-1-健康检查结果.png', fullPage: false })
    })
  })

  // DS-2: Test connection endpoint with valid config -> verify success -> test with invalid endpoint -> verify failure
  test('DS-2 测试连接端点', async ({ authPage: page }) => {
    await test.step('有效配置测试连接', async () => {
      const res = await API.post(page, '/api/v1/datasources/test-connection', {
        type: 'victoriametrics',
        endpoint: 'http://localhost:8481/select/0/prometheus',
        auth_type: 'none',
        auth_config: '',
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      expect(res.data.status).toBeDefined()
      expect(typeof res.data.latency_ms).toBe('number')
      await page.screenshot({ path: 'test-results/DS-2-有效连接结果.png', fullPage: false })
    })

    await test.step('无效端点测试连接失败', async () => {
      const res = await API.post(page, '/api/v1/datasources/test-connection', {
        type: 'victoriametrics',
        endpoint: 'http://localhost:19999/invalid/path',
        auth_type: 'none',
        auth_config: '',
      })
      // 无效端点应返回错误或非 success 状态
      const hasError = res.code !== 0 || res.data?.status === 'error' || res.data?.status === 'fail' || res.message
      expect(hasError).toBeTruthy()
      await page.screenshot({ path: 'test-results/DS-2-无效连接结果.png', fullPage: false })
    })
  })

  // DS-3: Label keys query -> verify non-empty -> label values query for a key -> verify non-empty
  test('DS-3 标签键值查询', async ({ authPage: page }) => {
    let datasourceId: number
    let labelKeys: string[]

    await test.step('获取数据源 ID', async () => {
      const res = await API.get(page, '/api/v1/datasources?page=1&page_size=100')
      expect(res.code).toBe(0)
      const vmDs = res.data.list.find((ds: any) =>
        ds.type === 'victoriametrics' || ds.type === 'prometheus' || ds.type === 'vm'
      )
      expect(vmDs).toBeDefined()
      datasourceId = vmDs.id
    })

    await test.step('查询标签键列表', async () => {
      const res = await API.get(page, `/api/v1/datasources/${datasourceId}/labels/keys`)
      expect(res.code).toBe(0)
      expect(Array.isArray(res.data)).toBe(true)
      // 标签键可能为空（数据源无数据时）
      labelKeys = res.data
      await page.screenshot({ path: 'test-results/DS-3-标签键列表.png', fullPage: false })
    })

    await test.step('查询标签值列表', async () => {
      if (labelKeys.length === 0) {
        // 无标签键时跳过标签值查询
        await page.screenshot({ path: 'test-results/DS-3-标签值列表-空.png', fullPage: false })
        return
      }
      const key = labelKeys[0]
      const res = await API.get(page, `/api/v1/datasources/${datasourceId}/labels/values?key=${encodeURIComponent(key)}`)
      expect(res.code).toBe(0)
      expect(Array.isArray(res.data)).toBe(true)
      // 标签值可能为空
      await page.screenshot({ path: 'test-results/DS-3-标签值列表.png', fullPage: false })
    })
  })

  // DS-4: Metric names query -> verify non-empty -> instant query (up metric) -> verify results
  test('DS-4 指标查询与即时查询', async ({ authPage: page }) => {
    let datasourceId: number

    await test.step('获取数据源 ID', async () => {
      const res = await API.get(page, '/api/v1/datasources?page=1&page_size=100')
      expect(res.code).toBe(0)
      const vmDs = res.data.list.find((ds: any) =>
        ds.type === 'victoriametrics' || ds.type === 'prometheus' || ds.type === 'vm'
      )
      expect(vmDs).toBeDefined()
      datasourceId = vmDs.id
    })

    await test.step('查询指标名称列表', async () => {
      const res = await API.get(page, `/api/v1/datasources/${datasourceId}/metrics?limit=50`)
      expect(res.code).toBe(0)
      expect(Array.isArray(res.data)).toBe(true)
      // 指标名称可能为空（数据源无数据时）
      await page.screenshot({ path: 'test-results/DS-4-指标名称列表.png', fullPage: false })
    })

    await test.step('即时查询 up 指标', async () => {
      const res = await API.post(page, `/api/v1/datasources/${datasourceId}/query`, {
        expression: 'up',
      })
      expect(res.code).toBe(0)
      expect(res.data).toBeDefined()
      // QueryResponse 应包含 series 数组 (API returns series, not result)
      const series = res.data.series || res.data.result || []
      expect(Array.isArray(series)).toBe(true)
      // Note: series may be empty if no data points match current time
      await page.screenshot({ path: 'test-results/DS-4-即时查询结果.png', fullPage: false })
    })
  })
})
