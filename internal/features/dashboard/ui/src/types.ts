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

export interface DashboardState {
  /** RFC3339 UTC; when this snapshot was built. */
  generated: string
  /** Unsynced child events, as `dacli status` reports. */
  pending_events: number
  projects: Project[]
  /** Newest-first and already liveness-filtered by the server. */
  agents: Agent[]
}

/**
 * The poller's tri-state, the sole driver of every global state view (DESIGN.md
 * §6). `loading` before the first success; `live` once a snapshot is in;
 * `error` on a failed poll (the last good snapshot is retained, not blanked).
 */
export type Phase = 'loading' | 'live' | 'error'
