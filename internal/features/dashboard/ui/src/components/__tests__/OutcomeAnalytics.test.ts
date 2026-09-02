import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import OutcomeAnalytics from '../OutcomeAnalytics.vue'
import type { OutcomeAnalyticsResponse, OutcomeMeasure, OutcomeMetric } from '@/types'

function measure(
  key: string,
  value: number | null,
  state: OutcomeMeasure['state'] = 'complete',
): OutcomeMeasure {
  return {
    key,
    label: key,
    value,
    unit: key === 'cost' ? 'USD' : 'tasks',
    sample_size: value === null ? 0 : 4,
    eligible: 5,
    coverage: value === null ? 0 : 0.8,
    state,
    provenance: 'exact durable records',
    evidence: { tasks: ['01TASK'], runs: ['01RUN'], truncated: false },
  }
}

function metric(
  key: string,
  value: number | null,
  state: OutcomeMeasure['state'] = 'complete',
): OutcomeMetric {
  return {
    key,
    label: key.replace(/_/g, ' '),
    current: measure(key, value, state),
    previous: measure(key, value === null ? null : value - 1),
    change: value === null ? null : 1,
    trend: value === null ? 'not-comparable' : 'up',
  }
}

function analytics(): OutcomeAnalyticsResponse {
  return {
    schema: 'outcome-analytics/v1',
    generated: '2026-09-02T12:00:00Z',
    project: 'core',
    current_window: { start: '2026-08-03T12:00:00Z', end: '2026-09-02T12:00:00Z', days: 30 },
    previous_window: { start: '2026-07-04T12:00:00Z', end: '2026-08-03T12:00:00Z', days: 30 },
    metrics: [
      metric('throughput', 4),
      metric('execution_time', 2.5, 'partial'),
      metric('current_tree_acceptance', 80),
      metric('first_pass_review', null, 'unknown'),
      metric('retry_rate', 1),
      metric('cost', 0.25, 'partial'),
    ],
    breakdowns: [
      {
        dimension: 'task_size',
        key: 'small',
        size_band: 'small',
        current: measure('throughput', 4),
        previous: measure('throughput', 3),
        comparable: true,
        evidence: { tasks: ['01TASK'], runs: ['01RUN'], truncated: false },
      },
    ],
    series: [
      {
        day: '2026-09-01',
        completed: 2,
        runs: 4,
        tokens: 200,
        evidence: { tasks: ['01TASK'], runs: ['01RUN'], truncated: false },
      },
    ],
    performance: {
      tasks_scanned: 5,
      runs_scanned: 7,
      series_points: 1,
      build_ms: 3,
      evidence_cap: 100,
      cache: 'fresh-index',
      cache_entries: 1,
    },
    notes: ['descriptive', 'provider-reported, not billing'],
  }
}

describe('OutcomeAnalytics', () => {
  it('renders coverage and unknowns without converting missing evidence to zero', async () => {
    const wrapper = mount(OutcomeAnalytics, {
      props: { analytics: analytics(), range: 30, stale: true, focusDay: '2026-09-01' },
    })
    expect(wrapper.text()).toContain('Is the delivery system improving?')
    expect(wrapper.text()).toContain('Unknown')
    expect(wrapper.text()).toContain('provider-reported, not billing')
    expect(wrapper.text()).toContain('Stale analytics snapshot')
    expect(wrapper.text()).not.toContain('NaN')
    const metricLink = wrapper.findAll('a').find((button) => button.text().includes('throughput'))!
    expect(metricLink.attributes('href')).toBe(
      '#/agents?project=core&outcome_range=30d&metric=throughput',
    )
    await metricLink.trigger('click')
    expect(wrapper.text()).toContain('task 01TASK')
    expect(wrapper.text()).toContain('run 01RUN')
    expect(wrapper.text()).toContain('2026-09-01 exact evidence')
  })

  it('emits bounded window changes', async () => {
    const wrapper = mount(OutcomeAnalytics, { props: { analytics: analytics(), range: 30 } })
    await wrapper
      .findAll('button')
      .find((button) => button.text() === '7d')!
      .trigger('click')
    expect(wrapper.emitted('range')).toEqual([[7]])
  })
})
