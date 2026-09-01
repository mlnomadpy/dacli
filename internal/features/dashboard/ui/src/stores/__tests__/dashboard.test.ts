import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { FAST_POLL_MS, ROSTER_POLL_MS, SLOW_POLL_MS, useDashboardStore } from '../dashboard'
import { emptyBurn, emptyGraph } from '@/types'
import type { Agent, AgentDetail, Project, Role, TaskDetail, TaskSummary } from '@/types'

const generated = '2026-08-31T20:00:00Z'
const projects: Project[] = [
  {
    slug: 'core',
    title: 'dacli',
    stage: 'build',
    total: 3,
    counts: { open: 2, done: 1 },
    burndown: { done_points: 2, remaining_points: 4, unestimated: 0, per_day: [] },
    graph: emptyGraph(),
  },
  {
    slug: 'docs',
    title: 'docs',
    stage: 'verify',
    total: 1,
    counts: { open: 1 },
    burndown: { done_points: 0, remaining_points: 1, unestimated: 0, per_day: [] },
    graph: emptyGraph(),
  },
]
const agents: Agent[] = [
  {
    run_id: '01RUN',
    child: 'a-one',
    task: '932',
    role: 'frontend-engineer',
    runtime: 'codex',
    pid: 42,
    started: generated,
    state: 'acting',
    runtime_secs: 10,
    last_activity: generated,
    transcript_url: '/transcript',
    diff_url: '/diff',
  },
]
const roles: Role[] = [
  {
    name: 'frontend-engineer',
    summary: 'builds UI',
    kind: 'implementer',
    grant: 'rw',
    runtime: 'codex',
    model: 'gpt',
    wip: 1,
    max_points: 8,
    scope: ['internal/features/dashboard/**'],
    out_of_scope: [],
    skills: ['frontend'],
    shortcuts: [],
    escalate_to: [],
    active_agents: 1,
    wip_exceeded: true,
    has_prompt: true,
  },
]
const agentDetail: AgentDetail = {
  id: 'a-one',
  role: 'frontend-engineer',
  parent: 'a-root',
  grant: 'rw',
  retired: false,
  children: ['a-child'],
  tasks: [],
  runs: [],
}
const taskRow: TaskSummary = {
  id: 't-01TASK935',
  project: 'core',
  seq: 935,
  slug: 'task-explorer',
  title: 'Build task explorer',
  status: 'open',
  priority: 'high',
  owner: '',
  points: 3,
  estimated: true,
}
const taskDetail: TaskDetail = {
  ...taskRow,
  estimate: { optimistic: 2, probable: 3, pessimistic: 5, expected: 3.17 },
  so_that: 'operators can identify exact work',
  context: 'count-only boards hide task identity',
  acceptance: [{ text: 'show exact identity', done: false }],
  acceptance_done: 0,
  acceptance_total: 1,
  deps: [],
  parent: '',
  log: [],
}

function payload(url: string): unknown {
  if (url === '/api/overview') {
    return {
      generated,
      project_count: 2,
      task_count: 4,
      counts: { open: 3, done: 1 },
      pending_events: 2,
      live_agents: 1,
    }
  }
  if (url === '/api/projects') return { generated, projects }
  if (url === '/api/tasks?project=core') return { generated, tasks: [taskRow] }
  if (url === '/api/task?ref=t-01TASK935') return { generated, task: taskDetail }
  if (url === '/api/events?task=t-01TASK935') {
    return { generated, task: taskRow.id, limit: 50, truncated: false, events: [] }
  }
  if (url.startsWith('/api/events?state=')) {
    return {
      generated,
      limit: 50,
      truncated: false,
      partial: false,
      unreadable_records: 0,
      filters: { state: 'pending', range: '24h' },
      events: [],
    }
  }
  if (url === '/api/delivery-timeline?task=t-01TASK935') {
    return {
      schema: 'delivery-attempt-timeline/v1',
      generated,
      task: {
        id: taskRow.id,
        sequence: 935,
        generation: 1,
        project: 'core',
        title: taskRow.title,
        status: taskRow.status,
      },
      attempts: [],
      summary: 'No attempts recorded.',
    }
  }
  if (url === '/api/agents') return { generated, agents }
  if (url === '/api/agent?id=a-one') return { generated, agent: agentDetail }
  if (url === '/api/roles') return { generated, roles }
  if (url === '/api/burn') return { generated, ...emptyBurn(), ceiling: 100, rate: 90 }
  if (url.startsWith('/api/graph?project=')) {
    const project = decodeURIComponent(url.split('=')[1])
    return { generated, ...emptyGraph(), project }
  }
  throw new Error(`unexpected URL ${url}`)
}

