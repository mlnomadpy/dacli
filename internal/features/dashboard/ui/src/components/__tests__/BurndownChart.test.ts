import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BurndownChart from '../BurndownChart.vue'
import type { Burndown } from '@/types'

function make(over: Partial<Burndown> = {}): Burndown {
  return { done_points: 88.5, remaining_points: 31, unestimated: 4, per_day: [], ...over }
}

describe('BurndownChart', () => {
  it('empty per_day: note plus the numeric summary still shows', () => {
    const w = mount(BurndownChart, { props: { burndown: make() } })
    expect(w.text()).toContain('nothing completed yet')
    expect(w.text()).toContain('88.5 done · 31.0 remaining pts · 4 unestimated')
    expect(w.find('.chart').exists()).toBe(false)
  })

  it('renders one bar per day in the server-given order (never re-sorted)', () => {
    const w = mount(BurndownChart, {
      props: {
        burndown: make({
          per_day: [
            { day: '2026-07-20', points: 12 },
            { day: '2026-07-21', points: 8.5 },
            { day: '2026-07-22', points: 4 },
          ],
        }),
      },
    })
    const bars = w.findAll('.bar')
    expect(bars).toHaveLength(3)
    // Order preserved; each bar carries an accessible per-day label.
    expect(bars[0].attributes('aria-label')).toBe('2026-07-20: 12.0 points')
    expect(bars[2].attributes('aria-label')).toBe('2026-07-22: 4.0 points')
    // The tallest day (12) is the 100% reference.
    expect((bars[0].element as HTMLElement).style.height).toBe('100%')
  })
})
