import type { Phase } from '@/types'

/**
 * The four per-surface states (DESIGN.md §6.3). Every section resolves exactly
 * one from the poller `phase` plus whether its own slice of the snapshot is
 * empty.
 */
export type SectionState = 'loading' | 'empty' | 'error' | 'live'

/**
 * Resolve a surface's state. Order encodes the spec's rule: a **cold** failure
 * (a poll errored and no snapshot has ever arrived) shows the section's own
 * `error` view; but once any snapshot is in, a later poll failure keeps the
 * surface on its retained data (`live`/`empty`) — the header alone shows the
 * connection error and dims the page (DESIGN.md §6.1–§6.3).
 */
export function sectionState(phase: Phase, hasSnapshot: boolean, isEmpty: boolean): SectionState {
  if (phase === 'loading') return 'loading'
  if (phase === 'error' && !hasSnapshot) return 'error'
  return isEmpty ? 'empty' : 'live'
}

export function useSectionState() {
  return { sectionState }
}
