import { describe, it, expect } from 'vitest'
import { getSeverityType } from './alert'
import { severityType } from './severity'

describe('getSeverityType', () => {
  it('returns error for critical', () => {
    expect(getSeverityType('critical')).toBe('error')
  })

  it('returns error for p0', () => {
    expect(getSeverityType('p0')).toBe('error')
  })

  it('returns warning for p1', () => {
    expect(getSeverityType('p1')).toBe('warning')
  })

  it('returns warning for p2', () => {
    expect(getSeverityType('p2')).toBe('warning')
  })

  it('returns warning for warning', () => {
    expect(getSeverityType('warning')).toBe('warning')
  })

  it('returns info for p3', () => {
    expect(getSeverityType('p3')).toBe('info')
  })

  it('returns info for info', () => {
    expect(getSeverityType('info')).toBe('info')
  })

  it('returns default for p4', () => {
    expect(getSeverityType('p4')).toBe('default')
  })

  it('returns default for unknown', () => {
    expect(getSeverityType('unknown')).toBe('default')
  })
})

describe('severityType re-export', () => {
  it('is an alias for getSeverityType', () => {
    expect(severityType).toBe(getSeverityType)
  })
})