function routerFetch(overrides: Record<string, number> = {}): typeof fetch {
  return vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input)
    const status = overrides[url] ?? 200
    return new Response(status === 200 ? JSON.stringify(payload(url)) : 'nope', {
      status,
      headers: { 'Content-Type': 'application/json' },
    })
  }) as unknown as typeof fetch
}

describe('useDashboardStore per-surface polling', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('starts every surface cold without fabricating data', () => {
    const store = useDashboardStore()
    expect(store.phase).toBe('loading')
    expect(store.projectsSurface.lastOk).toBeNull()
    expect(store.agentsSurface.lastOk).toBeNull()
    expect(store.rolesSurface.lastOk).toBeNull()
    expect(store.graphSurface.lastOk).toBeNull()
    expect(store.projects).toEqual([])
    expect(store.burn.series).toEqual([])
  })

  it('loads canonical endpoints independently and never calls /api/state', async () => {
    const fetchImpl = routerFetch()
    const store = useDashboardStore()
    await store.retryAll(fetchImpl)

    expect(store.phase).toBe('live')
    expect(store.pendingEvents).toBe(2)
    expect(store.projects).toHaveLength(2)
    expect(store.agents).toHaveLength(1)
    expect(store.roles).toHaveLength(1)
    expect(store.selectedSlug).toBe('core')
    expect(store.graphSurface.data.project).toBe('core')
    expect(fetchImpl).not.toHaveBeenCalledWith('/api/state', expect.anything())
  })

  it('keeps healthy surfaces live when one endpoint fails and retains its last good data', async () => {
    const store = useDashboardStore()
    await store.retryAll(routerFetch())
    store.activateRoute('agents')
    const failedAgents = routerFetch({ '/api/agents': 503 })
    await store.pollAgents(failedAgents)

    expect(store.phase).toBe('partial')
    expect(store.agentsSurface.phase).toBe('error')
    expect(store.agentsSurface.error).toBe('HTTP 503')
    expect(store.agents).toHaveLength(1)
    expect(store.projectsSurface.phase).toBe('live')
    expect(store.projects).toHaveLength(2)

    store.activateRoute('activity')
    expect(store.phase).toBe('partial')
    expect(store.error).toBeNull()
  })

  it('ignores an older response that completes after a newer observation', async () => {
    const store = useDashboardStore()
    const releases: Array<(value: Response) => void> = []
    const deferredFetch = vi.fn(
      () => new Promise<Response>((resolve) => releases.push(resolve)),
    ) as unknown as typeof fetch

    const older = store.pollAgents(deferredFetch)
    const newer = store.pollAgents(deferredFetch)
    releases[1](
      new Response(JSON.stringify({ generated, agents: [{ ...agents[0], child: 'a-new' }] })),
    )
    await newer
    releases[0](
      new Response(JSON.stringify({ generated, agents: [{ ...agents[0], child: 'a-old' }] })),
    )
    await older

    expect(store.agents[0].child).toBe('a-new')
  })

  it('ignores a response from a route that was left while its request was pending', async () => {
    const store = useDashboardStore()
    store.activateRoute('agents')
    let release: ((value: Response) => void) | undefined
    const deferredFetch = vi.fn(
      () => new Promise<Response>((resolve) => (release = resolve)),
    ) as unknown as typeof fetch

    const pending = store.pollAgents(deferredFetch)
    store.activateRoute('activity')
    release?.(new Response(JSON.stringify({ generated, agents })))
    await pending

    expect(store.agents).toEqual([])
    expect(store.agentsSurface.lastOk).toBeNull()
  })

  it('polls only the active route and changes the observation set without eager hidden reads', async () => {
    vi.useFakeTimers()
    const fetchImpl = routerFetch()
    const store = useDashboardStore()
    store.start(fetchImpl, 'overview')
    await vi.advanceTimersByTimeAsync(SLOW_POLL_MS)

    let urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(urls.filter((url) => url === '/api/overview')).toHaveLength(
      1 + SLOW_POLL_MS / FAST_POLL_MS,
    )
    expect(urls.filter((url) => url === '/api/projects')).toHaveLength(2)
    expect(urls).not.toContain('/api/agents')
    expect(urls).not.toContain('/api/burn')
    expect(urls).not.toContain('/api/roles')
    expect(urls).not.toContain('/api/graph?project=core')

    vi.mocked(fetchImpl).mockClear()
    store.activateRoute('agents')
    await vi.advanceTimersByTimeAsync(SLOW_POLL_MS)
    urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(urls.filter((url) => url === '/api/agents')).toHaveLength(
      1 + SLOW_POLL_MS / FAST_POLL_MS,
    )
    expect(urls.filter((url) => url === '/api/burn')).toHaveLength(2)
    expect(urls).not.toContain('/api/projects')
    expect(urls).not.toContain('/api/roles')
    expect(urls).not.toContain('/api/graph?project=core')

    vi.mocked(fetchImpl).mockClear()
    store.activateRoute('delivery')
    await vi.advanceTimersByTimeAsync(0)
    urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(urls).toContain('/api/projects')
    expect(urls).toContain('/api/graph?project=core')
    expect(urls).not.toContain('/api/agents')
    expect(urls).not.toContain('/api/burn')
    expect(urls).not.toContain('/api/graph?project=docs')
    expect(urls).not.toContain('/api/state')
    expect(ROSTER_POLL_MS).toBeGreaterThan(SLOW_POLL_MS)
    store.stop()
  })

  it('requests bounded graph modes, exact focus, status filters, and history pages from the server', async () => {
    const fetchImpl = routerFetch()
    const store = useDashboardStore()
    await store.pollProjects(fetchImpl)
    store.activateRoute('delivery')
    await store.setGraphQuery({ statuses: ['active', 'blocked'] }, fetchImpl)
    await store.setGraphQuery({ focus: 't-01FOCUS', page: 1 }, fetchImpl)
    await store.setGraphQuery({ mode: 'history', focus: '', page: 3 }, fetchImpl)

    const urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(urls).toContain('/api/graph?project=core&status=active%2Cblocked')
    expect(urls).toContain('/api/graph?project=core&focus=t-01FOCUS')
    expect(urls).toContain('/api/graph?project=core&mode=history&page=3')
    expect(urls.filter((url) => url.startsWith('/api/graph?'))).toHaveLength(3)
  })

  it('requests one server-filtered activity page and retains it when refresh fails', async () => {
    const fetchImpl = routerFetch()
    const store = useDashboardStore()
    store.activateRoute('activity')
    await store.configureActivity(
      {
        project: 'core',
        task: 't-01TASK937',
        kind: 'finding',
        actor: 'a-reviewer',
        state: 'pending',
        range: '24h',
        cursor: '01KCURSOR',
        limit: 50,
      },
      fetchImpl,
    )
    const exact =
      '/api/events?state=pending&range=24h&limit=50&project=core&task=t-01TASK937&kind=finding&actor=a-reviewer&cursor=01KCURSOR'
    expect(vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))).toEqual([exact])
    expect(store.activitySurface.phase).toBe('live')

    await store.pollActivity(routerFetch({ [exact]: 503 }))
    expect(store.activitySurface.phase).toBe('error')
    expect(store.activitySurface.data?.filters.state).toBe('pending')
    expect(store.activitySurface.error).toBe('HTTP 503')
  })

  it('pauses automatic observations, aborts in-flight generations, and resumes the active route', async () => {
    vi.useFakeTimers()
    const fetchImpl = routerFetch()
    const store = useDashboardStore()
    store.start(fetchImpl, 'agents')
    await vi.advanceTimersByTimeAsync(0)
    expect(vi.mocked(fetchImpl).mock.calls.length).toBeGreaterThan(0)

    vi.mocked(fetchImpl).mockClear()
    store.setPaused(true)
    await vi.advanceTimersByTimeAsync(SLOW_POLL_MS * 2)
    expect(fetchImpl).not.toHaveBeenCalled()
    expect(store.paused).toBe(true)

    store.setPaused(false)
    await vi.advanceTimersByTimeAsync(0)
    const urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(urls).toEqual(expect.arrayContaining(['/api/overview', '/api/agents', '/api/burn']))
    expect(store.paused).toBe(false)
    store.stop()
  })

  it('loads one cold snapshot for a paused deep link without arming refresh timers', async () => {
    vi.useFakeTimers()
    const fetchImpl = routerFetch()
    const store = useDashboardStore()
    store.setPaused(true)
    store.start(fetchImpl, 'agents')
    await vi.advanceTimersByTimeAsync(0)

    const initial = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(initial).toEqual(expect.arrayContaining(['/api/overview', '/api/agents', '/api/burn']))
    vi.mocked(fetchImpl).mockClear()
    await vi.advanceTimersByTimeAsync(SLOW_POLL_MS * 2)
    expect(fetchImpl).not.toHaveBeenCalled()
    store.stop()
  })

  it('drops a stale selected-project graph response', async () => {
    const store = useDashboardStore()
    store.activateRoute('delivery')
    const releases = new Map<string, (value: Response) => void>()
    const deferredGraph = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      return new Promise<Response>((resolve) => releases.set(url, resolve))
    }) as unknown as typeof fetch

    const coreRequest = store.selectProject('core', deferredGraph)
    const docsRequest = store.selectProject('docs', deferredGraph)
    releases.get('/api/graph?project=docs')?.(
      new Response(JSON.stringify({ generated, ...emptyGraph(), project: 'docs' })),
    )
    await docsRequest
    releases.get('/api/graph?project=core')?.(
      new Response(JSON.stringify({ generated, ...emptyGraph(), project: 'core' })),
    )
    await coreRequest

    expect(store.selectedSlug).toBe('docs')
    expect(store.graphSurface.data.project).toBe('docs')
  })

  it('loads agent detail only after exact selection and caches that observation', async () => {
    const fetchImpl = routerFetch()
    const store = useDashboardStore()
    store.activateRoute('agents')

    expect(fetchImpl).not.toHaveBeenCalled()
    await store.selectAgent('a-one', fetchImpl)
    expect(store.agentDetailSurface.data?.id).toBe('a-one')
    expect(vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))).toEqual([
      '/api/agent?id=a-one',
    ])

    await store.selectAgent('a-one', fetchImpl)
    expect(vi.mocked(fetchImpl).mock.calls).toHaveLength(1)
  })

  it('loads task rows by project and lazily fetches one exact detail plus event record', async () => {
    const fetchImpl = routerFetch()
    const store = useDashboardStore()
    await store.pollProjects(fetchImpl)
    store.activateRoute('work')
    vi.mocked(fetchImpl).mockClear()

    await store.pollTasks(fetchImpl)
    expect(store.tasksSurface.data.map((task) => task.id)).toEqual(['t-01TASK935'])
    expect(vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))).toEqual([
      '/api/tasks?project=core',
    ])

    await store.selectTask('t-01TASK935', fetchImpl)
    expect(store.taskDetailSurface.data?.id).toBe('t-01TASK935')
    expect(store.taskEventsSurface.data?.task).toBe('t-01TASK935')
    expect(vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))).toEqual([
      '/api/tasks?project=core',
      '/api/task?ref=t-01TASK935',
      '/api/events?task=t-01TASK935',
    ])

    await store.selectTask('t-01TASK935', fetchImpl)
    expect(vi.mocked(fetchImpl).mock.calls).toHaveLength(3)
  })

  it('refuses mismatched task identity and retains prior evidence on an unavailable refresh', async () => {
    const store = useDashboardStore()
    store.activateRoute('work')
    store.selectedSlug = 'core'
    await store.pollTasks(routerFetch())

    const wrongIdentity = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input)
      const body = url.startsWith('/api/task?')
        ? { generated, task: { ...taskDetail, id: 'core/other' } }
        : { generated, task: taskRow.id, limit: 50, truncated: false, events: [] }
      return new Response(JSON.stringify(body), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    }) as unknown as typeof fetch
    await store.selectTask('t-01TASK935', wrongIdentity)
    expect(store.selectedTaskRef).toBe('t-01TASK935')
    expect(store.taskDetailSurface.data).toBeNull()
    expect(store.taskDetailSurface.error).toContain('identity mismatch')

    await store.selectTask('', routerFetch())
    await store.selectTask('t-01TASK935', routerFetch())
    const unavailable = routerFetch({ '/api/task?ref=t-01TASK935': 404 })
    await store.pollTaskDetail(unavailable)
    expect(store.taskDetailSurface.data?.id).toBe('t-01TASK935')
    expect(store.taskDetailSurface.phase).toBe('error')
    expect(store.taskDetailSurface.status).toBe(404)
  })

  it('keeps the selected task identity when a refreshed row moves status columns', async () => {
    const store = useDashboardStore()
    store.activateRoute('work')
    store.selectedSlug = 'core'
    await store.pollTasks(routerFetch())
    await store.selectTask('t-01TASK935', routerFetch())

    const moved = vi.fn(
      async () =>
        new Response(JSON.stringify({ generated, tasks: [{ ...taskRow, status: 'done' }] }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
    ) as unknown as typeof fetch
    await store.pollTasks(moved)

    expect(store.selectedTaskRef).toBe('t-01TASK935')
    expect(store.tasksSurface.data[0].status).toBe('done')
    expect(store.taskDetailSurface.data?.id).toBe('t-01TASK935')
  })

  it('hydrates the shared task inspector once for a delivery deep link', async () => {
    const fetchImpl = routerFetch()
    const store = useDashboardStore()
    store.activateRoute('delivery')
    store.selectedTaskRef = 't-01TASK935'

    await store.pollTimeline(fetchImpl)

    expect(store.timelineSurface.data?.task.id).toBe('t-01TASK935')
    expect(store.taskDetailSurface.data?.id).toBe('t-01TASK935')
    expect(vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))).toEqual([
      '/api/delivery-timeline?task=t-01TASK935',
      '/api/task?ref=t-01TASK935',
      '/api/events?task=t-01TASK935',
    ])
  })

  it('refuses a mismatched or stale agent identity instead of replacing the selection', async () => {
    const store = useDashboardStore()
    store.activateRoute('agents')
    const wrongIdentity = vi.fn(
      async () =>
        new Response(JSON.stringify({ generated, agent: { ...agentDetail, id: 'a-other' } }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
    ) as unknown as typeof fetch

    await store.selectAgent('a-one', wrongIdentity)
    expect(store.selectedAgentID).toBe('a-one')
    expect(store.agentDetailSurface.data).toBeNull()
    expect(store.agentDetailSurface.phase).toBe('error')
    expect(store.agentDetailSurface.error).toContain('identity mismatch')
  })

  it('retains prior agent evidence and distinguishes an unavailable refresh', async () => {
    const store = useDashboardStore()
    store.activateRoute('agents')
    await store.selectAgent('a-one', routerFetch())

    const unavailable = routerFetch({ '/api/agent?id=a-one': 404 })
    await store.pollAgentDetail(unavailable)
    expect(store.agentDetailSurface.data?.id).toBe('a-one')
    expect(store.agentDetailSurface.phase).toBe('error')
    expect(store.agentDetailSurface.status).toBe(404)
  })
})
