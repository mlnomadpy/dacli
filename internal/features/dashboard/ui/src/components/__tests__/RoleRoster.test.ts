import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import RoleRoster from '../RoleRoster.vue'
import RoleRow from '../RoleRow.vue'
import RoleCard from '../RoleCard.vue'
import RoleRosterSection from '../RoleRosterSection.vue'
import type { Role } from '@/types'

function role(over: Partial<Role> = {}): Role {
  return {
    name: 'builder',
    summary: 'writes the code',
    kind: 'implementer',
    grant: 'rw',
    runtime: 'claude',
    model: 'sonnet',
    wip: 2,
    max_points: 5,
    scope: ['internal/**'],
    out_of_scope: ['internal/agentid/**'],
    skills: ['go'],
    shortcuts: ['test'],
    escalate_to: ['maintainer'],
    active_agents: 1,
    wip_exceeded: false,
    has_prompt: true,
    ...over,
  }
}

describe('RoleRoster (states)', () => {
  it('empty is calm — a workspace can have no roles yet, that is not an error', () => {
    const w = mount(RoleRoster, {
      props: { roles: [], phase: 'live', hasSnapshot: true, error: null },
    })
    expect(w.text()).toContain('no roles defined yet')
    expect(w.find('[role="alert"]').exists()).toBe(false)
    expect(w.find('table').exists()).toBe(false)
  })

  it('loading shows a skeleton, not the empty copy', () => {
    const w = mount(RoleRoster, {
      props: { roles: [], phase: 'loading', hasSnapshot: false, error: null },
    })
    expect(w.find('.skeleton-table').exists()).toBe(true)
    expect(w.text()).not.toContain('no roles defined yet')
  })

  it('live renders a table with column-scoped headers and one row per role', () => {
    const w = mount(RoleRoster, {
      props: {
        roles: [role(), role({ name: 'maintainer' })],
        phase: 'live',
        hasSnapshot: true,
        error: null,
      },
    })
    const heads = w.findAll('thead th')
    expect(heads).toHaveLength(9)
    heads.forEach((th) => expect(th.attributes('scope')).toBe('col'))
    expect(w.findAllComponents(RoleRow)).toHaveLength(2)
    expect(w.findAllComponents(RoleCard)).toHaveLength(2)
    const scrollRegion = w.get('[aria-label^="Team roster table"]')
    expect(scrollRegion.attributes('role')).toBe('region')
    expect(scrollRegion.attributes('tabindex')).toBe('0')
    expect(w.findAll('button[aria-label="Inspect builder"]')).toHaveLength(2)
  })

  it('emits the exact role and semantic button when Inspect is activated', async () => {
    const w = mount(RoleRoster, {
      props: { roles: [role()], phase: 'live', hasSnapshot: true, error: null },
    })
    const button = w.get<HTMLButtonElement>('button[aria-label="Inspect builder"]')
    await button.trigger('click')
    expect(w.emitted('inspect')?.[0]).toEqual(['builder', button.element])
  })

  it('cold error shows a danger panel with Retry', async () => {
    const w = mount(RoleRoster, {
      props: { roles: [], phase: 'error', hasSnapshot: false, error: 'network down' },
    })
    expect(w.find('[role="alert"]').exists()).toBe(true)
    await w.find('button.retry').trigger('click')
    expect(w.emitted('retry')).toHaveLength(1)
  })
})

describe('RoleRow (the mechanical facts)', () => {
  it('shows grant, kind, cost routing, scope and skills', () => {
    const w = mount(RoleRow, { props: { role: role() } })
    const text = w.text()
    expect(text).toContain('builder')
    expect(text).toContain('implementer')
    expect(text).toContain('rw')
    expect(text).toContain('claude / sonnet')
    expect(text).toContain('internal/**')
    expect(text).toContain('go')
  })

  it('reads WIP as active/cap, and yells only when the cap is reached', () => {
    const under = mount(RoleRow, { props: { role: role({ active_agents: 1, wip: 2 }) } })
    expect(under.find('.wip').text()).toBe('1/2')
    expect(under.find('.wip').classes()).not.toContain('text-destructive')

    const at = mount(RoleRow, {
      props: { role: role({ active_agents: 2, wip: 2, wip_exceeded: true }) },
    })
    expect(at.find('.wip').text()).toBe('2/2')
    expect(at.find('.wip').classes()).toContain('text-destructive')
    // The number is always the label — colour is never the only signal.
    expect(at.find('.wip').attributes('title')).toContain('refused')
  })

  it('an uncapped role shows a bare count, not a fake denominator', () => {
    const w = mount(RoleRow, { props: { role: role({ wip: 0, active_agents: 3 }) } })
    expect(w.find('.wip').text()).toBe('3')
    expect(w.find('.wip').attributes('title')).toContain('uncapped')
  })

  it('an undeclared scope reads "everywhere" — permissive by design, not missing data', () => {
    const w = mount(RoleRow, { props: { role: role({ scope: [], out_of_scope: [] }) } })
    expect(w.find('.scope').text()).toBe('everywhere')
  })

  it('out-of-scope globs are carried in the scope cell title, since deny beats allow', () => {
    const w = mount(RoleRow, { props: { role: role() } })
    expect(w.find('.scope').attributes('title')).toContain('internal/agentid/**')
  })

  it('flags a metadata-only role as having no standing instructions', () => {
    expect(mount(RoleRow, { props: { role: role({ has_prompt: false }) } }).text()).toContain(
      '(no prompt)',
    )
    expect(mount(RoleRow, { props: { role: role() } }).text()).not.toContain('(no prompt)')
  })

  it('a kindless role reads "any" — it opts out of phase gating, it is not broken', () => {
    const w = mount(RoleRow, { props: { role: role({ kind: '' }) } })
    expect(w.text()).toContain('any')
  })
})

describe('RoleRosterSection (header)', () => {
  it('counts the roles and surfaces how many are at their WIP cap', () => {
    const w = mount(RoleRosterSection, {
      props: {
        roles: [role({ wip_exceeded: true }), role({ name: 'maintainer' })],
        phase: 'live',
        hasSnapshot: true,
        error: null,
      },
    })
    expect(w.text()).toContain('2 roles')
    expect(w.find('.capped').text()).toContain('1 at WIP cap')
  })

  it('says nothing about caps when none is reached', () => {
    const w = mount(RoleRosterSection, {
      props: { roles: [role()], phase: 'live', hasSnapshot: true, error: null },
    })
    expect(w.text()).toContain('1 roles')
    expect(w.find('.capped').exists()).toBe(false)
  })

  it('is a labelled landmark section', () => {
    const w = mount(RoleRosterSection, {
      props: { roles: [role()], phase: 'live', hasSnapshot: true, error: null },
    })
    expect(w.find('section').attributes('aria-labelledby')).toBe('roster-h')
    expect(w.find('#roster-h').text()).toBe('Team roster')
  })
})
