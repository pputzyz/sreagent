import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { formatTime, relTime, formatDuration, kvArrayToRecord, recordToKVArray, getErrorMessage } from './format'

describe('formatDuration', () => {
  it('returns 0s for 0', () => {
    expect(formatDuration(0)).toBe('0s')
  })

  it('returns 0s for negative', () => {
    expect(formatDuration(-5)).toBe('0s')
  })

  it('formats seconds only', () => {
    expect(formatDuration(45)).toBe('45s')
  })

  it('formats minutes and seconds', () => {
    expect(formatDuration(270)).toBe('4m 30s')
  })

  it('formats hours, minutes, seconds', () => {
    expect(formatDuration(3661)).toBe('1h 1m 1s')
  })

  it('formats days', () => {
    expect(formatDuration(90061)).toBe('1d 1h 1m 1s')
  })
})

describe('kvArrayToRecord', () => {
  it('converts pairs to record', () => {
    expect(kvArrayToRecord([
      { key: 'a', value: '1' },
      { key: 'b', value: '2' },
    ])).toEqual({ a: '1', b: '2' })
  })

  it('skips empty keys', () => {
    expect(kvArrayToRecord([
      { key: '', value: '1' },
      { key: 'b', value: '2' },
    ])).toEqual({ b: '2' })
  })

  it('trims keys', () => {
    expect(kvArrayToRecord([
      { key: '  a  ', value: '1' },
    ])).toEqual({ a: '1' })
  })

  it('returns empty record for empty array', () => {
    expect(kvArrayToRecord([])).toEqual({})
  })
})

describe('recordToKVArray', () => {
  it('converts record to pairs', () => {
    expect(recordToKVArray({ a: '1', b: '2' })).toEqual([
      { key: 'a', value: '1' },
      { key: 'b', value: '2' },
    ])
  })

  it('returns empty array for null', () => {
    expect(recordToKVArray(null)).toEqual([])
  })

  it('returns empty array for undefined', () => {
    expect(recordToKVArray(undefined)).toEqual([])
  })
})

describe('getErrorMessage', () => {
  it('extracts message from Error', () => {
    expect(getErrorMessage(new Error('test'))).toBe('test')
  })

  it('extracts message from object with message', () => {
    expect(getErrorMessage({ message: 'foo' })).toBe('foo')
  })

  it('converts string to string', () => {
    expect(getErrorMessage('raw string')).toBe('raw string')
  })

  it('converts number to string', () => {
    expect(getErrorMessage(42)).toBe('42')
  })
})

describe('formatTime', () => {
  it('returns "-" for null', () => {
    expect(formatTime(null)).toBe('-')
  })

  it('returns "-" for undefined', () => {
    expect(formatTime(undefined)).toBe('-')
  })

  it('returns "-" for empty string', () => {
    expect(formatTime('')).toBe('-')
  })

  it('returns "-" for invalid date string', () => {
    expect(formatTime('not-a-date')).toBe('-')
  })

  it('formats a valid ISO date string', () => {
    const result = formatTime('2024-06-15T10:30:00Z')
    // Should contain date components — exact format depends on locale
    expect(result).not.toBe('-')
    expect(result.length).toBeGreaterThan(5)
  })

  it('formats a valid date with time components', () => {
    const result = formatTime('2024-01-01T00:00:00Z')
    expect(result).not.toBe('-')
  })
})

describe('relTime', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2024-06-15T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns "—" for null', () => {
    expect(relTime(null)).toBe('—')
  })

  it('returns "—" for undefined', () => {
    expect(relTime(undefined)).toBe('—')
  })

  it('returns "—" for empty string', () => {
    expect(relTime('')).toBe('—')
  })

  it('shows seconds ago for recent timestamps', () => {
    const ts = new Date('2024-06-15T11:59:30Z').toISOString() // 30s ago
    const result = relTime(ts)
    expect(result).toContain('30')
    expect(result).toContain('s')
  })

  it('shows minutes ago', () => {
    const ts = new Date('2024-06-15T11:55:00Z').toISOString() // 5m ago
    const result = relTime(ts)
    expect(result).toContain('5')
    expect(result).toContain('m')
  })

  it('shows hours ago', () => {
    const ts = new Date('2024-06-15T09:00:00Z').toISOString() // 3h ago
    const result = relTime(ts)
    expect(result).toContain('3')
    expect(result).toContain('h')
  })

  it('shows days ago', () => {
    const ts = new Date('2024-06-13T12:00:00Z').toISOString() // 2d ago
    const result = relTime(ts)
    expect(result).toContain('2')
    expect(result).toContain('d')
  })

  it('uses t function when provided', () => {
    const t = vi.fn((key: string, params?: Record<string, unknown>) => `${key}:${JSON.stringify(params)}`)
    const ts = new Date('2024-06-15T11:59:30Z').toISOString() // 30s ago
    relTime(ts, t)
    expect(t).toHaveBeenCalledWith('alert.secsAgo', expect.objectContaining({ n: expect.any(Number) }))
  })

  it('returns 0s ago for very recent timestamps', () => {
    const ts = new Date('2024-06-15T12:00:00Z').toISOString() // now
    const result = relTime(ts)
    expect(result).toContain('0')
    expect(result).toContain('s')
  })
})
