import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type {
  Agent,
  AgentsResponse,
  Burn,
  BurnResponse,
  Graph,
  GraphResponse,
  OverviewResponse,
  Phase,
  Project,
  ProjectsResponse,
  Role,
  RolesResponse,
} from '@/types'
import { emptyBurn, emptyGraph } from '@/types'

// Fast-changing process state stays near the old two-second heartbeat. Durable
// project/burn/graph projections refresh less often, and the roster is almost
// static. These intervals are exported so request-count tests prove the policy
// instead of sleeping and hoping (issue #932).
export const FAST_POLL_MS = 2_000
export const SLOW_POLL_MS = 10_000
export const ROSTER_POLL_MS = 60_000

type SurfaceName = 'overview' | 'projects' | 'agents' | 'burn' | 'roles' | 'graph'

interface Surface<T> {
  data: T
  phase: Exclude<Phase, 'partial'>
  error: string | null
  generated: string | null
  lastOk: number | null
  generation: number
  controller: AbortController | null
}

type SurfaceStatus = Pick<Surface<unknown>, 'phase' | 'error' | 'generated' | 'lastOk'>

function surface<T>(data: T): Surface<T> {
  return {
    data,
    phase: 'loading',
    error: null,
    generated: null,
    lastOk: null,
    generation: 0,
    controller: null,
  }
}

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

/**
 * Per-surface dashboard state (issue #932).
 *
 * The previous store polled `/api/state` every two seconds, rebuilding and
 * transferring every graph, role and burn record even when the user was not
 * looking at them. Each canonical GET projection now owns its freshness and
 * failure state. A request generation and AbortController prevent a slow,
 * older response from overwriting a newer observation.
 */
