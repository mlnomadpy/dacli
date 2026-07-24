import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BurnRate from '../BurnRate.vue'
import { emptyBurn } from '@/types'
import type { Burn } from '@/types'

function make(over: Partial<Burn> = {}): Burn {
  return { ...emptyBurn(), ...over }
}

describe('BurnRate', () => {
  it('is 0-safe when nothing has burned: empty note, no bars, no alert', () => {
    const w = mount(BurnRate, { props: { burn: make() } })
    expect(w.text()).toContain('no usage recorded yet')
    expect(w.findAll('.bar')).toHaveLength(0)
    expect(w.find('[role="alert"]').exists()).toBe(false)
    expect(w.find('section').classes()).not.toContain('alert')
    // No NaN leaks into any style.
    expect(w.html()).not.toContain('NaN')
  })

  it('YELLS when the rate reaches the alert threshold: danger panel + live banner', () => {
    const w = mount(BurnRate, {
      props: {
        burn: make({
          ceiling: 100,
          rate: 500,
          ratio: 5,
          alert: true,
          series: [{ day: '2026-07-24', tokens: 500, cost_usd: 0.6, runs: 1, per_run: 500 }],
        }),
      },
    })
    // The whole panel turns danger, not a passive line.
    expect(w.find('section').classes()).toContain('alert')
    // An assertive live region announces the overspend.
    const alert = w.find('[role="alert"]')
    expect(alert.exists()).toBe(true)
    expect(alert.attributes('aria-live')).toBe('assertive')
    expect(alert.text()).toContain('5.0× the calibrated ceiling of 100 output_tokens/run')
    // The breaching day's bar is painted hot.
    expect(w.find('.bar.hot').exists()).toBe(true)
    // Both the rate and the ceiling are rendered.
    expect(w.text()).toContain('500')
    expect(w.text()).toContain('100')
  })

  it('stays a passive line below the threshold: no danger, no banner, no hot bar', () => {
    const w = mount(BurnRate, {
      props: {
        burn: make({
          ceiling: 100,
          rate: 140, // 1.4× — under the 1.5× alert_at
          ratio: 1.4,
          alert: false,
          series: [{ day: '2026-07-24', tokens: 140, cost_usd: 0.2, runs: 1, per_run: 140 }],
        }),
      },
    })
    expect(w.find('section').classes()).not.toContain('alert')
    expect(w.find('[role="alert"]').exists()).toBe(false)
    expect(w.find('.bar.hot').exists()).toBe(false)
    expect(w.find('.bar').exists()).toBe(true) // the line is still drawn
    expect(w.text()).toContain('1.4× ceiling')
  })

  it('draws a ceiling reference line only when a calibrated ceiling exists', () => {
    const withCeiling = mount(BurnRate, {
      props: {
        burn: make({
          ceiling: 100,
          series: [{ day: '2026-07-24', tokens: 80, cost_usd: 0, runs: 1, per_run: 80 }],
        }),
      },
    })
    expect(withCeiling.find('.ceiling-line').exists()).toBe(true)

    const noCeiling = mount(BurnRate, {
      props: {
        burn: make({
          ceiling: 0, // no token history yet
          series: [{ day: '2026-07-24', tokens: 80, cost_usd: 0, runs: 1, per_run: 80 }],
        }),
      },
    })
    expect(noCeiling.find('.ceiling-line').exists()).toBe(false)
    // With no ceiling there can be no alert to raise.
    expect(noCeiling.find('[role="alert"]').exists()).toBe(false)
  })
})
