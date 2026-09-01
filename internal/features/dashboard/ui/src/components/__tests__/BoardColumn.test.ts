import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BoardColumn from '../BoardColumn.vue'
import type { TaskSummary } from '@/types'

const task: TaskSummary = {
  id: 't-01TASK935',
  project: 'core',
  seq: 935,
  slug: 'task-explorer',
  title: 'Build task explorer',
  status: 'open',
  priority: 'high',
  owner: 'a-builder',
  points: 3,
  estimated: true,
}

describe('BoardColumn', () => {
  it('renders truthful counts and inspectable task identities instead of count-only chips', async () => {
    const w = mount(BoardColumn, { props: { status: 'open', count: 5, tasks: [task] } })
    expect(w.find('.count').text()).toBe('5')
    expect(w.text()).toContain('935 · Build task explorer')
    expect(w.text()).toContain('1 matching 5 total')
    await w.get('button').trigger('click')
    expect(w.emitted('inspect')?.[0]?.[0]).toBe('t-01TASK935')
  })

  it('renders an explicit zero state rather than a blank gap', () => {
    const w = mount(BoardColumn, { props: { status: 'blocked', count: 0, tasks: [] } })
    expect(w.find('.count').text()).toBe('0')
    expect(w.find('.none').text()).toBe('no tasks')
  })

  it('labels the column region for screen readers', () => {
    const w = mount(BoardColumn, { props: { status: 'active', count: 3, tasks: [] } })
    expect(w.find('[role="group"]').attributes('aria-label')).toBe('active — 3 tasks')
  })
})
