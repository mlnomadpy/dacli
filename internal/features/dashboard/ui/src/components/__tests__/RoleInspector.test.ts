import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import RoleInspector from '../RoleInspector.vue'
import type { Agent, Role } from '@/types'

function role(over: Partial<Role> = {}): Role {
  return {
    name: 'builder',
    summary: 'Builds exact product slices',
    kind: 'implementer',
    grant: 'rw',
    runtime: 'codex',
    model: 'gpt-5.6-terra',
    wip: 2,
    max_points: 8,
    scope: ['internal/**'],
    out_of_scope: ['internal/agentid/**'],
    skills: ['go', 'frontend'],
    shortcuts: ['test'],
    escalate_to: ['maintainer'],
    active_agents: 1,
    wip_exceeded: false,
    has_prompt: true,
    ...over,
  }
}

function agent(over: Partial<Agent> = {}): Agent {
  return {
    run_id: '01RUN',
    child: 'a-builder',
    task: '933',
    role: 'builder',
    runtime: 'codex',
    pid: 42,
    started: '2026-09-01T00:00:00Z',
    state: 'acting',
    runtime_secs: 12,
    last_activity: '2026-09-01T00:00:10Z',
    transcript_url: '/transcript',
    diff_url: '/diff',
    ...over,
  }
}

function props(over: Record<string, unknown> = {}) {
  return {
    open: true,
    selectedName: 'builder',
    role: role(),
    rolesPhase: 'live' as const,
    rolesHasSnapshot: true,
    agents: [agent(), agent({ run_id: '02RUN', child: 'a-review', role: 'reviewer' })],
    agentsPhase: 'live' as const,
    agentsHasSnapshot: true,
    agentsError: null,
    ...over,
  }
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('RoleInspector', () => {
  it('renders a labelled read-only dialog with every mechanical role fact', async () => {
    const wrapper = mount(RoleInspector, { attachTo: document.body, props: props() })
    await nextTick()
    const dialog = document.querySelector<HTMLElement>('[role="dialog"]')
    expect(dialog).not.toBeNull()
    const labelID = dialog?.getAttribute('aria-labelledby')
    expect(labelID).toBeTruthy()
    expect(document.getElementById(labelID ?? '')?.textContent).toContain('builder')
    const text = dialog?.textContent ?? ''
    for (const fact of [
      'Builds exact product slices',
      'implementer',
      'rw',
      'codex',
      'gpt-5.6-terra',
      'Te 8',
      'internal/**',
      'internal/agentid/**',
      'frontend',
      'test',
      'maintainer',
      'Standing instructions: defined',
      'a-builder',
      'task 933',
      'projection only · no edit or spawn authority',
    ]) {
      expect(text).toContain(fact)
    }
    expect(text).not.toContain('a-review')
    wrapper.unmount()
  })

  it('states empty occupancy and never fabricates a live member', async () => {
    const wrapper = mount(RoleInspector, {
      attachTo: document.body,
      props: props({ agents: [] }),
    })
    await nextTick()
    expect(document.body.textContent).toContain('0 live members')
    expect(document.body.textContent).toContain('No live agents are assigned to this role')
    wrapper.unmount()
  })

  it('keeps the exact missing identity instead of substituting another role', async () => {
    const wrapper = mount(RoleInspector, {
      attachTo: document.body,
      props: props({ selectedName: 'removed-role', role: null }),
    })
    await nextTick()
    expect(document.body.textContent).toContain('removed-role')
    expect(document.body.textContent).toContain('Role no longer observed')
    expect(document.body.textContent).not.toContain('Builds exact product slices')
    wrapper.unmount()
  })

  it('closes through Escape using the dialog focus/keyboard primitive', async () => {
    const wrapper = mount(RoleInspector, { attachTo: document.body, props: props() })
    await nextTick()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(wrapper.emitted('close')).toHaveLength(1)
    wrapper.unmount()
  })

  it('returns an attempted outside focus move to the modal sheet', async () => {
    const outside = document.createElement('button')
    outside.textContent = 'outside'
    document.body.append(outside)
    const wrapper = mount(RoleInspector, { attachTo: document.body, props: props() })
    await nextTick()

    const dialog = document.querySelector<HTMLElement>('[role="dialog"]')
    outside.focus()
    await nextTick()
    expect(dialog?.contains(document.activeElement)).toBe(true)

    wrapper.unmount()
  })

  it('uses a bounded desktop drawer that remains full-width on narrow viewports', async () => {
    const wrapper = mount(RoleInspector, { attachTo: document.body, props: props() })
    await nextTick()
    const dialog = document.querySelector<HTMLElement>('[role="dialog"]')
    expect(dialog?.classList.contains('w-full')).toBe(true)
    expect(dialog?.classList.contains('max-w-[580px]')).toBe(true)
    expect(dialog?.classList.contains('right-0')).toBe(true)
    wrapper.unmount()
  })
})
