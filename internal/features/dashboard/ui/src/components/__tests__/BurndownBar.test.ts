import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BurndownBar from '../BurndownBar.vue'
import type { Burndown } from '@/types'

function make(over: Partial<Burndown> = {}): Burndown {
  return { done_points: 0, remaining_points: 0, unestimated: 0, per_day: [], ...over }
}

describe('BurndownBar', () => {
  it('is 0-safe: empty track and caption when nothing is estimated', () => {
    const w = mount(BurndownBar, { props: { burndown: make() } })
    expect(w.text()).toContain('no estimated points yet')
    // No segments rendered — the empty --border track carries the full width.
    expect(w.find('.done-seg').exists()).toBe(false)
    expect(w.find('.rem-seg').exists()).toBe(false)
  })

  it('splits the bar by done/total and captions the points', () => {
    const w = mount(BurndownBar, {
      props: { burndown: make({ done_points: 75, remaining_points: 25 }) },
    })
    expect((w.find('.done-seg').element as HTMLElement).style.width).toBe('75%')
    expect((w.find('.rem-seg').element as HTMLElement).style.width).toBe('25%')
    expect(w.text()).toContain('75.0 done · 25.0 remaining pts')
  })

  it('appends the unestimated caption only when > 0', () => {
    const some = mount(BurndownBar, {
      props: { burndown: make({ done_points: 10, remaining_points: 0, unestimated: 4 }) },
    })
    expect(some.text()).toContain('· 4 unestimated')
    const none = mount(BurndownBar, {
      props: { burndown: make({ done_points: 10, remaining_points: 0, unestimated: 0 }) },
    })
    expect(none.text()).not.toContain('unestimated')
  })
})
