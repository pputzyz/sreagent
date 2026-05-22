import { describe, it, expect, beforeEach } from 'vitest'
import { useFilterMemory } from './useFilterMemory'

describe('useFilterMemory', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('restore returns default when nothing stored', () => {
    const mem = useFilterMemory('test-page')
    expect(mem.restore('search', '')).toBe('')
    expect(mem.restore('count', 0)).toBe(0)
    expect(mem.restore('tag', null)).toBeNull()
  })

  it('save + restore round-trips values', async () => {
    const mem = useFilterMemory('test-page', 0)
    mem.save({ search: 'hello', count: 42 })
    // save is debounced with 0ms, wait a tick
    await new Promise((r) => setTimeout(r, 10))
    expect(mem.restore('search', '')).toBe('hello')
    expect(mem.restore('count', 0)).toBe(42)
  })

  it('restoreAll returns all stored fields', async () => {
    const mem = useFilterMemory('test-page', 0)
    mem.save({ a: '1', b: '2' })
    await new Promise((r) => setTimeout(r, 10))
    expect(mem.restoreAll()).toEqual({ a: '1', b: '2' })
  })

  it('clear removes stored data', async () => {
    const mem = useFilterMemory('test-page', 0)
    mem.save({ search: 'test' })
    await new Promise((r) => setTimeout(r, 10))
    mem.clear()
    expect(mem.restore('search', '')).toBe('')
  })

  it('different pageKeys are isolated', async () => {
    const memA = useFilterMemory('page-a', 0)
    const memB = useFilterMemory('page-b', 0)
    memA.save({ val: 'fromA' })
    await new Promise((r) => setTimeout(r, 10))
    expect(memA.restore('val', '')).toBe('fromA')
    expect(memB.restore('val', '')).toBe('')
  })
})
