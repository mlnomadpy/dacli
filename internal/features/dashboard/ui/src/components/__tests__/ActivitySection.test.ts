import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ActivitySection from '../ActivitySection.vue'
import type { ActivityResponse } from '@/types'

const response: ActivityResponse = {
  generated: '2026-09-01T20:00:00Z',
  task: '',
  limit: 2,
  next_cursor: '01OLDER',
  truncated: true,
  partial: true,
  unreadable_records: 1,
  filters: { project: 'core', state: 'pending', range: '7d' },
  events: [
    {
      id: '01EVENT2',
      kind: 'block',
      label: 'Policy refusal',
      category: 'refusal',
      actor: 'a-reviewer',
      about: 't-02',
      origin: 'file:<withheld-local-path>',
      against: 'a-builder',
      applied: false,
      at: '2026-09-01T19:00:00Z',
      body: '<img src="https://attacker.invalid/pixel"> refused by policy',
      related_task: 't-02',
      related_agent: 'a-reviewer',
    },
    {
      id: '01EVENT1',
      kind: 'review',
      label: 'Review verdict',
      category: 'review',
      actor: 'a-reviewer',
      about: 't-01',
      origin: 'agent',
      against: '',
      applied: true,
      at: '2026-09-01T18:00:00Z',
      body: 'approved exact tree',
      related_task: 't-01',
      related_agent: 'a-reviewer',
    },
  ],
}

const projects = [
  {
    slug: 'core',
    title: 'Core',
    stage: 'build',
    total: 2,
    counts: { open: 2 },
    burndown: { done_points: 0, remaining_points: 0, unestimated: 0, per_day: [] },
    graph: {
      project: 'core',
      nodes: [],
      edges: [],
      critical_path: [],
      duration: 0,
      scheduled: false,
      note: '',
      projection: {
        mode: '' as const,
        rule: '',
        statuses: [],
        page: 1,
        limit: 0,
        total_nodes: 0,
        visible_nodes: 0,
        hidden_nodes: 0,
        total_edges: 0,
        visible_edges: 0,
        hidden_edges: 0,
        critical_total: 0,
        has_more: false,
      },
    },
  },
]

describe('ActivitySection', () => {
  it('renders newest-first typed evidence and treats bodies as inert text', () => {
    const w = mount(ActivitySection, {
      props: {
        activity: response,
        phase: 'live',
        hasSnapshot: true,
        error: null,
        selection: { project: 'core', event_state: 'pending', range: '7d' },
        projects,
      },
    })
    expect(w.findAll('ol > li')).toHaveLength(2)
    expect(w.findAll('ol > li')[0].text()).toContain('Policy refusal')
    expect(w.text()).toContain('pending owner action')
    expect(w.text()).toContain('applied / journaled')
    expect(w.text()).toContain('Partial journal observation: 1 durable record')
    expect(w.html()).toContain('&lt;img src="https://attacker.invalid/pixel"&gt;')
    expect(w.find('img').exists()).toBe(false)
    expect(w.get('a[href="#/work?task=t-02"]').text()).toBe('t-02')
    expect(w.get('a[href="#/agents?agent=a-reviewer"]').text()).toBe('a-reviewer')
    expect(w.get('a[href="#/agents?agent=a-builder"]').text()).toBe('a-builder')
  })

  it('keeps filters and stable pagination URL-owned', async () => {
    const w = mount(ActivitySection, {
      props: {
        activity: response,
        phase: 'live',
        hasSnapshot: true,
        error: null,
        selection: { project: 'core', event_state: 'pending', range: '7d' },
        projects,
      },
    })
    const older = w.findAll('button').find((button) => button.text() === 'Older events')!
    await older.trigger('click')
    expect(w.emitted('change')?.[0]?.[0]).toEqual({
      project: 'core',
      event_state: 'pending',
      range: '7d',
      cursor: '01OLDER',
    })
    const state = w.findAll('select')[2]
    await state.setValue('applied')
    expect(w.emitted('change')?.[1]?.[0]).toEqual({
      project: 'core',
      event_state: 'applied',
      range: '7d',
    })
  })

  it('names empty, loading, and cold-error states without mutation controls', () => {
    const empty = mount(ActivitySection, {
      props: {
        activity: {
          ...response,
          partial: false,
          unreadable_records: 0,
          truncated: false,
          next_cursor: undefined,
          events: [],
        },
        phase: 'live',
        hasSnapshot: true,
        error: null,
        selection: {},
        projects,
      },
    })
    expect(empty.text()).toContain('No events match this observation')
    expect(empty.text()).toContain('No record was marked applied by this view')
    expect(empty.text()).not.toMatch(/\b(sync|dismiss|approve)\b.*button/i)

    const loading = mount(ActivitySection, {
      props: {
        activity: null,
        phase: 'loading',
        hasSnapshot: false,
        error: null,
        selection: {},
        projects,
      },
    })
    expect(loading.get('[role="status"]').text()).toContain('Loading durable activity')

    const failed = mount(ActivitySection, {
      props: {
        activity: null,
        phase: 'error',
        hasSnapshot: false,
        error: 'journal unreadable',
        selection: {},
        projects,
      },
    })
    expect(failed.get('[role="alert"]').text()).toContain('journal unreadable')

    const stale = mount(ActivitySection, {
      props: {
        activity: response,
        phase: 'error',
        hasSnapshot: true,
        error: 'refresh timed out',
        selection: {},
        projects,
      },
    })
    expect(stale.get('[role="alert"]').text()).toContain('Stale activity snapshot')
    expect(stale.get('[role="alert"]').text()).toContain('refresh timed out')
  })
})
