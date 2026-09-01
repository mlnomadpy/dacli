import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DeliveryTimeline from '../DeliveryTimeline.vue'
import type { DeliveryTimelineResponse, GraphNode } from '@/types'

const tasks: GraphNode[] = [
  {
    id: 't-952',
    seq: 952,
    slug: 'delivery-waterfall',
    title: 'Delivery waterfall',
    status: 'active',
    points: 5,
    estimated: true,
    critical: true,
    slack: 0,
    early_start: 0,
  },
]

const timeline: DeliveryTimelineResponse = {
  schema: 'delivery-attempt-timeline/v1',
  generated: '2026-09-01T00:05:00Z',
  task: {
    id: 't-952',
    sequence: 952,
    generation: 2,
    project: 'core',
    title: 'Delivery waterfall',
    status: 'active',
  },
  summary: 'Attempt 2 is running at CI.',
  attempts: [
    {
      attempt: 2,
      run_id: '01RUN',
      agent_id: 'a-worker',
      role: 'frontend-engineer',
      runtime: 'codex',
      model: 'gpt-5.6-terra',
      generation: 2,
      started: '2026-09-01T00:00:00Z',
      outcome: 'running',
      recovered: true,
      usage: { available: true, input_tokens: 10, output_tokens: 25, turns: 3, cost_usd: 0.125 },
      identity: {
        task_id: 't-952',
        run_id: '01RUN',
        commit_sha: 'commit',
        tree_sha: 'tree',
        pr_url: 'https://github.com/mlnomadpy/dacli/pull/1',
        pr_generation: 2,
      },
      pull_requests: [
        { url: 'https://github.com/mlnomadpy/dacli/pull/0', generation: 1, state: 'superseded' },
        { url: 'https://github.com/mlnomadpy/dacli/pull/1', generation: 2, state: 'current' },
      ],
      spans: [
        {
          phase: 'verified',
          status: 'complete',
          duration_ms: 4_000,
          source: 'verification evidence',
          freshness: 'current task generation',
          detail: 'Verification passed.',
          next_action: 'inspect review',
          contract: 'go test ./...',
          verdict: 'approve',
          correction: 1,
        },
        {
          phase: 'ci',
          status: 'current',
          duration_ms: null,
          source: 'loop phase journal',
          freshness: 'current task generation',
          detail: 'Checks are pending.',
          next_action: 'wait for checks',
        },
        {
          phase: 'merged',
          status: 'pending',
          duration_ms: null,
          source: 'loop phase journal',
          freshness: 'current task generation',
          detail: 'No merge observed.',
          next_action: 'wait for merge',
        },
      ],
    },
  ],
}

describe('DeliveryTimeline', () => {
  it('renders exact identities, honest unknown duration, mobile fallback, and safe deep links', () => {
    const wrapper = mount(DeliveryTimeline, {
      props: { tasks, selectedTask: 't-952', timeline },
    })
    const text = wrapper.text()
    for (const fact of [
      'Attempt 2',
      'frontend-engineer',
      'codex / gpt-5.6-terra',
      '25 output tokens',
      'unknown duration',
      'Recovered',
      'tree tree',
      'Attempt 2 is running at CI.',
      'approve · correction 1',
      'g1 · superseded',
      'g2 · current',
    ]) {
      expect(text).toContain(fact)
    }
    expect(wrapper.find('ol[aria-label="Delivery phases"]').exists()).toBe(true)
    expect(wrapper.find('ol[aria-label="Delivery phases mobile"]').exists()).toBe(true)
    expect(wrapper.find('a[href*="task=t-952"]').exists()).toBe(true)
    expect(wrapper.html()).not.toContain('transcript')
    expect(wrapper.html()).not.toContain('prompt')
  })

  it('emits the exact selected task and exposes exact phase evidence in tooltips', async () => {
    const wrapper = mount(DeliveryTimeline, {
      props: { tasks, selectedTask: '', timeline: null },
    })
    await wrapper.find('select').setValue('t-952')
    expect(wrapper.emitted('select')).toEqual([['t-952']])

    await wrapper.setProps({ selectedTask: 't-952', timeline })
    const tooltip = wrapper.find('button[title*="Source: loop phase journal"]')
    expect(tooltip.attributes('title')).toContain('Next: wait for checks')
    expect(tooltip.attributes('title')).toContain('Duration: unknown duration')
  })

  it('keeps refused evidence visibly non-green', () => {
    const refused = { ...timeline, refusal: 'phase journal is corrupt' }
    const wrapper = mount(DeliveryTimeline, {
      props: { tasks, selectedTask: 't-952', timeline: refused },
    })
    expect(wrapper.get('[role="alert"]').text()).toContain('Evidence refused')
    expect(wrapper.get('[role="alert"]').text()).toContain('phase journal is corrupt')
  })
})