export const useDashboardStore = defineStore('dashboard', () => {
  const overviewSurface = ref(surface<OverviewResponse | null>(null))
  const projectsSurface = ref(surface<Project[]>([]))
  const agentsSurface = ref(surface<Agent[]>([]))
  const burnSurface = ref(surface<Burn>(emptyBurn()))
  const rolesSurface = ref(surface<Role[]>([]))
  const graphSurface = ref(surface<Graph>(emptyGraph()))
  const selectedSlug = ref('')

  let running = false
  const timers = new Map<SurfaceName, ReturnType<typeof setTimeout>>()
  let activeFetch: typeof fetch = fetch

  async function request<TPayload, TValue>(
    target: { value: Surface<TValue> },
    url: string,
    select: (payload: TPayload) => TValue,
    fetchImpl: typeof fetch = activeFetch,
  ): Promise<boolean> {
    const generation = target.value.generation + 1
    target.value.generation = generation
    target.value.controller?.abort()
    const controller = new AbortController()
    target.value.controller = controller
    try {
      const res = await fetchImpl(url, { cache: 'no-store', signal: controller.signal })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const payload = (await res.json()) as TPayload & { generated?: string }
      if (target.value.generation !== generation) return false
      target.value.data = select(payload)
      target.value.generated = payload.generated ?? null
      target.value.lastOk = Date.now()
      target.value.error = null
      target.value.phase = 'live'
      return true
    } catch (err) {
      if (target.value.generation !== generation || controller.signal.aborted) return false
      target.value.error = message(err)
      target.value.phase = 'error'
      return false
    } finally {
      if (target.value.generation === generation) target.value.controller = null
    }
  }

  const pollOverview = (fetchImpl: typeof fetch = activeFetch) =>
    request<OverviewResponse, OverviewResponse | null>(
      overviewSurface,
      '/api/overview',
      (payload) => payload,
      fetchImpl,
    )

  async function pollProjects(fetchImpl: typeof fetch = activeFetch): Promise<boolean> {
    const ok = await request<ProjectsResponse, Project[]>(
      projectsSurface,
      '/api/projects',
      (payload) => payload.projects ?? [],
      fetchImpl,
    )
    if (!ok) return false
    const projects = projectsSurface.value.data
    if (!projects.some((project) => project.slug === selectedSlug.value)) {
      selectedSlug.value = projects[0]?.slug ?? ''
      if (selectedSlug.value) void pollGraph(fetchImpl)
      else resetGraph()
    }
    return true
  }

  const pollAgents = (fetchImpl: typeof fetch = activeFetch) =>
    request<AgentsResponse, Agent[]>(
      agentsSurface,
      '/api/agents',
      (payload) => payload.agents ?? [],
      fetchImpl,
    )

  const pollBurn = (fetchImpl: typeof fetch = activeFetch) =>
    request<BurnResponse, Burn>(
      burnSurface,
      '/api/burn',
      (payload) => ({
        unit: payload.unit,
        series: payload.series,
        bands: payload.bands,
        windows: payload.windows,
        ceiling: payload.ceiling,
        rate: payload.rate,
        ratio: payload.ratio,
        alert: payload.alert,
        alert_at: payload.alert_at,
      }),
      fetchImpl,
    )

  const pollRoles = (fetchImpl: typeof fetch = activeFetch) =>
    request<RolesResponse, Role[]>(
      rolesSurface,
      '/api/roles',
      (payload) => payload.roles ?? [],
      fetchImpl,
    )

  async function pollGraph(fetchImpl: typeof fetch = activeFetch): Promise<boolean> {
    const slug = selectedSlug.value
    if (!slug) {
      resetGraph()
      return true
    }
    const ok = await request<GraphResponse, Graph>(
      graphSurface,
      `/api/graph?project=${encodeURIComponent(slug)}`,
      (payload) => ({
        project: payload.project,
        nodes: payload.nodes,
        edges: payload.edges,
        critical_path: payload.critical_path,
        duration: payload.duration,
        scheduled: payload.scheduled,
        note: payload.note,
      }),
      fetchImpl,
    )
    // Selection may have changed while the response was in flight. The request
    // generation normally catches this; checking the payload identity makes the
    // invariant explicit even under a custom fetch implementation.
    if (ok && graphSurface.value.data.project !== selectedSlug.value) {
      graphSurface.value.generation++
      graphSurface.value.phase = 'loading'
      return false
    }
    return ok
  }

  function resetGraph(): void {
    graphSurface.value.controller?.abort()
    graphSurface.value.generation++
    graphSurface.value.controller = null
    graphSurface.value.data = emptyGraph()
    graphSurface.value.phase = 'loading'
    graphSurface.value.error = null
    graphSurface.value.generated = null
    graphSurface.value.lastOk = null
  }

  function selectProject(slug: string, fetchImpl: typeof fetch = activeFetch): Promise<boolean> {
    if (slug === selectedSlug.value) return Promise.resolve(true)
    selectedSlug.value = slug
    resetGraph()
    if (slug) return pollGraph(fetchImpl)
    return Promise.resolve(true)
  }

  function schedule(
    name: SurfaceName,
    interval: number,
    poll: (fetchImpl?: typeof fetch) => Promise<boolean>,
  ): void {
    const loop = async () => {
      await poll(activeFetch)
      if (running) timers.set(name, setTimeout(loop, interval))
    }
    void loop()
  }

  function start(fetchImpl: typeof fetch = fetch): void {
    if (running) return
    running = true
    activeFetch = fetchImpl
    schedule('overview', FAST_POLL_MS, pollOverview)
    schedule('agents', FAST_POLL_MS, pollAgents)
    schedule('projects', SLOW_POLL_MS, pollProjects)
    schedule('burn', SLOW_POLL_MS, pollBurn)
    schedule('roles', ROSTER_POLL_MS, pollRoles)
    // The first graph request is started by pollProjects once a real project is
    // selected. Subsequent requests are independent and project-scoped.
    const graphLoop = async () => {
      if (selectedSlug.value) await pollGraph(activeFetch)
      if (running) timers.set('graph', setTimeout(graphLoop, SLOW_POLL_MS))
    }
    timers.set('graph', setTimeout(graphLoop, SLOW_POLL_MS))
  }

  function stop(): void {
    running = false
    for (const timer of timers.values()) clearTimeout(timer)
    timers.clear()
    for (const target of [
      overviewSurface,
      projectsSurface,
      agentsSurface,
      burnSurface,
      rolesSurface,
      graphSurface,
    ]) {
      target.value.controller?.abort()
      target.value.controller = null
    }
  }

  async function retryAll(fetchImpl: typeof fetch = activeFetch): Promise<void> {
    await Promise.all([
      pollOverview(fetchImpl),
      pollProjects(fetchImpl),
      pollAgents(fetchImpl),
      pollBurn(fetchImpl),
      pollRoles(fetchImpl),
    ])
    if (selectedSlug.value) await pollGraph(fetchImpl)
  }

  const projects = computed(() =>
    projectsSurface.value.data.map((project) => ({
      ...project,
      graph:
        project.slug === selectedSlug.value && graphSurface.value.data.project === project.slug
          ? graphSurface.value.data
          : emptyGraph(),
    })),
  )
  const agents = computed(() => agentsSurface.value.data)
  const roles = computed(() => rolesSurface.value.data)
  const burn = computed(() => burnSurface.value.data)
  const pendingEvents = computed(() => overviewSurface.value.data?.pending_events ?? 0)

  const surfaces = computed<SurfaceStatus[]>(() => {
    const current: SurfaceStatus[] = [
      overviewSurface.value,
      projectsSurface.value,
      agentsSurface.value,
      burnSurface.value,
      rolesSurface.value,
    ]
    if (selectedSlug.value) current.push(graphSurface.value)
    return current
  })
  const phase = computed<Phase>(() => {
    const current = surfaces.value
    const snapshots = current.filter((item) => item.lastOk !== null).length
    const errors = current.filter((item) => item.phase === 'error').length
    const loading = current.filter((item) => item.phase === 'loading').length
    if (snapshots === 0 && errors > 0 && loading === 0) return 'error'
    if (snapshots === 0) return 'loading'
    if (errors > 0 || loading > 0) return 'partial'
    return 'live'
  })
  const error = computed(() => {
    const failed: string[] = []
    const named: Array<[SurfaceName, Surface<unknown>]> = [
      ['overview', overviewSurface.value],
      ['projects', projectsSurface.value],
      ['agents', agentsSurface.value],
      ['burn', burnSurface.value],
      ['roles', rolesSurface.value],
      ['graph', graphSurface.value],
    ]
    for (const [name, value] of named) if (value.error) failed.push(`${name}: ${value.error}`)
    return failed.join('; ') || null
  })
  const generated = computed(() => {
    const stamps = surfaces.value
      .map((item) => item.generated)
      .filter((stamp): stamp is string => Boolean(stamp))
      .sort()
    return stamps.length > 0 ? stamps[stamps.length - 1] : null
  })

  return {
    overviewSurface,
    projectsSurface,
    agentsSurface,
    burnSurface,
    rolesSurface,
    graphSurface,
    selectedSlug,
    projects,
    agents,
    roles,
    burn,
    pendingEvents,
    phase,
    error,
    generated,
    pollOverview,
    pollProjects,
    pollAgents,
    pollBurn,
    pollRoles,
    pollGraph,
    selectProject,
    start,
    stop,
    retryAll,
  }
})
