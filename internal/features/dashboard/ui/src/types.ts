// The data contract — a verbatim mirror of the Go `dashboardState` structs in
// `internal/features/dashboard/dashboard.go`. These are the ONLY fields the UI
// may bind to (DESIGN.md §0). Adding a field here without a matching server
// change is a spec violation: the UI never derives data the snapshot lacks.

/** Canonical task statuses, in the order they appear everywhere in the UI. */
export const STATUSES = ['open', 'active', 'blocked', 'done'] as const
export type Status = (typeof STATUSES)[number]

/** `counts` is a PARTIAL map — a status key is absent when its count is zero. */
export type StatusCounts = Partial<Record<Status, number>>

export interface BurndownDay {
  /** `YYYY-MM-DD`, already sorted chronologically by the server. */
  day: string
  points: number
}

export interface Burndown {
  done_points: number
  remaining_points: number
  /** Tasks with no PERT estimate; contribute to totals/counts but not points. */
  unestimated: number
  /** Done points that landed each day, chronological. May be empty. */
  per_day: BurndownDay[]
}

/** One task in the dependency DAG. `critical`/`slack`/`early_start` are
 * meaningful only when the graph is `scheduled` and the node is in the
 * scheduled (open, non-blocked) subset; otherwise `slack` is -1 and `critical`
 * is false. */
export interface GraphNode {
  id: string
  seq: number
  slug: string
  title: string
  status: Status
  /** PERT expected duration; 0 when unestimated. */
  points: number
  estimated: boolean
  /** On the zero-slack critical path. */
  critical: boolean
  /** -1 when unscheduled (done, blocked, or the graph is not scheduled). */
  slack: number
  early_start: number
}

/** A dependency edge: `from` must satisfy its type before `to`. Both ends are
 * always ids present in `Graph.nodes`. */
export interface GraphEdge {
  from: string
  to: string
  /** FS | SS | FF | SF (FS when unspecified). */
  type: string
}

/**
 * The task dependency DAG plus, when the open tasks are schedulable, the CPM
 * critical path drawn over them (what `internal/spm/criticalpath.go` computes).
 * The DAG (`nodes` + `edges`) is always present; the critical-path overlay is
 * best-effort: `scheduled` is true only when every open task is estimated and
 * the open subgraph is acyclic, else `note` explains the absence and the DAG
 * still renders. 0-safe: an empty project yields empty arrays and scheduled
 * false.
 */
export interface Graph {
  /** The project slug this graph covers; '' when it spans every project. */
  project: string
  nodes: GraphNode[]
  edges: GraphEdge[]
  /** Zero-slack chain in topological order (task ids); empty when unscheduled. */
  critical_path: string[]
  /** Project duration in Te units; 0 when unscheduled. */
  duration: number
  scheduled: boolean
  /** Why the critical path is absent (unestimated, cycle, or no open tasks). */
  note: string
}

export interface Project {
  slug: string
  title: string
  stage: string
  /** Task count across all statuses. */
  total: number
  counts: StatusCounts
  burndown: Burndown
  /** Dependency DAG + CPM critical path for this project. */
  graph: Graph
}

/** The honest per-agent activity the server derives from the task status and
 * transcript — never a guess from RAM/CPU. `thinking` = last line is assistant
 * prose; `acting` = last line is a `[tool: X]` marker; `waiting` = nothing
 * rendered yet (fresh spawn or a text runtime buffering to exit); `stalled` =
 * the transcript has frozen past the server's stall window while the process
 * is still alive; `blocked` = the agent's task has an outstanding `dacli ask`;
 * `silent` = a text runtime has stayed quiet past the stall window (buffered
 * output is normal briefly, not for minutes). */
export const AGENT_STATES = [
  'thinking',
  'acting',
  'waiting',
  'stalled',
  'blocked',
  'silent',
] as const
export type AgentState = (typeof AGENT_STATES)[number]

export interface Agent {
  run_id: string
  child: string
  task: string
  role: string
  runtime: string
  pid: number
  /** RFC3339 UTC. */
  started: string
  /** Honest activity derived server-side from the transcript. */
  state: AgentState
  /** Uptime, seconds. */
  runtime_secs: number
  /** transcript.log mtime, RFC3339 UTC; falls back to `started`. */
  last_activity: string
  /** Same-origin read-only link to the run's rendered transcript. */
  transcript_url: string
  /** Same-origin read-only link to the run's uncommitted diff (`git diff HEAD`). */
  diff_url: string
}

/** One day of burn: output tokens and USD summed across every usage-bearing run
 * that started that day, the run count, and the derived per-run intensity. */
