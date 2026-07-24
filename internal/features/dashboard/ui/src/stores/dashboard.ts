import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { DashboardState, Phase } from '@/types'

/** Poll cadence, ms — carried over verbatim from `static/index.html` (POLL_MS). */
export const POLL_MS = 2000

const STATE_URL = '/api/state'

/**
 * The single source of the app's `phase` (DESIGN.md §6.1), backed by Pinia so
 * the whole app reads one reactive snapshot rather than threading a composable's
 * refs through props. Semantics are identical to the spec's `usePoll`:
 *
 *  - `loading` before the first successful response (`state === null`)
 *  - `live`    once a poll succeeds; `state` holds the latest snapshot
 *  - `error`   when a poll throws/returns non-2xx; the LAST GOOD `state` is
 *              retained and kept on screen — the app never blanks on a transient
 *              failure. Polling continues; the first success flips back to live.
 *
 * The store owns the fetch loop; components stay pure and read via getters.
 */
export const useDashboardStore = defineStore('dashboard', () => {
  const state = ref<DashboardState | null>(null)
  const phase = ref<Phase>('loading')
  const error = ref<string | null>(null)
  /** Epoch ms of the last successful poll, or null before the first. */
  const lastOk = ref<number | null>(null)

  let timer: ReturnType<typeof setTimeout> | null = null
  let running = false

  /** True once any snapshot has ever arrived — sections only show their own
   * cold `error` view when this is false (DESIGN.md §6.3). */
  const hasSnapshot = computed(() => state.value !== null)

  const projects = computed(() => state.value?.projects ?? [])
  const agents = computed(() => state.value?.agents ?? [])
  const pendingEvents = computed(() => state.value?.pending_events ?? 0)
  const generated = computed(() => state.value?.generated ?? null)

  /**
   * A single fetch. Injectable `fetchImpl` keeps the network mockable in unit
   * tests without touching global fetch. Never throws — failure lands in
   * `error` and flips `phase` to `error` while retaining the last good `state`.
   */
  async function pollOnce(fetchImpl: typeof fetch = fetch): Promise<void> {
    try {
      const res = await fetchImpl(STATE_URL, { cache: 'no-store' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      state.value = (await res.json()) as DashboardState
      error.value = null
      lastOk.value = epochMs()
      phase.value = 'live'
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
      phase.value = 'error'
    }
  }

  /** Begin polling on the interval. Idempotent — a second call is a no-op. */
  function start(fetchImpl: typeof fetch = fetch): void {
    if (running) return
    running = true
    const loop = async () => {
      await pollOnce(fetchImpl)
      if (running) timer = setTimeout(loop, POLL_MS)
    }
    void loop()
  }

  /** Stop the interval (test teardown, unmount). */
  function stop(): void {
    running = false
    if (timer !== null) {
      clearTimeout(timer)
      timer = null
    }
  }

  /** Manual re-poll for the error-state Retry button (DESIGN.md §6.2). */
  function retry(fetchImpl: typeof fetch = fetch): Promise<void> {
    return pollOnce(fetchImpl)
  }

  return {
    // state
    state,
    phase,
    error,
    lastOk,
    // getters
    hasSnapshot,
    projects,
    agents,
    pendingEvents,
    generated,
    // actions
    pollOnce,
    start,
    stop,
    retry,
  }
})

// `Date.now()` isolated behind a helper so tests can reason about it and the one
// impure call in the store is named.
function epochMs(): number {
  return Date.now()
}
