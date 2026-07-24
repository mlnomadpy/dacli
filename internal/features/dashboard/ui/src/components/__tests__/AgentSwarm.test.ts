import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import AgentSwarm from '../AgentSwarm.vue'
import AgentRow from '../AgentRow.vue'
import type { Agent } from '@/types'

function agent(over: Partial<Agent> = {}): Agent {
  return {
    run_id: '01KY8KW3W1GSP57K39ZY77NH6S',
    child: 'a-nhkth9j71n',
    task: '131',
    role: 'designer',
    runtime: 'claude',
    pid: 48213,
    started: '2026-07-23T16:00:00Z',
    runtime_secs: 600,
    last_activity: new Date().toISOString(),
    ...over,
  }
}

describe('AgentSwarm (states)', () => {
  it('empty is the calm resting state — "no live agents", not an error', () => {
    const w = mount(AgentSwarm, {
      props: { agents: [], phase: 'live', hasSnapshot: true, error: null },
    })
    expect(w.text()).toContain('no live agents')
    expect(w.find('[role="alert"]').exists()).toBe(false)
    expect(w.find('table').exists()).toBe(false)
  })

  it('loading shows a skeleton, not the empty copy', () => {
    const w = mount(AgentSwarm, {
      props: { agents: [], phase: 'loading', hasSnapshot: false, error: null },
    })
    expect(w.find('.skeleton-table').exists()).toBe(true)
    expect(w.text()).not.toContain('no live agents')
  })

  it('live renders a table with column-scoped headers and one row per agent', () => {
    const w = mount(AgentSwarm, {
      props: {
        agents: [agent(), agent({ run_id: '01ZZZZZZZZ0000000000000000', child: 'a-two' })],
        phase: 'live',
        hasSnapshot: true,
        error: null,
      },
    })
    const heads = w.findAll('thead th')
    expect(heads).toHaveLength(8)
    heads.forEach((th) => expect(th.attributes('scope')).toBe('col'))
    expect(w.findAllComponents(AgentRow)).toHaveLength(2)
  })

  it('cold error shows a danger panel with Retry', async () => {
    const w = mount(AgentSwarm, {
      props: { agents: [], phase: 'error', hasSnapshot: false, error: 'network down' },
    })
    expect(w.find('[role="alert"]').exists()).toBe(true)
    await w.find('button.retry').trigger('click')
    expect(w.emitted('retry')).toHaveLength(1)
  })
})