export interface BurnPoint {
  /** `YYYY-MM-DD` UTC, already sorted chronologically by the server. */
  day: string
  tokens: number
  cost_usd: number
  runs: number
  /** tokens / runs; 0 when the day had no runs. */
  per_run: number
}

/** One calibrated agent band's per-run token expectation (role×model×runtime). */
export interface BurnBand {
  /** `role/model/runtime`. */
  band: string
  role: string
  /** Median output tokens per run for the band. */
  expected: number
  n: number
  /** n >= 10 — the store's authoritative-calibration gate. */
  calibrated: boolean
}

/** One project's live governor window spend, read from the persisted snapshot. */
export interface BurnWindow {
  project: string
  spent: number
  /** RFC3339 UTC, or '' when no window has opened yet. */
  start: string
}

/**
 * The token/cost burn story: actuals over time (`series`) measured against the
 * calibrated `ceiling`. `alert` is the server's verdict that `rate` is at or
 * above `alert_at`× the ceiling — the signal the chart must YELL on, not draw
 * as a passive line. 0-safe: an empty workspace yields an empty series and a
 * zero ceiling, and `alert` is false (nothing to compare against).
 */
export interface Burn {
  /** What series/ceiling/rate are measured in — `output_tokens`. */
  unit: string
  series: BurnPoint[]
  bands: BurnBand[]
  windows: BurnWindow[]
  /** Calibrated per-run token norm; 0 when there is no token history. */
  ceiling: number
  /** Current burn intensity: the latest day's per-run tokens. */
  rate: number
  /** rate / ceiling; 0 when ceiling is 0. */
  ratio: number
  /** ratio >= alert_at (and ceiling > 0): the chart yells. */
  alert: boolean
  /** The multiple that yells (1.5), echoed so the client thresholds identically. */
  alert_at: number
}

/**
 * One role on the team roster (dacli 226). A role is the only thing that
 * mechanically changes what an agent can do — which skills load, which paths are
 * in scope, which runtime and model it costs, how many may run at once — so this
 * is the whole configuration an operator would otherwise have to read out of
 * `.dacli/roles/*.md` by hand. Every list field is always an array, never null.
 */
export interface Role {
  name: string
  summary: string
  /** Lifecycle function phase gating acts on; '' means the role opts out. */
  kind: string
  /** Default capability a spawn into this role receives (ro | rw). */
  grant: string
  runtime: string
  model: string
  /** Concurrent-agent cap; 0 means uncapped. */
  wip: number
  /** Largest task expected size (Te) this role may take; 0 means uncapped. */
  max_points: number
  scope: string[]
  out_of_scope: string[]
  skills: string[]
  shortcuts: string[]
  escalate_to: string[]
  /** Non-retired agents currently holding the role — the WIP numerator. */
  active_agents: number
  /** The server's verdict that another spawn would break the cap. */
  wip_exceeded: boolean
  /** Whether the role file carries standing instructions below its frontmatter —
   * the difference between a role that is described and one that is defined. */
  has_prompt: boolean
}

export interface DashboardState {
  /** RFC3339 UTC; when this snapshot was built. */
  generated: string
  /** Unsynced child events, as `dacli status` reports. */
  pending_events: number
  projects: Project[]
  /** Newest-first and already liveness-filtered by the server. */
  agents: Agent[]
  /** Token/cost burn-rate over time vs the calibrated ceiling. */
  burn: Burn
  /** The team roster, sorted by name. Identical to what `/api/roles` serves. */
  roles: Role[]
}

/** A zero-safe empty burn — the getter's fallback before the first snapshot and
 * a resilient default if a payload ever omits the field. */
export function emptyBurn(): Burn {
  return {
    unit: 'output_tokens',
    series: [],
    bands: [],
    windows: [],
    ceiling: 0,
    rate: 0,
    ratio: 0,
    alert: false,
    alert_at: 1.5,
  }
}

/** A zero-safe empty graph — the fallback when a project has no graph yet, so
 * the DAG view never binds to undefined. */
export function emptyGraph(): Graph {
  return {
    project: '',
    nodes: [],
    edges: [],
    critical_path: [],
    duration: 0,
    scheduled: false,
    note: '',
  }
}

/**
 * The poller's tri-state, the sole driver of every global state view (DESIGN.md
 * §6). `loading` before the first success; `live` once a snapshot is in;
 * `error` on a failed poll (the last good snapshot is retained, not blanked).
 */
export type Phase = 'loading' | 'live' | 'error'
