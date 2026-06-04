import { test, expect } from '../fixtures/auth'
import { API } from '../helpers/api'

test('permission test', async ({ authPage: page }) => {
  // Test GET (should work)
  const getResp = await API.get(page, '/api/v1/alert-rules')
  console.log('GET response:', JSON.stringify(getResp))

  // Test POST (might fail with 403)
  const postResp = await API.post(page, '/api/v1/alert-rules', {
    name: 'test',
    expression: 'up == 0',
    severity: 'critical',
    datasource_type: 'prometheus'
  })
  console.log('POST response:', JSON.stringify(postResp))
})
