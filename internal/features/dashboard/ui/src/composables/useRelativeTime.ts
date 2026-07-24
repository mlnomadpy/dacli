// Time formatting for the UI — relative "4m ago" strings and human durations.
// Ported verbatim from `static/index.html`'s `ago()`/`duration()` so the Vue
// build reads identically to the page it replaces (DESIGN.md §0: render relative
// in the viewer's locale, keep the absolute UTC in a title).

/** "20s ago" / "4m ago" / "1h 4m ago" / "2d ago"; `-` for empty/invalid. */
export function ago(iso: string | null | undefined, now: number = Date.now()): string {
  if (!iso) return '-'
  const t = new Date(iso).getTime()
  if (Number.isNaN(t)) return '-'
  const secs = Math.max(0, Math.floor((now - t) / 1000))
  if (secs < 60) return `${secs}s ago`
  const mins = Math.floor(secs / 60)
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ${mins % 60}m ago`
  return `${Math.floor(hrs / 24)}d ago`
}

/** "10m 0s" / "1h 4m" / "45s" from a whole-second uptime. */
export function duration(secs: number): string {
  const s = Math.max(0, Math.floor(secs) || 0)
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const rem = s % 60
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${rem}s`
  return `${rem}s`
}

/**
 * Freshness of an agent's `last_activity` → the swarm's activity-dot bucket
 * (DESIGN.md §7.3). Thresholds are UI-local, documented here, not server fields.
 *  - `fresh`  < 60s  → `--ok`, pulsing ("still moving")
 *  - `idle`   < 5m   → `--muted`
 *  - `stale`  older  → `--blocked`, static ("possibly hung")
 */
export type Freshness = 'fresh' | 'idle' | 'stale'

export function freshness(iso: string | null | undefined, now: number = Date.now()): Freshness {
  if (!iso) return 'stale'
  const t = new Date(iso).getTime()
  if (Number.isNaN(t)) return 'stale'
  const secs = Math.max(0, Math.floor((now - t) / 1000))
  if (secs < 60) return 'fresh'
  if (secs < 300) return 'idle'
  return 'stale'
}

export function useRelativeTime() {
  return { ago, duration, freshness }
}
