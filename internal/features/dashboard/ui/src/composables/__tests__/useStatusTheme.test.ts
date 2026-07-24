import { describe, it, expect } from 'vitest'
import { statusColor, count } from '../useStatusTheme'

describe('statusColor', () => {
  it('maps each status to its single-source-of-truth token', () => {
    expect(statusColor('open')).toBe('var(--open)')
    expect(statusColor('active')).toBe('var(--active)')
    expect(statusColor('blocked')).toBe('var(--blocked)')
    expect(statusColor('done')).toBe('var(--done)')
  })
})

describe('count', () => {
  it('returns 0 for a missing key (counts is a partial map)', () => {
    expect(count({ open: 12, done: 26 }, 'active')).toBe(0)
    expect(count({}, 'open')).toBe(0)
    expect(count(null, 'done')).toBe(0)
    expect(count(undefined, 'blocked')).toBe(0)
  })

  it('returns the present value', () => {
    expect(count({ open: 12, active: 3 }, 'open')).toBe(12)
    expect(count({ open: 12, active: 3 }, 'active')).toBe(3)
  })
})
