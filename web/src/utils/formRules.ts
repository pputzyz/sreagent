/**
 * Unified form validation rules for Naive UI forms.
 *
 * Usage:
 *   import { requiredRule, emailRule, promqlRule } from '@/utils/formRules'
 *   <n-form :rules="{ name: requiredRule('Name'), email: emailRule() }">
 */
import type { FormItemRule } from 'naive-ui'

// ─── Required ───

/** Field is required (non-empty string, non-null, non-undefined). */
export function requiredRule(label = 'This field'): FormItemRule[] {
  return [
    {
      required: true,
      message: `${label} is required`,
      trigger: ['blur', 'input'],
    },
  ]
}

// ─── Email ───

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

/** Validates email format. */
export function emailRule(label = 'Email'): FormItemRule[] {
  return [
    ...requiredRule(label),
    {
      pattern: EMAIL_RE,
      message: `Please enter a valid ${label.toLowerCase()}`,
      trigger: ['blur'],
    },
  ]
}

// ─── URL ───

const URL_RE = /^https?:\/\/.+/i

/** Validates URL format (must start with http:// or https://). */
export function urlRule(label = 'URL'): FormItemRule[] {
  return [
    ...requiredRule(label),
    {
      pattern: URL_RE,
      message: `Please enter a valid ${label.toLowerCase()} (http:// or https://)`,
      trigger: ['blur'],
    },
  ]
}

/** Optional URL (empty is ok, but if present must be valid). */
export function optionalUrlRule(label = 'URL'): FormItemRule[] {
  return [
    {
      validator: (_rule: FormItemRule, value: string) => {
        if (!value) return true // optional
        return URL_RE.test(value) ? true : new Error(`Please enter a valid ${label.toLowerCase()}`)
      },
      trigger: ['blur'],
    },
  ]
}

// ─── JSON ───

/** Validates that the value is valid JSON. */
export function jsonRule(label = 'JSON'): FormItemRule[] {
  return [
    ...requiredRule(label),
    {
      validator: (_rule: FormItemRule, value: string) => {
        if (!value) return true
        try {
          JSON.parse(value)
          return true
        } catch {
          return new Error(`Invalid ${label} format`)
        }
      },
      trigger: ['blur'],
    },
  ]
}

/** Optional JSON (empty is ok, but if present must be valid). */
export function optionalJsonRule(label = 'JSON'): FormItemRule[] {
  return [
    {
      validator: (_rule: FormItemRule, value: string) => {
        if (!value) return true
        try {
          JSON.parse(value)
          return true
        } catch {
          return new Error(`Invalid ${label} format`)
        }
      },
      trigger: ['blur'],
    },
  ]
}

// ─── PromQL ───

const PROMQL_BASIC_RE = /^[a-zA-Z_:][a-zA-Z0-9_:]*(\{.*?\})?(\[.*?\])?$/

/** Basic PromQL expression validation (metric name + optional selectors/range). */
export function promqlRule(label = 'PromQL expression'): FormItemRule[] {
  return [
    ...requiredRule(label),
    {
      validator: (_rule: FormItemRule, value: string) => {
        if (!value) return true
        const trimmed = value.trim()
        if (!trimmed) return new Error(`${label} cannot be empty`)
        // Very basic check: must look like a PromQL expression
        // (not a full parser, just sanity check)
        if (/^[a-zA-Z_:]/.test(trimmed) || trimmed.startsWith('(') || trimmed.startsWith('sum') || trimmed.startsWith('rate') || trimmed.startsWith('increase')) {
          return true
        }
        return new Error(`Invalid ${label} — must start with a metric name or function`)
      },
      trigger: ['blur'],
    },
  ]
}

// ─── Min / Max Length ───

/** Minimum length validation. */
export function minLengthRule(min: number, label = 'This field'): FormItemRule[] {
  return [
    {
      validator: (_rule: FormItemRule, value: string) => {
        if (!value) return true // let required handle empty
        return value.length >= min ? true : new Error(`${label} must be at least ${min} characters`)
      },
      trigger: ['blur'],
    },
  ]
}

/** Maximum length validation. */
export function maxLengthRule(max: number, label = 'This field'): FormItemRule[] {
  return [
    {
      validator: (_rule: FormItemRule, value: string) => {
        if (!value) return true
        return value.length <= max ? true : new Error(`${label} must be at most ${max} characters`)
      },
      trigger: ['input', 'blur'],
    },
  ]
}

/** Combined min + max length. */
export function lengthRule(min: number, max: number, label = 'This field'): FormItemRule[] {
  return [
    ...minLengthRule(min, label),
    ...maxLengthRule(max, label),
  ]
}

// ─── Severity Enum ───

const SEVERITY_VALUES = ['critical', 'warning', 'info', 'p0', 'p1', 'p2', 'p3', 'p4']

/** Validates that the value is a known severity level. */
export function severityRule(label = 'Severity'): FormItemRule[] {
  return [
    ...requiredRule(label),
    {
      validator: (_rule: FormItemRule, value: string) => {
        if (!value) return true
        return SEVERITY_VALUES.includes(value) ? true : new Error(`Invalid ${label}: ${value}`)
      },
      trigger: ['blur', 'change'],
    },
  ]
}

// ─── Positive Integer ───

/** Validates that the value is a positive integer (> 0). */
export function positiveIntRule(label = 'This field'): FormItemRule[] {
  return [
    ...requiredRule(label),
    {
      validator: (_rule: FormItemRule, value: number | string) => {
        if (value == null || value === '') return true
        const num = Number(value)
        if (!Number.isInteger(num) || num <= 0) {
          return new Error(`${label} must be a positive integer`)
        }
        return true
      },
      trigger: ['blur'],
    },
  ]
}

/** Non-negative integer (>= 0). */
export function nonNegativeIntRule(label = 'This field'): FormItemRule[] {
  return [
    {
      validator: (_rule: FormItemRule, value: number | string) => {
        if (value == null || value === '') return true
        const num = Number(value)
        if (!Number.isInteger(num) || num < 0) {
          return new Error(`${label} must be a non-negative integer`)
        }
        return true
      },
      trigger: ['blur'],
    },
  ]
}

// ─── Port Number ───

/** Validates port number (1-65535). */
export function portRule(label = 'Port'): FormItemRule[] {
  return [
    ...requiredRule(label),
    {
      validator: (_rule: FormItemRule, value: number | string) => {
        if (value == null || value === '') return true
        const num = Number(value)
        if (!Number.isInteger(num) || num < 1 || num > 65535) {
          return new Error(`${label} must be between 1 and 65535`)
        }
        return true
      },
      trigger: ['blur'],
    },
  ]
}

// ─── Pattern (Regex) ───

/** Validates against a custom regex pattern. */
export function patternRule(pattern: RegExp, message: string): FormItemRule[] {
  return [
    {
      pattern,
      message,
      trigger: ['blur'],
    },
  ]
}
