import { describe, it, expect } from 'vitest'
import { computeTimeStep } from './timeStep'

describe('computeTimeStep', () => {
  it('returns 15s for <= 5 min (300s)', () => {
    expect(computeTimeStep(300)).toBe('15s')
  })

  it('returns 15s for very short durations', () => {
    expect(computeTimeStep(1)).toBe('15s')
    expect(computeTimeStep(60)).toBe('15s')
  })

  it('returns 30s for <= 1 hour (3600s)', () => {
    expect(computeTimeStep(301)).toBe('30s')
    expect(computeTimeStep(3600)).toBe('30s')
  })

  it('returns 1m for <= 6 hours (21600s)', () => {
    expect(computeTimeStep(3601)).toBe('1m')
    expect(computeTimeStep(21600)).toBe('1m')
  })

  it('returns 5m for <= 24 hours (86400s)', () => {
    expect(computeTimeStep(21601)).toBe('5m')
    expect(computeTimeStep(86400)).toBe('5m')
  })

  it('returns 15m for <= 7 days (604800s)', () => {
    expect(computeTimeStep(86401)).toBe('15m')
    expect(computeTimeStep(604800)).toBe('15m')
  })

  it('returns 1h for > 7 days', () => {
    expect(computeTimeStep(604801)).toBe('1h')
    expect(computeTimeStep(2592000)).toBe('1h') // 30 days
  })

  it('returns 15s for 0', () => {
    expect(computeTimeStep(0)).toBe('15s')
  })

  it('returns 15s for negative values', () => {
    expect(computeTimeStep(-100)).toBe('15s')
  })

  it('returns step for boundary at exactly 5 min', () => {
    expect(computeTimeStep(300)).toBe('15s')
    expect(computeTimeStep(301)).toBe('30s')
  })
})
