import { describe, it, expect } from 'vitest'
import { severityType } from './severity'

describe('severityType', () => {
  it('returns error for critical', () => {
    expect(severityType('critical')).toBe('error')
  })

  it('returns error for p0', () => {
    expect(severityType('p0')).toBe('error')
  })

  it('returns error for p1', () => {
    expect(severityType('p1')).toBe('error')
  })

  it('returns warning for warning', () => {
    expect(severityType('warning')).toBe('warning')
  })

  it('returns warning for p2', () => {
    expect(severityType('p2')).toBe('warning')
  })

  it('returns info for info', () => {
    expect(severityType('info')).toBe('info')
  })

  it('returns info for p4', () => {
    expect(severityType('p4')).toBe('info')
  })

  it('returns info for unknown', () => {
    expect(severityType('unknown')).toBe('info')
  })
})
