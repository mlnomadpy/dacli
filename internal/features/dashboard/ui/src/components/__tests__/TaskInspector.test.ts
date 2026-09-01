import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import TaskInspector from '../TaskInspector.vue'
import type { TaskDetail, TaskEventsResponse } from '@/types'

const task: TaskDetail = {
  id: 't-01TASK935',
  project: 'core',
  seq: 935,
  slug: 'task-explorer',
  title: 'Build task explorer',
  status: 'blocked',
  priority: 'critical',
  owner: 'a-builder',
  points: 3.17,
  estimated: true,
  estimate: { optimistic: 2, probable: 3, pessimistic: 5, expected: 3.17 },
  so_that: 'operators can inspect exact work',
  context: 'count-only boards hide task identity',
  acceptance: [
    { text: 'render task identity', done: true },
    { text: 'retain missing evidence', done: false },
  ],
  acceptance_done: 1,
  acceptance_total: 2,
  deps: [
    {
      ref: '934',
      type: 'FS',
      id: 't-01TASK934',
      title: 'Inspect agent history',
      status: 'done',
      resolved: true,
    },
    { ref: 'missing', type: 'SS', id: '', title: '', status: '', resolved: false },
  ],
  parent: 't-01TASK939',
  log: [
    { at: '2026-09-01T10:00:00Z', text: 'created' },
    { at: '2026-09-01T11:00:00Z', text: 'blocked on evidence' },
  ],
}

const events: TaskEventsResponse = {
  generated: '2026-09-01T12:00:00Z',
  task: 't-01TASK935',
  limit: 50,
  truncated: false,
  events: [
    {
      id: '01EVENT',
      kind: 'finding',
      actor: 'a-reviewer',
      about: 't-01TASK935',
      origin: 'agent',
      against: 'a-builder',
      applied: false,
      at: '2026-09-01T11:30:00Z',
      body: '<img src=x onerror=alert(1)> remains plain evidence',
    },
  ],
}

function props(over: Record<string, unknown> = {}) {
  return {
    open: true,
    selectedRef: 't-01TASK935',
    task,
    phase: 'live' as const,
    hasSnapshot: true,
    error: null,
    status: null,
    events,
    eventsPhase: 'live' as const,
    eventsHasSnapshot: true,
    eventsError: null,
    ...over,
  }
}

afterEach(() => document.body.replaceChildren())

describe('TaskInspector', () => {
  it('renders identity, estimate, narrative, acceptance, relations, history, and plain events', async () => {
    const wrapper = mount(TaskInspector, { attachTo: document.body, props: props() })
    await nextTick()
    const text = document.body.textContent ?? ''
    for (const fact of [
      'core / t-01TASK935',
      'blocked',
      'critical',
      'a-builder',
      'Optimistic',
      'operators can inspect exact work',
      '1 / 2 checked',
      't-01TASK934',
      'unresolved dependency',
      'blocked on evidence',
      'finding · a-reviewer',
      '<img src=x onerror=alert(1)> remains plain evidence',
    ])
      expect(text).toContain(fact)
    expect(document.body.querySelector('img')).toBeNull()
    expect(text.indexOf('blocked on evidence')).toBeLessThan(text.indexOf('created'))
    wrapper.unmount()
  })

  it('navigates parent and resolved dependencies but leaves dangling refs non-interactive', async () => {
    const wrapper = mount(TaskInspector, { attachTo: document.body, props: props() })
    await nextTick()
    const buttons = [...document.querySelectorAll<HTMLButtonElement>('button.task-link')]
    buttons[0].click()
    buttons[1].click()
    await nextTick()
    expect(wrapper.emitted('navigateTask')).toEqual([['t-01TASK939'], ['t-01TASK934']])
    expect(buttons).toHaveLength(2)
    wrapper.unmount()
  })

  it('keeps stale task evidence while distinguishing a failed event sub-surface', async () => {
    const wrapper = mount(TaskInspector, {
      attachTo: document.body,
      props: props({
        phase: 'error',
        error: 'HTTP 500',
        events: null,
        eventsPhase: 'error',
        eventsHasSnapshot: false,
        eventsError: 'HTTP 503',
      }),
    })
    await nextTick()
    expect(document.body.textContent).toContain('retained task evidence is stale')
    expect(document.body.textContent).toContain('Event history unavailable. HTTP 503')
    expect(document.body.textContent).toContain('render task identity')
    wrapper.unmount()
  })

  it('distinguishes an ambiguous or missing cold record without substitution', async () => {
    const wrapper = mount(TaskInspector, {
      attachTo: document.body,
      props: props({
        task: null,
        phase: 'error',
        hasSnapshot: false,
        status: 400,
        error: 'HTTP 400: ref is ambiguous',
      }),
    })
    await nextTick()
    expect(document.querySelector('[role="alert"]')?.textContent).toContain(
      'Task reference ambiguous',
    )
    expect(document.body.textContent).toContain('t-01TASK935')
    expect(document.body.textContent).not.toContain('Build task explorer')
    wrapper.unmount()
  })

  it('is keyboard-dismissible, 390px-safe, and read-only', async () => {
    const wrapper = mount(TaskInspector, { attachTo: document.body, props: props() })
    await nextTick()
    const dialog = document.querySelector<HTMLElement>('[role="dialog"]')
    expect(dialog?.classList.contains('w-full')).toBe(true)
    expect(dialog?.classList.contains('max-w-[720px]')).toBe(true)
    expect(document.body.textContent).toContain(
      'no transition, priority, acceptance, or dependency authority',
    )
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(wrapper.emitted('close')).toHaveLength(1)
    wrapper.unmount()
  })
})
