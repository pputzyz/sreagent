/**
 * Format a numeric value with unit-aware display.
 * Inspired by n9e's valueFormatter.
 */

const byteUnits = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
const timeUnits = ['ns', 'µs', 'ms', 's', 'm', 'h', 'd']

export type ValueFormat =
  | 'none'
  | 'bytes'
  | 'seconds'
  | 'milliseconds'
  | 'percent'
  | 'percentUnit' // 0-1 range → 0-100%
  | 'short'
  | 'scientific'

export function formatValue(value: number, format: ValueFormat = 'none', decimals = 2): string {
  if (value == null || isNaN(value)) return '-'

  switch (format) {
    case 'bytes':
      return formatBytes(value, decimals)
    case 'seconds':
      return formatDuration(value, decimals)
    case 'milliseconds':
      return formatDuration(value / 1000, decimals)
    case 'percent':
      return value.toFixed(decimals) + '%'
    case 'percentUnit':
      return (value * 100).toFixed(decimals) + '%'
    case 'scientific':
      return value.toExponential(decimals)
    case 'short':
      return formatShort(value, decimals)
    default:
      return formatAuto(value, decimals)
  }
}

function formatBytes(bytes: number, decimals: number): string {
  if (bytes === 0) return '0 B'
  const abs = Math.abs(bytes)
  const sign = bytes < 0 ? '-' : ''
  const i = Math.floor(Math.log(abs) / Math.log(1024))
  if (i < 0) return sign + abs.toFixed(decimals) + ' B'
  const idx = Math.min(i, byteUnits.length - 1)
  return sign + (abs / Math.pow(1024, idx)).toFixed(decimals) + ' ' + byteUnits[idx]
}

function formatDuration(seconds: number, decimals: number): string {
  if (seconds === 0) return '0s'
  const abs = Math.abs(seconds)
  const sign = seconds < 0 ? '-' : ''

  if (abs < 0.001) return sign + (seconds * 1e6).toFixed(0) + 'ns'
  if (abs < 1) return sign + (seconds * 1000).toFixed(decimals) + 'ms'
  if (abs < 60) return sign + seconds.toFixed(decimals) + 's'
  if (abs < 3600) return sign + (seconds / 60).toFixed(decimals) + 'm'
  if (abs < 86400) return sign + (seconds / 3600).toFixed(decimals) + 'h'
  return sign + (seconds / 86400).toFixed(decimals) + 'd'
}

function formatShort(value: number, decimals: number): string {
  const abs = Math.abs(value)
  const sign = value < 0 ? '-' : ''
  if (abs >= 1e12) return sign + (abs / 1e12).toFixed(decimals) + 'T'
  if (abs >= 1e9) return sign + (abs / 1e9).toFixed(decimals) + 'B'
  if (abs >= 1e6) return sign + (abs / 1e6).toFixed(decimals) + 'M'
  if (abs >= 1e3) return sign + (abs / 1e3).toFixed(decimals) + 'K'
  return sign + abs.toFixed(decimals)
}

function formatAuto(value: number, decimals: number): string {
  if (Number.isInteger(value) && Math.abs(value) < 1e15) return value.toString()
  return value.toFixed(decimals)
}
