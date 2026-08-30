import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import OperatorPulse from '../OperatorPulse.vue'
import { emptyBurn, emptyGraph } from '@/types'
import type { Agent, Project, Role } from '@/types'

function project(over: Partial<Project> = {}): Project {
  return {
    slug: 'core',
    title: 'dacli',
    stage: 'build',
    total: 8,
    counts: { open: 4, active: 1, blocked: 2, done: 1 },
    burndown: { done_points: 3, remaining_points: 8, unestimated: 0, per_day: [] },
    graph: {
      ...emptyGraph(),
      project: 'core',
      scheduled: true,
      duration: 8,
      critical_path: ['t-183', 't-184'],
      nodes: [
        {
          id: 't-183',
          seq: 183,
          slug: 'verification-route',
          title: 'Route verification by changed path',
          status: 'done',
          points: 2,
          estimated: true,
          critical: true,
          slack: 0,
          early_start: 0,
        },
        {
          id: 't-184',
          seq: 184,
          slug: 'review-gate',
          title: 'Bind review to the exact tree',
          status: 'open',
          points: 3,
          estimated: true,
          critical: true,
          slack: 0,
          early_start: 0,
        },
      ],
      edges: [],
    },
    ...over,
  }
}

function agent(over: Partial<Agent> = {}): Agent {
  return {
    run_id: '01RUN',
    child: 'a-review',
    task: '184',
    role: 'reviewer',
    runtime: 'codex',
    pid: 42,
    started: '2026-08-30T12:00:00Z',
    state: 'stalled',
    runtime_secs: 120,
    last_activity: '2026-08-30T12:01:00Z',
    transcript_url: '/api/agents/transcript?run=01RUN',
    diff_url: '/api/agents/diff?run=01RUN',
    ...over,
  }
}

function role(over: Partial<Role> = {}): Role {
  return {
    name: 'reviewer',
    summary: 'reviews exact trees',
    kind: 'reviewer',
    grant: 'ro',
    runtime: 'codex',
    model: 'gpt-5.6-terra',
    wip: 1,
    max_points: 5,
    scope: ['**'],
    out_of_scope: [],
    skills: [],
    shortcuts: [],
    escalate_to: ['maintainer'],
    active_agents: 1,
    wip_exceeded: true,
    has_prompt: true,
    ...over,
  }
}

describe('OperatorPulse', () => {
  it('puts critical focus and every existing attention class in the first view', () => {
    const burn = { ...emptyBurn(), ceiling: 100, rate: 180, ratio: 1.8, alert_at: 1.5, alert: true }
    const w = mount(OperatorPulse, {
      props: {
        projects: [project()],
        agents: [agent()],
        roles: [role()],
        burn,
        pendingEvents: 3,
      },
    })

    expect(w.text()).toContain('Next on the critical path')
    expect(w.text()).toContain('#184 · Bind review to the exact tree')
    expect(w.text()).toContain('Recorded path')
    expect(w.text()).toContain('5 signals need attention')
    expect(w.text()).toContain('Burn is 1.8× the calibrated ceiling')
    expect(w.text()).toContain('2 blocked tasks')
    expect(w.text()).toContain('1 unhealthy agent')
    expect(w.text()).toContain('3 pending events')
    expect(w.text()).toContain('1 role at its WIP cap')
    expect(w.find('a[href="#delivery"]').exists()).toBe(true)
    expect(w.find('a[href="#agents"]').exists()).toBe(true)
    expect(w.find('a[href="#team"]').exists()).toBe(true)
  })

  it('states the calm observed condition instead of leaving attention empty', () => {
    const w = mount(OperatorPulse, {
      props: {
        projects: [project({ counts: { open: 4, active: 1, done: 3 } })],
        agents: [agent({ state: 'acting' })],
        roles: [role({ wip_exceeded: false })],
        burn: emptyBurn(),
        pendingEvents: 0,
      },
    })

    expect(w.text()).toContain('No recorded blockers')
    expect(w.text()).toContain('Observed signals are within policy')
  })
})
