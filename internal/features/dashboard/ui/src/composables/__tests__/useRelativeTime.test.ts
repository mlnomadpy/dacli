import { describe, it, expect } from 'vitest'
import { ago, duration, freshness } from '../useRelativeTime'

const NOW = Date.parse('2026-07-23T16:10:00Z')

describe('ago', () => {
  it('returns "-" for empty or invalid input', () => {
    expect(ago(null, NOW)).toBe('-')
    expect(ago(undefined, NOW)).toBe('-')
    expect(ago('not-a-date', NOW)).toBe('-')
  })

  it('formats seconds, minutes, hours, days', () => {
    expect(ago('2026-07-23T16:09:40Z', NOW)).toBe('20s ago')
    expect(ago('2026-07-23T16:06:00Z', NOW)).toBe('4m ago')
    expect(ago('2026-07-23T14:50:00Z', NOW)).toBe('1h 20m ago')
    expect(ago('2026-07-21T16:10:00Z', NOW)).toBe('2d ago')
  })

  it('clamps a future timestamp to 0s', () => {
    expect(ago('2026-07-23T16:11:00Z', NOW)).toBe('0s ago')
  })
})

describe('duration', () => {
  it('formats sub-minute, sub-hour, and hour+ durations', () => {
    expect(duration(45)).toBe('45s')
    expect(duration(600)).toBe('10m 0s')
    expect(duration(3840)).toBe('1h 4m')
  })

  it('is 0-safe and clamps negatives', () => {
    expect(duration(0)).toBe('0s')
    expect(duration(-5)).toBe('0s')
    expect(duration(Number.NaN)).toBe('0s')
  })
})

describe('freshness', () => {
  it('buckets by the documented UI-local thresholds', () => {
    expect(freshness('2026-07-23T16:09:40Z', NOW)).toBe('fresh') // 20s
    expect(freshness('2026-07-23T16:07:00Z', NOW)).toBe('idle') // 3m
    expect(freshness('2026-07-23T16:00:00Z', NOW)).toBe('stale') // 10m
  })

  it('treats missing/invalid as stale', () => {
    expect(freshness(null, NOW)).toBe('stale')
    expect(freshness('nope', NOW)).toBe('stale')
  })
})
