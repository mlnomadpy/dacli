import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AttentionQueue from '../AttentionQueue.vue'
import type { OperatorAttentionResponse } from '@/types'

function response(): OperatorAttentionResponse {
  return {
    schema: 'operator-attention/v1',
    generated: '2026-09-02T12:00:00Z',
    alerts: [
      {
        id: 'github_state_unknown/core/task-1//',
        code: 'github_state_unknown',
        severity: 'critical',
        affected: { project: 'core', task: 'task-1', pr: 'https://github.test/pull/1' },
        first_observed: '2026-09-02T09:00:00Z',
        last_observed: '2026-09-02T11:00:00Z',
        freshness: 'stale',
        retryable: true,
        summary: 'GitHub state is unknown; absence is not healthy.',
        next_action: 'Re-observe the exact head after GitHub recovers.',
        link: '#/delivery?project=core&task=task-1',
        evidence: [
          {
            kind: 'github-check',
            id: 'check-99',
            url: 'https://github.test/check/99',
            observed_at: '2026-09-02T11:00:00Z',
            confidence: 'low',
          },
        ],
        occurrences: 3,
        duration_seconds: 7200,
        critical_path: true,
        confidence: 'low',
        rank: 1,
        rank_reason: 'severity=critical; critical_path=true; age=3h; confidence=low',
      },
    ],
    ranking_rule: 'severity, critical path, age, confidence, stable identity',
  }
}

describe('AttentionQueue', () => {
  it('renders ranked policy evidence and exact drill-downs without color-only meaning', () => {
    const wrapper = mount(AttentionQueue, {
      props: { attention: response(), phase: 'live', hasSnapshot: true, error: null },
    })
    expect(wrapper.text()).toContain('critical severity')
    expect(wrapper.text()).toContain('critical path')
    expect(wrapper.text()).toContain('stale evidence')
    expect(wrapper.text()).toContain('Occurrences3')
    expect(wrapper.text()).toContain('Why this is rank 1')
    expect(wrapper.get('a[href="https://github.test/check/99"]').text()).toContain('check-99')
    expect(wrapper.get('a[href="#/delivery?project=core&task=task-1"]').text()).toContain(
      'Inspect exact evidence',
    )
    expect(wrapper.get('ol').attributes('aria-label')).toBe('Prioritized operator alerts')
  })

  it('keeps the last snapshot explicitly stale after refresh failure', () => {
    const wrapper = mount(AttentionQueue, {
      props: { attention: response(), phase: 'error', hasSnapshot: true, error: 'HTTP 503' },
    })
    expect(wrapper.get('[role="status"]').text()).toContain('last successful queue snapshot')
    expect(wrapper.text()).toContain('HTTP 503')
    expect(wrapper.text()).toContain('github state unknown')
  })

  it('is responsive at 390px and preserves the same keyboard links', () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 390 })
    const wrapper = mount(AttentionQueue, {
      props: { attention: response(), phase: 'live', hasSnapshot: true, error: null },
    })
    expect(wrapper.get('ol').classes()).toContain('grid')
    expect(wrapper.findAll('a')).toHaveLength(2)
  })

  it('does not claim missing external state is healthy when the queue is empty', () => {
    const empty = response()
    empty.alerts = []
    const wrapper = mount(AttentionQueue, {
      props: { attention: empty, phase: 'live', hasSnapshot: true, error: null },
    })
    expect(wrapper.text()).toContain('current-state observation')
    expect(wrapper.text()).toContain('not a claim that missing external systems are healthy')
  })
})
