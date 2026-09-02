import { afterEach, describe, expect, it } from 'vitest'
import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { dashboardHref, parseDashboardHash, useDashboardRoute } from '../useDashboardRoute'

afterEach(() => window.history.replaceState(null, '', '/'))

describe('dashboard route contract', () => {
  it('parses stable deep links and serializes exact safe selections', () => {
    expect(parseDashboardHash('#/delivery?project=core&task=941')).toEqual({
      name: 'delivery',
      path: 'delivery',
      selection: { project: 'core', task: '941' },
      invalidSelection: false,
    })
    expect(dashboardHref('team', { project: 'core', role: 'frontend-engineer' })).toBe(
      '#/team?project=core&role=frontend-engineer',
    )
    expect(
      parseDashboardHash(
        '#/agents?q=task+950&filter_role=frontend-engineer&runtime=codex&state=acting&range=24h&live=paused',
      ).selection,
    ).toEqual({
      q: 'task 950',
      filter_role: 'frontend-engineer',
      runtime: 'codex',
      state: 'acting',
      range: '24h',
      live: 'paused',
    })
    expect(
      dashboardHref('agents', {
        q: 'task 950',
        filter_role: 'frontend-engineer',
        runtime: 'codex',
        state: 'acting',
        range: '24h',
        live: 'paused',
      }),
    ).toBe(
      '#/agents?filter_role=frontend-engineer&runtime=codex&state=acting&q=task+950&range=24h&live=paused',
    )
    expect(
      parseDashboardHash('#/agents?project=core&outcome_range=90d&metric=throughput&day=2026-09-01')
        .selection,
    ).toEqual({ project: 'core', metric: 'throughput', day: '2026-09-01', outcome_range: '90d' })
  })

  it('preserves an exact project/task identity while refusing traversal-shaped refs', () => {
    expect(parseDashboardHash('#/work?project=core&task=t-01TASK935').selection.task).toBe(
      't-01TASK935',
    )
    expect(dashboardHref('work', { project: 'core', task: 't-01TASK935' })).toBe(
      '#/work?project=core&task=t-01TASK935',
    )
    expect(dashboardHref('work', { project: 'core', task: '../secret' })).toBe(
      '#/work?project=core',
    )
  })

  it('round-trips activity filters and stable cursors while rejecting malformed values', () => {
    const selection = {
      project: 'core',
      task: 't-01TASK937',
      kind: 'finding',
      actor: 'a-reviewer',
      event_state: 'pending' as const,
      range: '24h' as const,
      cursor: '01KXYZ',
    }
    const href = dashboardHref('activity', selection)
    expect(href).toBe(
      '#/activity?project=core&task=t-01TASK937&kind=finding&actor=a-reviewer&event_state=pending&cursor=01KXYZ&range=24h',
    )
    expect(parseDashboardHash(href).selection).toEqual(selection)
    expect(parseDashboardHash('#/activity?event_state=mutated&cursor=../secret')).toMatchObject({
      selection: {},
      invalidSelection: true,
    })
  })

  it('fails closed on unknown routes and malformed identities', () => {
    expect(parseDashboardHash('#/elsewhere?project=../../secret')).toEqual({
      name: 'unknown',
      path: 'elsewhere',
      selection: {},
      invalidSelection: true,
    })
    expect(dashboardHref('agents', { agent: '../outside' })).toBe('#/agents')
    expect(parseDashboardHash('#/agents?range=forever&live=maybe&q=%00secret')).toMatchObject({
      selection: {},
      invalidSelection: true,
    })
  })

  it('tracks hash history events used by Back and Forward navigation', async () => {
    window.history.replaceState(null, '', '/#/overview')
    const Harness = defineComponent({
      setup: () => useDashboardRoute(),
      template: '<output>{{ location.name }}:{{ location.selection.project ?? "" }}</output>',
    })
    const wrapper = mount(Harness)

    window.history.pushState(null, '', '/#/work?project=core')
    window.dispatchEvent(new HashChangeEvent('hashchange'))
    await nextTick()
    expect(wrapper.text()).toBe('work:core')

    // Browsers emit the same event after Back/Forward. The composable reads the
    // URL as authority instead of replaying a private client navigation stack.
    window.history.replaceState(null, '', '/#/overview')
    window.dispatchEvent(new HashChangeEvent('hashchange'))
    await nextTick()
    expect(wrapper.text()).toBe('overview:')

    wrapper.unmount()
  })

  it('pushes an inspectable selection into browser history and syncs immediately', async () => {
    window.history.replaceState(null, '', '/#/team')
    const Harness = defineComponent({
      setup: () => useDashboardRoute(),
      template:
        '<button @click="pushSelection({ role: \'builder\' })">inspect</button><output>{{ location.selection.role ?? "" }}</output>',
    })
    const wrapper = mount(Harness)

    await wrapper.get('button').trigger('click')
    expect(window.location.hash).toBe('#/team?role=builder')
    expect(wrapper.get('output').text()).toBe('builder')

    window.history.replaceState(null, '', '/#/team')
    window.dispatchEvent(new PopStateEvent('popstate'))
    await nextTick()
    expect(wrapper.get('output').text()).toBe('')

    wrapper.unmount()
  })
})
