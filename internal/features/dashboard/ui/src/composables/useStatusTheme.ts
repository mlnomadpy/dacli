import type { Status, StatusCounts } from '@/types'
import { STATUSES } from '@/types'

// Status → color is a SINGLE source of truth (DESIGN.md §3.1). Every status dot,
// count chip, and board column reads from `statusColor` — no per-component
// hardcoded hex. The values are CSS custom properties defined once on `:root`.
const STATUS_VAR: Record<Status, string> = {
  open: 'var(--open)',
  active: 'var(--active)',
  blocked: 'var(--blocked)',
  done: 'var(--done)',
}

export function statusColor(status: Status): string {
  return STATUS_VAR[status]
}

/**
 * `counts` is a partial map — a status key is absent when its count is zero.
 * Always read through this helper so a missing key shows `0`, never `undefined`
 * (DESIGN.md §0).
 */
export function count(counts: StatusCounts | null | undefined, status: Status): number {
  return counts?.[status] ?? 0
}

export function useStatusTheme() {
  return { statuses: STATUSES, statusColor, count }
}
