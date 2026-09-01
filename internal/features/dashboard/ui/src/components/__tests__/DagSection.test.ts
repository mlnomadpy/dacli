import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DagSection from '../DagSection.vue'
import { emptyGraph } from '@/types'

const project = {
  slug: 'core',
  title: 'Core',
  stage: 'build',
  total: 553,
  counts: { open: 40, active: 8, blocked: 5, done: 500 },
  burndown: { done_points: 0, remaining_points: 0, unestimated: 0, per_day: [] },
  graph: {
    ...emptyGraph(),
    project: 'core',
    projection: {
      ...emptyGraph().projection,
      mode: 'operational' as const,
      rule: 'bounded operational view',
      total_nodes: 553,
      hidden_nodes: 553,
      limit: 120,
      has_more: true,
    },
  },
}

function wrapper(over: Partial<InstanceType<typeof DagSection>['$props']> = {}) {
  return mount(DagSection, {
    props: {
      projects: [project],
      selectedSlug: 'core',
      phase: 'live',
      hasSnapshot: true,
      error: null,
      graphMode: 'operational',
      graphStatuses: [],
      graphFocus: '',
      graphPage: 1,
      ...over,
    },
  })
}

describe('DagSection', () => {
  it('emits exact focus and server-side status filters', async () => {
    const w = wrapper()
    await w.get('#graph-focus').setValue('t-01FOCUS')
    await w.get('form').trigger('submit')
    expect(w.emitted('query')?.[0]?.[0]).toEqual({
      focus: 't-01FOCUS',
      mode: 'operational',
      page: 1,
    })
    const blocked = w.findAll('button').find((button) => button.text() === 'blocked')!
    await blocked.trigger('click')
    expect(w.emitted('query')?.[1]?.[0]).toEqual({ statuses: ['blocked'], focus: '', page: 1 })
  })

  it('makes completed history explicit and paginated', async () => {
    const w = wrapper({ graphMode: 'history', graphPage: 2 })
    expect(w.text()).toContain('Operational view')
    expect(w.text()).toContain('Page 2')
    const previous = w.findAll('button').find((button) => button.text() === 'Previous')!
    await previous.trigger('click')
    expect(w.emitted('query')?.[0]?.[0]).toEqual({ page: 1 })
  })
})
