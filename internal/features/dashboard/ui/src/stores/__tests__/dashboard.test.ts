import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { FAST_POLL_MS, ROSTER_POLL_MS, SLOW_POLL_MS, useDashboardStore } from '../dashboard'
import { emptyBurn, emptyGraph } from '@/types'
import type { Agent, Project, Role } from '@/types'

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
  if (url === '/api/agents') return { generated, agents }
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
    const failedAgents = routerFetch({ '/api/agents': 503 })
    await store.pollAgents(failedAgents)

    expect(store.phase).toBe('partial')
    expect(store.agentsSurface.phase).toBe('error')
    expect(store.agentsSurface.error).toBe('HTTP 503')
    expect(store.agents).toHaveLength(1)
    expect(store.projectsSurface.phase).toBe('live')
    expect(store.projects).toHaveLength(2)
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

  it('polls fast state frequently but roles and unselected graphs stay cold', async () => {
    vi.useFakeTimers()
    const fetchImpl = routerFetch()
    const store = useDashboardStore()
    store.start(fetchImpl)
    await vi.advanceTimersByTimeAsync(SLOW_POLL_MS)

    const urls = vi.mocked(fetchImpl).mock.calls.map(([input]) => String(input))
    expect(urls.filter((url) => url === '/api/overview')).toHaveLength(
      1 + SLOW_POLL_MS / FAST_POLL_MS,
    )
    expect(urls.filter((url) => url === '/api/agents')).toHaveLength(
      1 + SLOW_POLL_MS / FAST_POLL_MS,
    )
    expect(urls.filter((url) => url === '/api/projects')).toHaveLength(2)
    expect(urls.filter((url) => url === '/api/burn')).toHaveLength(2)
    expect(urls.filter((url) => url === '/api/roles')).toHaveLength(1)
    expect(urls.filter((url) => url === '/api/graph?project=core')).toHaveLength(2)
    expect(urls).not.toContain('/api/graph?project=docs')
    expect(urls).not.toContain('/api/state')
    expect(ROSTER_POLL_MS).toBeGreaterThan(SLOW_POLL_MS)
    store.stop()
  })

  it('drops a stale selected-project graph response', async () => {
    const store = useDashboardStore()
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
})
