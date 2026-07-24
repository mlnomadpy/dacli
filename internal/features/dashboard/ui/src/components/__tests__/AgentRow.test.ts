import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
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

/** An ISO timestamp `secs` seconds in the past, for freshness bucketing. */
function agoIso(secs: number): string {
  return new Date(Date.now() - secs * 1000).toISOString()
}

describe('AgentRow', () => {
  it('truncates run_id to 10 mono chars and renders pid + uptime', () => {
    const w = mount(AgentRow, { props: { agent: agent() } })
    expect(w.find('td.run').text()).toContain('01KY8KW3W1')
    expect(w.find('td.run').text()).not.toContain('01KY8KW3W1G') // exactly 10
    expect(w.text()).toContain('48213')
    expect(w.text()).toContain('10m 0s') // duration(600)
  })

  it('encodes last_activity freshness on the activity dot', () => {
    const fresh = mount(AgentRow, { props: { agent: agent({ last_activity: agoIso(10) }) } })
    expect(fresh.find('.dot').classes()).toContain('fresh')

    const idle = mount(AgentRow, { props: { agent: agent({ last_activity: agoIso(120) }) } })
    expect(idle.find('.dot').classes()).toContain('idle')

    const stale = mount(AgentRow, { props: { agent: agent({ last_activity: agoIso(1000) }) } })
    expect(stale.find('.dot').classes()).toContain('stale')
  })

  it('gives the dot a text title so meaning is not color-only', () => {
    const w = mount(AgentRow, { props: { agent: agent({ last_activity: agoIso(10) }) } })
    expect(w.find('.dot').attributes('title')).toBe('active <60s')
  })

  it('renders an em-dash for empty optional fields', () => {
    const w = mount(AgentRow, { props: { agent: agent({ child: '', task: '', role: '' }) } })
    expect(w.findAll('td').filter((td) => td.text() === '—').length).toBeGreaterThan(0)
  })
})
