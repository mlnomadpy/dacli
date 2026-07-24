import { afterEach, describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import App from '@/App.vue'
import type { DashboardState } from '@/types'

// End-to-end wiring test: mount the whole App, let the store's poll loop pull a
// stubbed /api/state snapshot, and assert every surface renders live from it —
// Overview cards, the four-column board, the per-day burndown, and the swarm
// table. This exercises the real component tree App → sections → leaves, the
// thing the unit tests each cover in isolation.

const SNAPSHOT: DashboardState = {
  generated: '2026-07-23T16:10:00Z',
  pending_events: 2,
  projects: [
    {
      slug: 'core',
      title: 'dacli remaining backlog',
      stage: 'build',
      total: 42,
      counts: { open: 12, active: 3, blocked: 1, done: 26 },
      burndown: {
        done_points: 88.5,
        remaining_points: 31,
        unestimated: 4,
        per_day: [
          { day: '2026-07-20', points: 12 },
          { day: '2026-07-21', points: 8.5 },
        ],
      },
    },
  ],
  agents: [
    {
      run_id: '01KY8KW3W1GSP57K39ZY77NH6S',
      child: 'a-nhkth9j71n',
      task: '131',
      role: 'designer',
      runtime: 'claude',
      pid: 48213,
      started: '2026-07-23T16:00:00Z',
      runtime_secs: 600,
      last_activity: new Date().toISOString(),
    },
  ],
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('App (end-to-end)', () => {
  it('polls a snapshot and renders all three surfaces live', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(
        async () =>
          new Response(JSON.stringify(SNAPSHOT), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          }),
      ),
    )

    const w = mount(App, { global: { plugins: [createPinia()] } })
    await flushPromises()

    // Header connection tell flipped to live + pending pill.
    expect(w.text()).toContain('live · updated')
    expect(w.text()).toContain('2 pending events')

    // Overview: the project card with counts + burndown caption.
    expect(w.text()).toContain('dacli remaining backlog')
    expect(w.text()).toContain('88.5 done · 31.0 remaining pts · 4 unestimated')

    // Board: four columns with the project's true counts.
    const cols = w.findAll('[role="group"]')
    expect(cols).toHaveLength(4)
    expect(cols[0].attributes('aria-label')).toBe('open — 12 tasks')
    expect(cols[3].attributes('aria-label')).toBe('done — 26 tasks')

    // Burndown chart: one bar per day, server order preserved.
    expect(w.findAll('.chart .bar')).toHaveLength(2)

    // Swarm: a real table row for the live agent, newest-first.
    expect(w.find('table').exists()).toBe(true)
    expect(w.text()).toContain('01KY8KW3W1')
    expect(w.text()).toContain('1 running')

    w.unmount()
  })

  it('cold error before any snapshot shows section error panels with Retry', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new Error('network down')
      }),
    )

    const w = mount(App, { global: { plugins: [createPinia()] } })
    await flushPromises()

    expect(w.text()).toContain('connection lost: network down')
    // No snapshot ever → surfaces show their own cold error, each with Retry.
    const alerts = w.findAll('[role="alert"]')
    expect(alerts.length).toBeGreaterThanOrEqual(1)
    expect(w.find('button.retry').exists()).toBe(true)

    w.unmount()
  })
})
