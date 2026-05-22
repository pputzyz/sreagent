import { describe, it, expect } from 'vitest'
import { formatValue, type ValueFormat } from './valueFormatter'

describe('formatValue — bytes', () => {
  it('formats 0 bytes', () => {
    expect(formatValue(0, 'bytes')).toBe('0 B')
  })

  it('formats 1 KB', () => {
    expect(formatValue(1024, 'bytes')).toBe('1.00 KB')
  })

  it('formats 1 MB', () => {
    expect(formatValue(1024 * 1024, 'bytes')).toBe('1.00 MB')
  })

  it('formats 1 GB', () => {
    expect(formatValue(1024 ** 3, 'bytes')).toBe('1.00 GB')
  })

  it('formats negative bytes with sign', () => {
    const result = formatValue(-1024, 'bytes')
    expect(result).toContain('-')
    expect(result).toContain('KB')
  })

  it('formats sub-byte values', () => {
    const result = formatValue(0.5, 'bytes')
    expect(result).toContain('B')
  })

  it('formats 1 TB', () => {
    const result = formatValue(1024 ** 4, 'bytes')
    expect(result).toContain('TB')
  })

  it('formats 1 PB', () => {
    const result = formatValue(1024 ** 5, 'bytes')
    expect(result).toContain('PB')
  })

  it('respects custom decimals', () => {
    expect(formatValue(1536, 'bytes', 0)).toBe('2 KB')
  })
})

describe('formatValue — seconds (duration)', () => {
  it('formats 0 seconds', () => {
    expect(formatValue(0, 'seconds')).toBe('0s')
  })

  it('formats sub-millisecond', () => {
    const result = formatValue(0.0005, 'seconds')
    expect(result).toContain('ns')
  })

  it('formats milliseconds', () => {
    const result = formatValue(0.5, 'seconds')
    expect(result).toContain('ms')
  })

  it('formats seconds', () => {
    expect(formatValue(30, 'seconds')).toContain('s')
  })

  it('formats minutes', () => {
    expect(formatValue(120, 'seconds')).toContain('m')
  })

  it('formats hours', () => {
    expect(formatValue(3600, 'seconds')).toContain('h')
  })

  it('formats days', () => {
    expect(formatValue(86400, 'seconds')).toContain('d')
  })
})

describe('formatValue — percent', () => {
  it('formats percent', () => {
    expect(formatValue(75.5, 'percent')).toBe('75.50%')
  })

  it('formats percentUnit (0-1 range)', () => {
    expect(formatValue(0.755, 'percentUnit')).toBe('75.50%')
  })
})

describe('formatValue — short', () => {
  it('formats thousands as K', () => {
    expect(formatValue(1500, 'short')).toContain('K')
  })

  it('formats millions as M', () => {
    expect(formatValue(2_500_000, 'short')).toContain('M')
  })

  it('formats billions as B', () => {
    expect(formatValue(3_000_000_000, 'short')).toContain('B')
  })

  it('formats trillions as T', () => {
    const result = formatValue(1.5e12, 'short')
    expect(result).toContain('T')
  })

  it('formats small numbers without suffix', () => {
    expect(formatValue(42, 'short')).toBe('42.00')
  })
})

describe('formatValue — scientific', () => {
  it('formats in scientific notation', () => {
    const result = formatValue(12345, 'scientific')
    expect(result).toContain('e')
  })
})

describe('formatValue — none (default)', () => {
  it('formats integers as-is', () => {
    expect(formatValue(100, 'none')).toBe('100')
  })

  it('formats floats with decimals', () => {
    expect(formatValue(3.14159, 'none')).toBe('3.14')
  })
})

describe('formatValue — edge cases', () => {
  it('returns "-" for NaN', () => {
    expect(formatValue(NaN, 'bytes')).toBe('-')
  })

  it('returns "-" for NaN with any format', () => {
    expect(formatValue(NaN)).toBe('-')
    expect(formatValue(NaN, 'percent')).toBe('-')
    expect(formatValue(NaN, 'short')).toBe('-')
  })
})
