import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import OperatorPulse from '../OperatorPulse.vue'
import { emptyGraph } from '@/types'
import type { OverviewResponse, Project } from '@/types'

function project(over: Partial<Project> = {}): Project {
  return {
    slug: 'core',
    title: 'dacli',
    stage: 'build',
    total: 8,
    counts: { open: 4, active: 1, blocked: 2, done: 1 },
    burndown: { done_points: 3, remaining_points: 8, unestimated: 0, per_day: [] },
    graph: emptyGraph(),
    ...over,
  }
}

function overview(over: Partial<OverviewResponse> = {}): OverviewResponse {
  return {
    generated: '2026-08-31T20:00:00Z',
    project_count: 1,
    task_count: 8,
    counts: { open: 4, active: 1, blocked: 2, done: 1 },
    pending_events: 3,
    live_agents: 1,
    ...over,
  }
}

describe('OperatorPulse', () => {
  it('keeps global attention and a project handoff in the lightweight overview', () => {
    const w = mount(OperatorPulse, {
      props: { projects: [project()], overview: overview() },
    })

    expect(w.text()).toContain('Next work area')
    expect(w.text()).toContain('dacli')
    expect(w.text()).toContain('1 active · 4 open · 2 blocked')
    expect(w.text()).toContain('2 signals need attention')
    expect(w.text()).toContain('2 blocked tasks')
    expect(w.text()).toContain('3 pending events')
    expect(w.find('a[href="#/work?project=core"]').exists()).toBe(true)
    expect(w.find('a[href="#/activity?event_state=pending"]').exists()).toBe(true)
    expect(w.text()).toContain('1')
  })

  it('states the limited calm observation instead of implying every route is healthy', () => {
    const w = mount(OperatorPulse, {
      props: {
        projects: [project({ counts: { open: 4, active: 1, done: 3 } })],
        overview: overview({ counts: { open: 4, active: 1, done: 3 }, pending_events: 0 }),
      },
    })

    expect(w.text()).toContain('No recorded blockers')
    expect(w.text()).toContain('lightweight global signals are calm')
    expect(w.text()).toContain('Open a focused area for its own evidence')
  })
})
