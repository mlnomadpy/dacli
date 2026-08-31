import { afterEach, describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import App from '@/App.vue'
import { emptyGraph } from '@/types'
import type { DashboardState } from '@/types'
import screenshotFixture from './fixtures/dashboard-state.json'

// End-to-end wiring test: mount the whole App, let the store's poll loop pull a
// stubbed /api/state snapshot, and assert every surface renders live from it —
// Overview cards, the four-column board, the per-day burndown, the swarm
// table, and the team roster. This exercises the real component tree App →
// sections → leaves, the thing the unit tests each cover in isolation.

const SNAPSHOT: DashboardState = {
  generated: '2026-07-23T16:10:00Z',
  pending_events: 2,
  projects: [
    {
      slug: 'core',
      title: 'dacli remaining backlog',
      stage: 'build',
      total: 42,
      counts: { open: 12, active: 3, blocked: 1, done: 26 },
      burndown: {
        done_points: 88.5,
        remaining_points: 31,
        unestimated: 4,
        per_day: [
          { day: '2026-07-20', points: 12 },
          { day: '2026-07-21', points: 8.5 },
        ],
      },
      graph: emptyGraph(),
    },
  ],
  agents: [
    {
      run_id: '01KY8KW3W1GSP57K39ZY77NH6S',
      child: 'a-nhkth9j71n',
      task: '131',
      role: 'designer',
      runtime: 'claude',
      pid: 48213,
      started: '2026-07-23T16:00:00Z',
      state: 'thinking',
      runtime_secs: 600,
      last_activity: new Date().toISOString(),
      transcript_url: '/api/agents/transcript?run=01KY8KW3W1GSP57K39ZY77NH6S',
      diff_url: '/api/agents/diff?run=01KY8KW3W1GSP57K39ZY77NH6S',
    },
  ],
  roles: [
    {
      name: 'builder',
      summary: 'writes the code',
      kind: 'implementer',
      grant: 'rw',
      runtime: 'claude',
      model: 'sonnet',
      wip: 1,
      max_points: 5,
      scope: ['internal/**'],
      out_of_scope: [],
      skills: ['go'],
      shortcuts: [],
      escalate_to: ['maintainer'],
      active_agents: 1,
      wip_exceeded: true,
      has_prompt: true,
    },
  ],
  burn: {
    unit: 'output_tokens',
    ceiling: 100,
    rate: 500,
    ratio: 5,
    alert: true,
    alert_at: 1.5,
    windows: [{ project: 'core', spent: 4200, start: '2026-07-24T00:00:00Z' }],
    bands: [
      { band: 'builder/opus/claude', role: 'builder', expected: 100, n: 1, calibrated: false },
    ],
    series: [
      { day: '2026-07-20', tokens: 100, cost_usd: 0.1, runs: 1, per_run: 100 },
      { day: '2026-07-24', tokens: 500, cost_usd: 0.6, runs: 1, per_run: 500 },
    ],
  },
}

function appFetch(snapshot: DashboardState): typeof fetch {
  return vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input)
    let body: unknown
    switch (true) {
      case url === '/api/overview':
        body = {
          generated: snapshot.generated,
          project_count: snapshot.projects.length,
          task_count: snapshot.projects.reduce((total, project) => total + project.total, 0),
          counts: snapshot.projects.reduce<Record<string, number>>((counts, project) => {
            for (const [status, count] of Object.entries(project.counts)) {
              counts[status] = (counts[status] ?? 0) + (count ?? 0)
            }
            return counts
          }, {}),
          pending_events: snapshot.pending_events,
          live_agents: snapshot.agents.length,
        }
        break
      case url === '/api/projects':
        body = { generated: snapshot.generated, projects: snapshot.projects }
        break
      case url === '/api/agents':
        body = { generated: snapshot.generated, agents: snapshot.agents }
        break
      case url === '/api/roles':
        body = { generated: snapshot.generated, roles: snapshot.roles }
        break
      case url === '/api/burn':
        body = { generated: snapshot.generated, ...snapshot.burn }
        break
      case url.startsWith('/api/graph?project='):
        const projectSlug = decodeURIComponent(url.split('=')[1])
        body = {
          generated: snapshot.generated,
          ...(snapshot.projects.find((project) => project.slug === projectSlug)?.graph ??
            emptyGraph()),
          project: projectSlug,
        }
        break
      default:
        return new Response('not found', { status: 404 })
    }
    return new Response(JSON.stringify(body), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  }) as unknown as typeof fetch
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('App (end-to-end)', () => {
  it('polls a snapshot and renders all three surfaces live', async () => {
    vi.stubGlobal('fetch', appFetch(SNAPSHOT))

    const w = mount(App, { global: { plugins: [createPinia()] } })
    await flushPromises()

    // Header connection tell flipped to live + pending pill.
    expect(w.text()).toContain('live · updated')
    expect(w.text()).toContain('2 pending events')

    // Overview: the project card with counts + burndown caption.
    expect(w.text()).toContain('dacli remaining backlog')
    expect(w.text()).toContain('88.5 done · 31.0 remaining pts · 4 unestimated')

    // Board: four columns with the project's true counts.
    const cols = w.findAll('[role="group"]')
    expect(cols).toHaveLength(4)
    expect(cols[0].attributes('aria-label')).toBe('open — 12 tasks')
    expect(cols[3].attributes('aria-label')).toBe('done — 26 tasks')

    // Burndown chart: one bar per day, server order preserved. (The burn chart
    // also uses .chart .bar, so scope this assertion to the burndown chart.)
    expect(w.findAll('.burndown .chart .bar')).toHaveLength(2)

    // Burn surface: wired in, live, and YELLING — the rate is 5× the ceiling.
    expect(w.find('#burn-h').exists()).toBe(true)
    expect(w.find('[role="alert"]').text()).toContain('5.0× the calibrated ceiling')

    // Swarm: a real table row for the live agent, newest-first.
    expect(w.find('table').exists()).toBe(true)
    expect(w.text()).toContain('01KY8KW3W1')
    expect(w.text()).toContain('1 running')

    // Roster (dacli 226): the team the same poll carries, with the WIP cap the
    // operator would otherwise have had to infer from .dacli by hand.
    expect(w.find('#roster-h').exists()).toBe(true)
    expect(w.text()).toContain('claude / sonnet')
    expect(w.find('.capped').text()).toContain('1 at WIP cap')

    w.unmount()
  })

  it('cold error before any snapshot shows section error panels with Retry', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new Error('network down')
      }),
    )

    const w = mount(App, { global: { plugins: [createPinia()] } })
    await flushPromises()

    expect(w.text()).toContain('connection lost: overview: network down')
    // No snapshot ever → surfaces show their own cold error, each with Retry.
    const alerts = w.findAll('[role="alert"]')
    expect(alerts.length).toBeGreaterThanOrEqual(1)
    expect(w.find('button.retry').exists()).toBe(true)

    w.unmount()
  })

  it('renders the representative screenshot fixture through the real operator hierarchy', async () => {
    vi.stubGlobal('fetch', appFetch(screenshotFixture as DashboardState))

    const w = mount(App, { global: { plugins: [createPinia()] } })
    await flushPromises()

    expect(w.find('#operator-pulse-h').exists()).toBe(true)
    expect(w.text()).toContain('#901 · Surface operator attention')
    expect(w.text()).toContain('4 signals need attention')
    expect(w.find('nav[aria-label="Dashboard sections"]').exists()).toBe(true)
    for (const id of ['#pulse', '#delivery', '#agents', '#team']) {
      expect(w.find(id).exists()).toBe(true)
    }
    expect(w.findAll('[role="group"]')).toHaveLength(4)

    w.unmount()
  })
})
