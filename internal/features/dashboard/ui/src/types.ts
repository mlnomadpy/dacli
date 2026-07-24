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

export interface Project {
  slug: string
  title: string
  stage: string
  /** Task count across all statuses. */
  total: number
  counts: StatusCounts
  burndown: Burndown
}

export interface Agent {
  run_id: string
  child: string
  task: string
  role: string
  runtime: string
  pid: number
  /** RFC3339 UTC. */
  started: string
  /** Uptime, seconds. */
  runtime_secs: number
  /** transcript.log mtime, RFC3339 UTC; falls back to `started`. */
  last_activity: string
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

/**
 * The poller's tri-state, the sole driver of every global state view (DESIGN.md
 * §6). `loading` before the first success; `live` once a snapshot is in;
 * `error` on a failed poll (the last good snapshot is retained, not blanked).
 */
export type Phase = 'loading' | 'live' | 'error'
