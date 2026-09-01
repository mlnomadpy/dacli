import { describe, expect, it } from 'vitest'
import {
  emptyBurn,
  emptyGraph,
  type Agent,
  type Project,
  type Role,
  type TaskSummary,
} from '@/types'
import {
  filterAgents,
  filterBurn,
  filterProjects,
  filterRoles,
  filterTasks,
  inactiveFilters,
} from '../useObservabilityFilters'

const now = Date.parse('2026-09-01T12:00:00Z')
const agents: Agent[] = [
  {
    run_id: '01CODEX',
    child: 'a-ui',
    task: '950',
    role: 'frontend-engineer',
    runtime: 'codex',
    pid: 1,
    started: '2026-09-01T11:00:00Z',
    state: 'acting',
    runtime_secs: 60,
    last_activity: '2026-09-01T11:59:00Z',
    transcript_url: '/transcript',
    diff_url: '/diff',
  },
  {
    run_id: '01CLAUDE',
    child: 'a-api',
    task: '949',
    role: 'backend-engineer',
    runtime: 'claude',
    pid: 2,
    started: '2026-08-20T11:00:00Z',
    state: 'blocked',
    runtime_secs: 60,
    last_activity: '2026-08-20T11:59:00Z',
    transcript_url: '/transcript',
    diff_url: '/diff',
  },
]
const roles: Role[] = [
  {
    name: 'frontend-engineer',
    summary: 'Builds the dashboard',
    kind: 'implementer',
    grant: 'rw',
    runtime: 'codex',
    model: 'gpt-5.6',
    wip: 1,
    max_points: 8,
    scope: ['internal/features/dashboard/**'],
    out_of_scope: [],
    skills: ['frontend'],
    shortcuts: [],
    escalate_to: [],
    active_agents: 1,
    wip_exceeded: false,
    has_prompt: true,
  },
]

describe('observability filters', () => {
  it('filters complete agent and role observations without substituting identities', () => {
    expect(
      filterAgents(
        agents,
        { q: '950', filter_role: 'frontend-engineer', runtime: 'codex', range: '24h' },
        now,
      ).map((agent) => agent.run_id),
    ).toEqual(['01CODEX'])
    expect(filterAgents(agents, { state: 'blocked', range: '24h' }, now)).toEqual([])
    expect(filterRoles(roles, { model: 'gpt-5.6', q: 'dashboard' })).toEqual(roles)
    expect(filterRoles(roles, { filter_role: 'missing' })).toEqual([])
  })

  it('filters projects exactly and time-bounds burn without treating missing days as zero', () => {
    const projects: Project[] = [
      {
        slug: 'core',
        title: 'Core',
        stage: 'build',
        total: 1,
        counts: { open: 1 },
        burndown: { done_points: 0, remaining_points: 1, unestimated: 0, per_day: [] },
        graph: emptyGraph(),
      },
    ]
    expect(filterProjects(projects, { project: 'missing' })).toEqual([])
    const burn = {
      ...emptyBurn(),
      series: [
        { day: '2026-08-01', tokens: 1, cost_usd: 1, runs: 1, per_run: 1 },
        { day: '2026-09-01', tokens: 2, cost_usd: 2, runs: 1, per_run: 2 },
      ],
    }
    expect(filterBurn(burn, { range: '7d' }, now).series.map((point) => point.day)).toEqual([
      '2026-09-01',
    ])
  })

  it('searches complete selected-project task rows across identity and operational fields', () => {
    const tasks: TaskSummary[] = [
      {
        id: 't-01TASK935',
        project: 'core',
        seq: 935,
        slug: 'task-explorer',
        title: 'Build task explorer',
        status: 'blocked',
        priority: 'critical',
        owner: 'a-builder',
        points: 0,
        estimated: false,
      },
    ]
    for (const query of [
      '01TASK935',
      'explorer',
      'a-builder',
      'critical',
      'blocked',
      'unestimated',
    ])
      expect(filterTasks(tasks, query)).toEqual(tasks)
    expect(filterTasks(tasks, 'missing')).toEqual([])
  })

  it('retains unsupported route filters as explicit inactive context', () => {
    expect(inactiveFilters('delivery', { project: 'core', runtime: 'codex', q: '950' })).toEqual([
      'runtime',
      'q',
    ])
    expect(inactiveFilters('work', { project: 'core', q: '935' })).toEqual([])
    expect(
      inactiveFilters('activity', {
        project: 'core',
        kind: 'finding',
        actor: 'a-reviewer',
        event_state: 'pending',
        range: '24h',
        cursor: '01KCURSOR',
      }),
    ).toEqual([])
  })
})
