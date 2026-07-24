import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ProjectGrid from '../ProjectGrid.vue'
import ProjectCard from '../ProjectCard.vue'
import type { Project } from '@/types'

function project(over: Partial<Project> = {}): Project {
  return {
    slug: 'core',
    title: 'dacli backlog',
    stage: 'build',
    total: 3,
    counts: { open: 2, done: 1 },
    burndown: { done_points: 10, remaining_points: 5, unestimated: 0, per_day: [] },
    ...over,
  }
}

describe('ProjectGrid (Overview states)', () => {
  it('loading: skeleton cards, no ProjectCard', () => {
    const w = mount(ProjectGrid, {
      props: { projects: [], phase: 'loading', hasSnapshot: false, error: null },
    })
    expect(w.findAllComponents(ProjectCard)).toHaveLength(0)
    expect(w.find('.skeleton-card').exists()).toBe(true)
  })

  it('empty: calm "no projects yet" once a snapshot has arrived', () => {
    const w = mount(ProjectGrid, {
      props: { projects: [], phase: 'live', hasSnapshot: true, error: null },
    })
    expect(w.text()).toContain('no projects yet')
    expect(w.find('[role="alert"]').exists()).toBe(false) // not an error
  })

  it('cold error: danger panel + Retry that emits', async () => {
    const w = mount(ProjectGrid, {
      props: { projects: [], phase: 'error', hasSnapshot: false, error: 'HTTP 500' },
    })
    expect(w.text()).toContain("couldn't load projects — HTTP 500")
    await w.find('button.retry').trigger('click')
    expect(w.emitted('retry')).toHaveLength(1)
  })

  it('retained snapshot stays live on a later poll failure (no cold error)', () => {
    const w = mount(ProjectGrid, {
      props: { projects: [project()], phase: 'error', hasSnapshot: true, error: 'blip' },
    })
    // hasSnapshot === true → data stays; the header (not the grid) owns the error.
    expect(w.findAllComponents(ProjectCard)).toHaveLength(1)
    expect(w.find('[role="alert"]').exists()).toBe(false)
  })

  it('live: one ProjectCard per project', () => {
    const w = mount(ProjectGrid, {
      props: {
        projects: [project(), project({ slug: 'ui', title: 'UI' })],
        phase: 'live',
        hasSnapshot: true,
        error: null,
      },
    })
    expect(w.findAllComponents(ProjectCard)).toHaveLength(2)
  })
})
