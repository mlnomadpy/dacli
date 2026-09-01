import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import AgentInspector from '../AgentInspector.vue'
import type { AgentDetail } from '@/types'

const detail: AgentDetail = {
  id: 'a-worker',
  role: 'frontend-engineer',
  parent: 'a-root',
  grant: 'ro',
  retired: false,
  children: ['a-child'],
  tasks: [
    {
      id: 't-01TASK934',
      project: 'core',
      seq: 934,
      slug: 'agent-detail',
      title: 'Inspect agents',
      status: 'active',
      priority: 'high',
      owner: 'a-worker',
      points: 3,
      estimated: true,
    },
  ],
  runs: [
    {
      run_id: '01NEW',
      task: 't-01TASK934',
      role: 'frontend-engineer',
      runtime: 'codex',
      pid: 42,
      started: '2026-09-01T12:00:00Z',
      live: true,
      transcript_url: '/transcript?run=01NEW',
      diff_url: '/diff?run=01NEW',
    },
    {
      run_id: '01OLD',
      task: 'core/900',
      role: 'frontend-engineer',
      runtime: 'codex',
      pid: 0,
      started: '2026-08-31T12:00:00Z',
      live: false,
      transcript_url: '/transcript?run=01OLD',
      diff_url: '/diff?run=01OLD',
    },
  ],
}

function props(over: Record<string, unknown> = {}) {
  return {
    open: true,
    selectedID: 'a-worker',
    agent: detail,
    phase: 'live' as const,
    hasSnapshot: true,
    error: null,
    status: null,
    live: true,
    ...over,
  }
}

afterEach(() => document.body.replaceChildren())

describe('AgentInspector', () => {
  it('renders authority, lineage, owned work, and newest-first live/dead run evidence', async () => {
    const wrapper = mount(AgentInspector, { attachTo: document.body, props: props() })
    await nextTick()
    const text = document.body.textContent ?? ''
    for (const fact of [
      'a-worker',
      'frontend-engineer',
      'ro',
      'a-root',
      'a-child',
      'Inspect agents',
      '01NEW',
      'live',
      '01OLD',
      'dead',
      'PID 42',
    ]) {
      expect(text).toContain(fact)
    }
    expect(text.indexOf('01NEW')).toBeLessThan(text.indexOf('01OLD'))
    expect(document.querySelector('a[href="#/team?role=frontend-engineer"]')).not.toBeNull()
    const taskLink = [...document.querySelectorAll<HTMLAnchorElement>('a')].find((link) =>
      link.textContent?.includes('934 · Inspect agents'),
    )
    expect(taskLink?.getAttribute('href')).toContain('#/work?')
    expect(taskLink?.getAttribute('href')).toContain('task=t-01TASK934')
    expect(document.querySelectorAll('a[rel="noopener"]')).toHaveLength(6)
    wrapper.unmount()
  })

  it('navigates exact parent and child identities inside the same inspection model', async () => {
    const wrapper = mount(AgentInspector, { attachTo: document.body, props: props() })
    await nextTick()
    const parent = [...document.querySelectorAll<HTMLButtonElement>('button.agent-link')].find(
      (button) => button.textContent === 'a-root',
    )!
    parent.click()
    await nextTick()
    expect(wrapper.emitted('navigateAgent')?.[0]).toEqual(['a-root'])
    const child = [...document.querySelectorAll<HTMLButtonElement>('button.agent-link')].find(
      (button) => button.textContent === 'a-child',
    )!
    child.click()
    await nextTick()
    expect(wrapper.emitted('navigateAgent')?.[1]).toEqual(['a-child'])
    wrapper.unmount()
  })

  it('keeps retained evidence visible when the agent retires or refresh fails', async () => {
    const wrapper = mount(AgentInspector, {
      attachTo: document.body,
      props: props({
        phase: 'error',
        error: 'HTTP 500',
        live: false,
        agent: { ...detail, retired: true },
      }),
    })
    await nextTick()
    expect(document.body.textContent).toContain('retired')
    expect(document.querySelector('[role="alert"]')?.textContent).toContain(
      'retained evidence is stale',
    )
    expect(document.body.textContent).toContain('01OLD')
    wrapper.unmount()
  })

  it('distinguishes a missing exact record and offers an independent retry', async () => {
    const wrapper = mount(AgentInspector, {
      attachTo: document.body,
      props: props({
        agent: null,
        phase: 'error',
        hasSnapshot: false,
        error: 'HTTP 404',
        status: 404,
        live: false,
      }),
    })
    await nextTick()
    const alert = document.querySelector<HTMLElement>('[role="alert"]')!
    expect(alert.textContent).toContain('Agent record unavailable')
    expect(alert.textContent).toContain('a-worker')
    const retry = alert.querySelector<HTMLButtonElement>('button')!
    retry.click()
    await nextTick()
    expect(wrapper.emitted('retry')).toBeTruthy()
    wrapper.unmount()
  })

  it('remains a read-only sheet with no lifecycle mutation controls', async () => {
    const wrapper = mount(AgentInspector, { attachTo: document.body, props: props() })
    await nextTick()
    const text = (document.body.textContent ?? '').toLowerCase()
    for (const forbidden of ['kill agent', 'resume agent', 'change grant', 'assign task'])
      expect(text).not.toContain(forbidden)
    expect(text).toContain('no kill, retry, grant, or ownership authority')
    wrapper.unmount()
  })

  it('closes through Escape using the shared dialog keyboard primitive', async () => {
    const wrapper = mount(AgentInspector, { attachTo: document.body, props: props() })
    await nextTick()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(wrapper.emitted('close')).toHaveLength(1)
    wrapper.unmount()
  })

  it('uses a bounded desktop drawer that becomes full-width at 390px', async () => {
    const wrapper = mount(AgentInspector, { attachTo: document.body, props: props() })
    await nextTick()
    const dialog = document.querySelector<HTMLElement>('[role="dialog"]')
    expect(dialog?.classList.contains('w-full')).toBe(true)
    expect(dialog?.classList.contains('max-w-[640px]')).toBe(true)
    expect(dialog?.classList.contains('right-0')).toBe(true)
    wrapper.unmount()
  })
})
