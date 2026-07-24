import { beforeEach, describe, it, expect } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDashboardStore } from '../dashboard'
import { emptyBurn } from '@/types'
import type { DashboardState } from '@/types'

const SNAPSHOT: DashboardState = {
  generated: '2026-07-23T16:10:00Z',
  pending_events: 3,
  projects: [
    {
      slug: 'core',
      title: 'dacli remaining backlog',
      stage: 'build',
      total: 42,
      counts: { open: 12, active: 3, blocked: 1, done: 26 },
      burndown: { done_points: 88.5, remaining_points: 31, unestimated: 4, per_day: [] },
    },
  ],
  agents: [],
  burn: {
    ...emptyBurn(),
    ceiling: 100,
    rate: 500,
    ratio: 5,
    alert: true,
    series: [{ day: '2026-07-24', tokens: 500, cost_usd: 0.6, runs: 1, per_run: 500 }],
  },
}

function okFetch(body: unknown): typeof fetch {
  return (async () =>
    new Response(JSON.stringify(body), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })) as unknown as typeof fetch
}

function statusFetch(code: number): typeof fetch {
  return (async () => new Response('nope', { status: code })) as unknown as typeof fetch
}

function throwingFetch(message: string): typeof fetch {
  return (async () => {
    throw new Error(message)
  }) as unknown as typeof fetch
}

describe('useDashboardStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('starts in loading with a null snapshot', () => {
    const store = useDashboardStore()
    expect(store.phase).toBe('loading')
    expect(store.state).toBeNull()
    expect(store.hasSnapshot).toBe(false)
    // burn defaults to a zero-safe empty before any snapshot arrives.
    expect(store.burn.alert).toBe(false)
    expect(store.burn.series).toHaveLength(0)
  })

  it('flips to live and exposes getters on a successful poll', async () => {
    const store = useDashboardStore()
    await store.pollOnce(okFetch(SNAPSHOT))
    expect(store.phase).toBe('live')
    expect(store.hasSnapshot).toBe(true)
    expect(store.pendingEvents).toBe(3)
    expect(store.projects).toHaveLength(1)
    // The burn getter carries the snapshot's burn payload through.
    expect(store.burn.alert).toBe(true)
    expect(store.burn.ceiling).toBe(100)
    expect(store.error).toBeNull()
    expect(store.lastOk).not.toBeNull()
  })

  it('enters error on a non-2xx response', async () => {
    const store = useDashboardStore()
    await store.pollOnce(statusFetch(500))
    expect(store.phase).toBe('error')
    expect(store.error).toBe('HTTP 500')
  })

  it('enters error on a thrown transport failure', async () => {
    const store = useDashboardStore()
    await store.pollOnce(throwingFetch('network down'))
    expect(store.phase).toBe('error')
    expect(store.error).toBe('network down')
  })

  it('retains the last good snapshot when a later poll fails', async () => {
    const store = useDashboardStore()
    await store.pollOnce(okFetch(SNAPSHOT))
    await store.pollOnce(throwingFetch('blip'))
    expect(store.phase).toBe('error')
    expect(store.hasSnapshot).toBe(true) // NOT blanked
    expect(store.projects).toHaveLength(1)
  })

  it('recovers to live on the first success after an error', async () => {
    const store = useDashboardStore()
    await store.pollOnce(throwingFetch('blip'))
    await store.retry(okFetch(SNAPSHOT))
    expect(store.phase).toBe('live')
    expect(store.error).toBeNull()
  })
})
