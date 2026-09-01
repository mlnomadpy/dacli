import type { DashboardRouteName } from '@/stores/dashboard'
import type { Agent, Burn, Project, Role, TaskSummary } from '@/types'
import type { DashboardSelection, DashboardTimeRange } from './useDashboardRoute'

export type DashboardFilterKey =
  | 'project'
  | 'q'
  | 'filter_role'
  | 'runtime'
  | 'model'
  | 'state'
  | 'range'
  | 'kind'
  | 'actor'
  | 'event_state'
  | 'cursor'

export const ROUTE_FILTER_SUPPORT: Record<DashboardRouteName, readonly DashboardFilterKey[]> = {
  overview: ['project'],
  work: ['project', 'q'],
  agents: ['q', 'filter_role', 'runtime', 'state', 'range'],
  team: ['q', 'filter_role', 'runtime', 'model'],
  activity: ['project', 'range', 'kind', 'actor', 'event_state', 'cursor'],
  delivery: ['project'],
  unknown: [],
}

const rangeMillis: Record<DashboardTimeRange, number> = {
  '24h': 24 * 60 * 60 * 1_000,
  '7d': 7 * 24 * 60 * 60 * 1_000,
  '30d': 30 * 24 * 60 * 60 * 1_000,
}

function includes(value: string, query: string): boolean {
  return value.toLocaleLowerCase().includes(query.toLocaleLowerCase())
}

export function filterProjects(projects: Project[], selection: DashboardSelection): Project[] {
  if (!selection.project) return projects
  return projects.filter((project) => project.slug === selection.project)
}

export function filterTasks(tasks: TaskSummary[], query = ''): TaskSummary[] {
  const needle = query.trim()
  if (!needle) return tasks
  return tasks.filter((task) =>
    includes(
      [
        task.id,
        task.title,
        task.owner,
        task.priority,
        task.status,
        task.estimated ? '' : 'unestimated',
      ].join(' '),
      needle,
    ),
  )
}

export function filterAgents(
  agents: Agent[],
  selection: DashboardSelection,
  now = Date.now(),
): Agent[] {
  const query = selection.q?.trim() ?? ''
  const cutoff = selection.range ? now - rangeMillis[selection.range] : null
  return agents.filter((agent) => {
    if (selection.filter_role && agent.role !== selection.filter_role) return false
    if (selection.runtime && agent.runtime !== selection.runtime) return false
    if (selection.state && agent.state !== selection.state) return false
    if (cutoff !== null) {
      const observed = Date.parse(agent.last_activity || agent.started)
      if (!Number.isFinite(observed) || observed < cutoff) return false
    }
    if (!query) return true
    return includes(
      [agent.run_id, agent.child, agent.task, agent.role, agent.runtime, agent.state].join(' '),
      query,
    )
  })
}

export function filterRoles(roles: Role[], selection: DashboardSelection): Role[] {
  const query = selection.q?.trim() ?? ''
  return roles.filter((role) => {
    if (selection.filter_role && role.name !== selection.filter_role) return false
    if (selection.runtime && role.runtime !== selection.runtime) return false
    if (selection.model && role.model !== selection.model) return false
    if (!query) return true
    return includes(
      [
        role.name,
        role.summary,
        role.kind,
        role.grant,
        role.runtime,
        role.model,
        ...role.scope,
        ...role.skills,
      ].join(' '),
      query,
    )
  })
}

export function filterBurn(burn: Burn, selection: DashboardSelection, now = Date.now()): Burn {
  if (!selection.range) return burn
  const cutoff = now - rangeMillis[selection.range]
  return {
    ...burn,
    series: burn.series.filter((point) => {
      const observed = Date.parse(`${point.day}T23:59:59Z`)
      return Number.isFinite(observed) && observed >= cutoff
    }),
  }
}

export function supportsFilter(route: DashboardRouteName, key: DashboardFilterKey): boolean {
  return ROUTE_FILTER_SUPPORT[route].includes(key)
}

export function inactiveFilters(
  route: DashboardRouteName,
  selection: DashboardSelection,
): DashboardFilterKey[] {
  const active = (Object.keys(selection) as Array<keyof DashboardSelection>).filter(
    (key): key is DashboardFilterKey =>
      key !== 'task' &&
      key !== 'agent' &&
      key !== 'role' &&
      key !== 'live' &&
      Boolean(selection[key]),
  )
  return active.filter((key) => !supportsFilter(route, key))
}
