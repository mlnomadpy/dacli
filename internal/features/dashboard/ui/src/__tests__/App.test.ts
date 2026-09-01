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

const TASK_ROWS = [
  ['t-01TASK935', 935, 'task-explorer', 'Build task explorer', 'open'],
  ['t-01TASK934', 934, 'agent-inspector', 'Inspect agent history', 'active'],
  ['t-01TASK936', 936, 'bounded-graph', 'Bound the dependency graph', 'blocked'],
  ['t-01TASK932', 932, 'surface-polling', 'Split surface polling', 'done'],
].map(([id, seq, slug, title, status]) => ({
  id,
  project: 'core',
  seq,
  slug,
  title,
  status,
  priority: status === 'blocked' ? 'critical' : 'high',
  owner: status === 'done' ? 'a-owner' : '',
  points: status === 'blocked' ? 0 : 3,
  estimated: status !== 'blocked',
}))

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
      case url.startsWith('/api/tasks?project='):
        body = { generated: snapshot.generated, tasks: TASK_ROWS }
        break
      case url.startsWith('/api/task?ref='):
        const taskRef = decodeURIComponent(url.split('=')[1])
        const task = TASK_ROWS.find((candidate) => candidate.id === taskRef)
        if (!task) return new Response('not found', { status: 404 })
        body = {
          generated: snapshot.generated,
          task: {
            ...task,
            estimate: task.estimated
              ? { optimistic: 2, probable: 3, pessimistic: 5, expected: 3.17 }
              : null,
            so_that: 'operators can inspect exact work',
            context: 'count-only boards hide identity',
            acceptance: [{ text: 'show exact identity', done: false }],
            acceptance_done: 0,
            acceptance_total: 1,
            deps: [],
            parent: '',
            log: [],
          },
        }
        break
      case url.startsWith('/api/events?task='):
        const eventRef = decodeURIComponent(url.split('=')[1])
        body = {
          generated: snapshot.generated,
          task: eventRef,
          limit: 50,
          truncated: false,
          events: [],
        }
        break
      case url.startsWith('/api/events?state='):
        body = {
          generated: snapshot.generated,
          limit: 50,
          truncated: false,
          partial: false,
          unreadable_records: 0,
          filters: {
            project: 'core',
            kind: 'finding',
            actor: 'a-reviewer',
            state: 'pending',
            range: '24h',
          },
          events: [
            {
              id: '01ACTIVITY937',
              kind: 'finding',
              label: 'Review finding',
              category: 'finding',
              actor: 'a-reviewer',
              about: 't-01TASK935',
              origin: 'agent',
              against: '',
              applied: false,
              at: snapshot.generated,
              body: '<img src="https://attacker.invalid/pixel"> needs owner attention',
              related_task: 't-01TASK935',
              related_agent: 'a-reviewer',
            },
          ],
        }
        break
      case url === '/api/agents':
        body = { generated: snapshot.generated, agents: snapshot.agents }
        break
      case url.startsWith('/api/agent?id='):
        body = {
          generated: snapshot.generated,
          agent: {
            id: decodeURIComponent(url.split('=')[1]),
            role: snapshot.agents[0]?.role ?? 'builder',
            parent: 'a-root',
            grant: 'rw',
            retired: false,
            children: ['a-descendant'],
            tasks: [],
            runs: snapshot.agents.map((agent) => ({
              run_id: agent.run_id,
              task: agent.task,
              role: agent.role,
              runtime: agent.runtime,
              pid: agent.pid,
              started: agent.started,
              live: true,
              transcript_url: agent.transcript_url,
              diff_url: agent.diff_url,
            })),
          },
        }
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
  window.history.replaceState(null, '', '/')
})

