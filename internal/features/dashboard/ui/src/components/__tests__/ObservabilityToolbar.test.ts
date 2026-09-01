import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ObservabilityToolbar from '../ObservabilityToolbar.vue'
import type { Agent, Project, Role } from '@/types'
import { emptyGraph } from '@/types'

const project: Project = {
  slug: 'core',
  title: 'dacli core',
  stage: 'build',
  total: 1,
  counts: { open: 1 },
  burndown: { done_points: 0, remaining_points: 1, unestimated: 0, per_day: [] },
  graph: emptyGraph(),
}
const agent: Agent = {
  run_id: '01RUN',
  child: 'a-ui',
  task: '950',
  role: 'frontend-engineer',
  runtime: 'codex',
  pid: 1,
  started: '2026-09-01T11:00:00Z',
  state: 'acting',
  runtime_secs: 60,
  last_activity: '2026-09-01T11:59:00Z',
  transcript_url: '/transcript',
  diff_url: '/diff',
}
const role: Role = {
  name: 'frontend-engineer',
  summary: 'Builds UI',
  kind: 'implementer',
  grant: 'rw',
  runtime: 'codex',
  model: 'gpt-5.6',
  wip: 1,
  max_points: 8,
  scope: [],
  out_of_scope: [],
  skills: [],
  shortcuts: [],
  escalate_to: [],
  active_agents: 1,
  wip_exceeded: false,
  has_prompt: true,
}

describe('ObservabilityToolbar', () => {
  it('emits complete URL-owned filter context and a pause transition', async () => {
    const wrapper = mount(ObservabilityToolbar, {
      props: {
        route: 'agents',
        selection: {},
        projects: [project],
        roles: [role],
        agents: [agent],
        generated: '2026-09-01T11:59:00Z',
        resultLabel: '1 of 1 live agents',
      },
    })

    expect(wrapper.text()).toContain('1 of 1 live agents')
    const runtime = wrapper.findAll('select').find((select) => select.element.value === '')
    expect(runtime).toBeDefined()
    await wrapper.find('input[type="search"]').setValue('950')
    await wrapper.find('input[type="search"]').trigger('change')
    const searchEvents = wrapper.emitted('change') ?? []
    expect(searchEvents[searchEvents.length - 1]?.[0]).toEqual({ q: '950' })

    await wrapper.get('button[aria-pressed="false"]').trigger('click')
    const pauseEvents = wrapper.emitted('change') ?? []
    expect(pauseEvents[pauseEvents.length - 1]?.[0]).toEqual({ live: 'paused' })
  })

  it('keeps unsupported filters visible as inactive context instead of applying them', () => {
    const wrapper = mount(ObservabilityToolbar, {
      props: {
        route: 'delivery',
        selection: { project: 'core', runtime: 'codex' },
        projects: [project],
        roles: [role],
        agents: [agent],
        generated: null,
        resultLabel: '1 of 1 projects · core',
      },
    })
    expect(wrapper.text()).toContain('Preserved for another route, not applied here: runtime')
    const runtimeSelect = wrapper
      .findAll('select')
      .find((select) => select.findAll('option').some((option) => option.text() === 'codex'))
    expect(runtimeSelect?.attributes('disabled')).toBeDefined()
  })

  it('keeps pause and freshness compact when a route owns its detailed filters', () => {
    const wrapper = mount(ObservabilityToolbar, {
      props: {
        route: 'activity',
        selection: { project: 'core', event_state: 'pending', range: '7d' },
        projects: [project],
        roles: [],
        agents: [],
        generated: '2026-09-01T11:59:00Z',
        resultLabel: '4 events on this page',
        compact: true,
      },
    })
    expect(wrapper.text()).toContain('4 events on this page')
    expect(wrapper.find('.observation-controls').exists()).toBe(false)
    expect(wrapper.get('button[aria-pressed="false"]').text()).toBe('Pause')
    expect(wrapper.text()).not.toContain('Preserved for another route')
  })
})
