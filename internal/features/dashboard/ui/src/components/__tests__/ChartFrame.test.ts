import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ChartFrame from '../ChartFrame.vue'
import type { ChartContract } from '@/types'

function contract(over: Partial<ChartContract> = {}): ChartContract {
  return {
    id: 'quality',
    title: 'Quality trend',
    metric_definition: 'Exact-tree acceptance by day',
    unit: 'percent',
    window: '2026-08-01 → 2026-09-01',
    source: 'verification evidence',
    freshness: '2026-09-02T00:00:00Z',
    coverage: '4/5 tasks',
    comparison: 'previous n=3 → current n=4',
    state: 'live',
    state_detail: '',
    summary: 'Four of five tasks carry exact-tree evidence.',
    points: [
      { id: 'missing', label: '2026-08-01', value: null, display: 'Missing', evidence_count: 0 },
      {
        id: 'known',
        label: '2026-08-02',
        value: 80,
        display: '80%',
        href: '#/work?project=core&day=2026-08-02',
        evidence_count: 4,
      },
    ],
    ...over,
  }
}

describe('ChartFrame', () => {
  it('exposes the complete contract and a bounded evidence-linked table', () => {
    const wrapper = mount(ChartFrame, { props: { contract: contract() } })
    expect(wrapper.text()).toContain('Exact-tree acceptance by day')
    expect(wrapper.text()).toContain('previous n=3 → current n=4')
    expect(wrapper.text()).toContain('Missing')
    const link = wrapper.get('a')
    expect(link.attributes('href')).toBe('#/work?project=core&day=2026-08-02')
    expect(link.text()).toContain('4 record(s)')
    expect(wrapper.html()).not.toContain('>0<')
  })

  it.each(['stale', 'partial', 'error'] as const)(
    'names %s state in text, not color alone',
    (state) => {
      const wrapper = mount(ChartFrame, {
        props: { contract: contract({ state, state_detail: `${state} evidence detail` }) },
      })
      expect(wrapper.text()).toContain(state)
      expect(wrapper.get('[role="status"]').text()).toContain(`${state} evidence detail`)
    },
  )

  it('keeps the evidence fallback reachable at a 390px viewport', () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 390 })
    window.dispatchEvent(new Event('resize'))
    const wrapper = mount(ChartFrame, { props: { contract: contract() } })
    expect(wrapper.get('section').classes()).toContain('min-w-0')
    expect(wrapper.get('details div').classes()).toContain('overflow-auto')
    expect(wrapper.get('a').attributes('href')).toContain('day=2026-08-02')
  })
})
