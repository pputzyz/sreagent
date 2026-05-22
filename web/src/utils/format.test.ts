import { describe, it, expect } from 'vitest'
import { formatDuration, kvArrayToRecord, recordToKVArray, getErrorMessage } from './format'

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