describe('App (end-to-end)', () => {
  it('lazy-mounts and lazy-fetches each routed workspace surface', async () => {
    const fetchImpl = appFetch(SNAPSHOT)
    vi.stubGlobal('fetch', fetchImpl)

    const w = mount(App, { attachTo: document.body, global: { plugins: [createPinia()] } })
    await flushPromises()

    // Header connection tell flipped to live + pending pill.
    expect(w.text()).toContain('live · updated')
    expect(w.text()).toContain('2 pending events')

    // Overview: the project card with counts + burndown caption.
    expect(w.text()).toContain('dacli remaining backlog')
    expect(w.text()).toContain('88.5 done · 31.0 remaining pts · 4 unestimated')
    expect(w.find('#burn-h').exists()).toBe(false)
    expect(w.find('#roster-h').exists()).toBe(false)
    expect(w.findAll('[role="group"]')).toHaveLength(0)
    let urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(new Set(urls)).toEqual(new Set(['/api/overview', '/api/projects']))

    vi.mocked(fetchImpl).mockClear()
    window.location.hash = '#/work?project=core'
    window.dispatchEvent(new HashChangeEvent('hashchange'))
    await flushPromises()

    // Board: four columns with the project's true counts.
    const cols = w.findAll('[role="group"]')
    expect(cols).toHaveLength(4)
    expect(cols[0].attributes('aria-label')).toBe('open — 12 tasks')
    expect(cols[3].attributes('aria-label')).toBe('done — 26 tasks')
    expect(w.text()).toContain('Build task explorer')
    expect(w.text()).toContain('Bound the dependency graph')
    expect(w.text()).toContain('unestimated')
    expect(document.activeElement?.id).toBe('route-heading')
    urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(urls).not.toContain('/api/agents')
    expect(urls).not.toContain('/api/burn')
    expect(urls).not.toContain('/api/roles')
    expect(urls).not.toContain('/api/graph?project=core')
    expect(urls).toContain('/api/tasks?project=core')

    vi.mocked(fetchImpl).mockClear()
    window.location.hash = '#/agents'
    window.dispatchEvent(new HashChangeEvent('hashchange'))
    await flushPromises()
    expect(w.find('#burn-h').exists()).toBe(true)
    expect(w.find('[role="alert"]').text()).toContain('5.0× the calibrated ceiling')
    expect(w.find('table').exists()).toBe(true)
    expect(w.text()).toContain('01KY8KW3W1')
    expect(w.text()).toContain('1 running')
    urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(urls).toContain('/api/agents')
    expect(urls).toContain('/api/burn')
    expect(urls).not.toContain('/api/roles')
    expect(urls).not.toContain('/api/graph?project=core')

    vi.mocked(fetchImpl).mockClear()
    window.location.hash = '#/team'
    window.dispatchEvent(new HashChangeEvent('hashchange'))
    await flushPromises()
    expect(w.find('#roster-h').exists()).toBe(true)
    expect(w.text()).toContain('claude / sonnet')
    expect(w.find('.capped').text()).toContain('1 at WIP cap')
    urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(urls).toContain('/api/roles')
    // Team combines the durable role roster with canonical live occupancy.
    // Since Agents was already observed above, route activation may reuse its
    // snapshot instead of immediately fetching it again.
    expect(urls).not.toContain('/api/burn')

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
    expect(w.text()).toContain('Agent delivery control plane')
    expect(w.text()).toContain('2 signals need attention')
    expect(w.find('nav[aria-label="Workspace areas"]').exists()).toBe(true)
    expect(w.findAll('nav[aria-label="Workspace areas"] a')).toHaveLength(6)
    expect(w.find('a[aria-current="page"]').text()).toContain('Overview')
    expect(w.find('.dashboard-workspace > nav[aria-label="Workspace areas"]').exists()).toBe(true)
    expect(w.get('.app-header').classes()).toContain('min-h-14')
    expect(w.get('.route-intro').classes()).toContain('min-h-11')
    expect(w.get('.route-intro').text()).not.toContain('read-only')
    expect(w.findAll('[role="group"]')).toHaveLength(0)

    w.unmount()
  })

  it('rejects unknown paths and unsafe identities without mounting hidden routes', async () => {
    window.history.replaceState(null, '', '/#/missing?project=../../etc/passwd')
    const fetchImpl = appFetch(SNAPSHOT)
    vi.stubGlobal('fetch', fetchImpl)

    const w = mount(App, { global: { plugins: [createPinia()] } })
    await flushPromises()

    expect(w.text()).toContain('This dashboard route does not exist')
    expect(w.text()).toContain('/missing')
    const urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(new Set(urls)).toEqual(new Set(['/api/overview']))
    expect(w.find('#roster-h').exists()).toBe(false)
    expect(w.find('#burn-h').exists()).toBe(false)
    expect(w.findAll('[role="group"]')).toHaveLength(0)

    w.unmount()
  })

  it('reopens an exact agent deep link and lazy-loads only that identity', async () => {
    const id = SNAPSHOT.agents[0].child
    window.history.replaceState(null, '', `/#/agents?agent=${id}`)
    const fetchImpl = appFetch(SNAPSHOT)
    vi.stubGlobal('fetch', fetchImpl)

    const wrapper = mount(App, { attachTo: document.body, global: { plugins: [createPinia()] } })
    await flushPromises()

    const dialog = document.querySelector<HTMLElement>('[role="dialog"]')
    expect(dialog?.textContent).toContain(id)
    expect(dialog?.textContent).toContain('a-root')
    expect(dialog?.textContent).toContain('Run ledger')
    const urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(urls.filter((url) => url === `/api/agent?id=${id}`)).toHaveLength(1)
    expect(urls).not.toContain('/api/roles')

    wrapper.unmount()
  })

  it('reopens an exact task deep link without eager per-row detail requests', async () => {
    window.history.replaceState(null, '', '/#/work?project=core&task=t-01TASK935')
    const fetchImpl = appFetch(SNAPSHOT)
    vi.stubGlobal('fetch', fetchImpl)

    const wrapper = mount(App, {
      attachTo: document.body,
      global: { plugins: [createPinia()] },
    })
    await flushPromises()

    const dialog = document.querySelector<HTMLElement>('[role="dialog"]')
    expect(dialog?.textContent).toContain('Build task explorer')
    expect(dialog?.textContent).toContain('show exact identity')
    const urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(urls.filter((url) => url === '/api/task?ref=t-01TASK935')).toHaveLength(1)
    expect(urls.filter((url) => url === '/api/events?task=t-01TASK935')).toHaveLength(1)
    expect(urls).toContain('/api/tasks?project=core')
    expect(urls).not.toContain('/api/graph?project=core')
    expect(urls.filter((url) => url.startsWith('/api/task?ref='))).toHaveLength(1)

    wrapper.unmount()
  })

  it('opens task inspection from the board and restores focus when history closes it', async () => {
    window.history.replaceState(null, '', '/#/work?project=core')
    vi.stubGlobal('fetch', appFetch(SNAPSHOT))
    const wrapper = mount(App, {
      attachTo: document.body,
      global: { plugins: [createPinia()] },
    })
    await flushPromises()

    const trigger = wrapper.get<HTMLButtonElement>('button[aria-label^="Inspect t-01TASK935:"]')
    trigger.element.focus()
    await trigger.trigger('click')
    await flushPromises()
    expect(window.location.hash).toContain('task=t-01TASK935')
    expect(document.querySelector('[role="dialog"]')?.textContent).toContain('Build task explorer')

    window.history.replaceState(null, '', '/#/work?project=core')
    window.dispatchEvent(new PopStateEvent('popstate'))
    await flushPromises()
    expect(document.querySelector('[role="dialog"]')).toBeNull()
    expect(document.activeElement).toBe(trigger.element)
    wrapper.unmount()
  })

  it('reloads an exact project deep link and requests only delivery dependencies', async () => {
    window.history.replaceState(null, '', '/#/delivery?project=core')
    const fetchImpl = appFetch(SNAPSHOT)
    vi.stubGlobal('fetch', fetchImpl)

    const w = mount(App, { global: { plugins: [createPinia()] } })
    await flushPromises()

    expect(w.find('a[aria-current="page"]').text()).toContain('Delivery')
    expect(w.find('#dag-section-h').exists()).toBe(true)
    const urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(urls).toContain('/api/overview')
    expect(urls).toContain('/api/projects')
    expect(urls).toContain('/api/graph?project=core')
    expect(urls).not.toContain('/api/agents')
    expect(urls).not.toContain('/api/burn')
    expect(urls).not.toContain('/api/roles')

    w.unmount()
  })

  it('deep-links an exact role, keeps missing identities exact, and never substitutes', async () => {
    window.history.replaceState(null, '', '/#/team?role=builder')
    const fetchImpl = appFetch(SNAPSHOT)
    vi.stubGlobal('fetch', fetchImpl)

    const w = mount(App, {
      attachTo: document.body,
      global: { plugins: [createPinia()] },
    })
    await flushPromises()

    expect(document.querySelector('[role="dialog"]')?.textContent).toContain('builder')
    expect(document.querySelector('[role="dialog"]')?.textContent).toContain('writes the code')
    const urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(new Set(urls)).toEqual(new Set(['/api/overview', '/api/roles', '/api/agents']))

    window.history.replaceState(null, '', '/#/team?role=removed-role')
    window.dispatchEvent(new PopStateEvent('popstate'))
    await flushPromises()
    const dialogText = document.querySelector('[role="dialog"]')?.textContent ?? ''
    expect(dialogText).toContain('removed-role')
    expect(dialogText).toContain('Role no longer observed')
    expect(dialogText).not.toContain('writes the code')

    w.unmount()
  })

  it('opens role inspection from its button and restores focus when history closes it', async () => {
    window.history.replaceState(null, '', '/#/team')
    vi.stubGlobal('fetch', appFetch(SNAPSHOT))

    const w = mount(App, {
      attachTo: document.body,
      global: { plugins: [createPinia()] },
    })
    await flushPromises()

    const trigger = w.get<HTMLButtonElement>('button[aria-label="Inspect builder"]')
    trigger.element.focus()
    await trigger.trigger('click')
    await flushPromises()
    expect(window.location.hash).toBe('#/team?role=builder')
    expect(document.querySelector('[role="dialog"]')).not.toBeNull()

    window.history.replaceState(null, '', '/#/team')
    window.dispatchEvent(new PopStateEvent('popstate'))
    await flushPromises()
    expect(document.querySelector('[role="dialog"]')).toBeNull()
    expect(document.activeElement).toBe(trigger.element)

    w.unmount()
  })

  it('restores URL-backed observation filters and reports exact filtered counts', async () => {
    window.history.replaceState(null, '', '/#/agents?q=not-present&runtime=codex&state=acting')
    vi.stubGlobal('fetch', appFetch(SNAPSHOT))

    const w = mount(App, { global: { plugins: [createPinia()] } })
    await flushPromises()

    expect(w.text()).toContain('0 of 1 live agents')
    expect(w.find('input[type="search"]').element.getAttribute('value')).toBe('not-present')
    expect(w.text()).toContain('no live agents')
    expect(window.location.hash).toContain('runtime=codex')
    expect(window.location.hash).toContain('state=acting')
    w.unmount()
  })

  it('restores an activity deep link with one bounded server request and inert event bodies', async () => {
    window.history.replaceState(
      null,
      '',
      '/#/activity?project=core&kind=finding&actor=a-reviewer&event_state=pending&range=24h',
    )
    const fetchImpl = appFetch(SNAPSHOT)
    vi.stubGlobal('fetch', fetchImpl)

    const w = mount(App, { attachTo: document.body, global: { plugins: [createPinia()] } })
    await flushPromises()

    expect(w.text()).toContain('Activity and refusals')
    expect(w.text()).toContain('Review finding')
    expect(w.text()).toContain('pending owner action')
    expect(w.find('img').exists()).toBe(false)
    const urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    const activityURL =
      '/api/events?state=pending&range=24h&limit=50&project=core&kind=finding&actor=a-reviewer'
    expect(urls.filter((url) => url === activityURL)).toHaveLength(1)
    expect(urls).not.toContain('/api/agents')
    expect(urls).not.toContain('/api/roles')
    expect(urls).not.toContain('/api/burn')

    w.unmount()
  })
})
