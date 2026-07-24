import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ConnectionStatus from '../ConnectionStatus.vue'

describe('ConnectionStatus', () => {
  it('shows "connecting…" and no Retry while loading', () => {
    const w = mount(ConnectionStatus, {
      props: { phase: 'loading', generated: null, pendingEvents: 0, error: null },
    })
    expect(w.text()).toContain('connecting…')
    expect(w.find('button.retry').exists()).toBe(false)
  })

  it('shows the pending-events pill only when live and > 0', () => {
    const w = mount(ConnectionStatus, {
      props: {
        phase: 'live',
        generated: '2026-07-23T16:10:00Z',
        pendingEvents: 3,
        error: null,
      },
    })
    expect(w.text()).toContain('live · updated')
    expect(w.text()).toContain('3 pending events')
  })

  it('singularizes the pill copy', () => {
    const w = mount(ConnectionStatus, {
      props: { phase: 'live', generated: '2026-07-23T16:10:00Z', pendingEvents: 1, error: null },
    })
    expect(w.text()).toContain('1 pending event')
    expect(w.text()).not.toContain('1 pending events')
  })

  it('renders a Retry button in error and emits on click', async () => {
    const w = mount(ConnectionStatus, {
      props: { phase: 'error', generated: null, pendingEvents: 0, error: 'HTTP 500' },
    })
    expect(w.text()).toContain('connection lost: HTTP 500')
    const btn = w.find('button.retry')
    expect(btn.exists()).toBe(true)
    await btn.trigger('click')
    expect(w.emitted('retry')).toHaveLength(1)
  })
})
