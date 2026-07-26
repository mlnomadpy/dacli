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
    state: 'thinking',
    runtime_secs: 600,
    last_activity: new Date().toISOString(),
    transcript_url: '/api/agents/transcript?run=01KY8KW3W1GSP57K39ZY77NH6S',
    diff_url: '/api/agents/diff?run=01KY8KW3W1GSP57K39ZY77NH6S',
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

  it('shows the honest state as a badge whose word is the label, not color-only', () => {
    for (const state of ['thinking', 'acting', 'waiting', 'stalled'] as const) {
      const w = mount(AgentRow, { props: { agent: agent({ state }) } })
      const badge = w.find('.badge')
      expect(badge.exists()).toBe(true)
      // The state word is rendered as text, so meaning survives without color.
      expect(badge.text()).toBe(state)
      expect(badge.classes()).toContain(state)
      // And a plain-language title backs the badge for screen readers / hover.
      expect(badge.attributes('title')).toBeTruthy()
    }
  })

  it('links to the read-only transcript and diff views for the run', () => {
    const w = mount(AgentRow, {
      props: {
        agent: agent({
          transcript_url: '/api/agents/transcript?run=RID',
          diff_url: '/api/agents/diff?run=RID',
        }),
      },
    })
    const links = w.findAll('.links a')
    expect(links).toHaveLength(2)
    const transcript = links.find((a) => a.text() === 'transcript')!
    const diff = links.find((a) => a.text() === 'diff')!
    expect(transcript.attributes('href')).toBe('/api/agents/transcript?run=RID')
    expect(diff.attributes('href')).toBe('/api/agents/diff?run=RID')
    // Read-only: they open the views, they never post/mutate.
    links.forEach((a) => expect(a.attributes('rel')).toBe('noopener'))
  })
})
