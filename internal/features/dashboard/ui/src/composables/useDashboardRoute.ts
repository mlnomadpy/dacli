import { computed, onMounted, onUnmounted, ref } from 'vue'
import type { DashboardRouteName } from '@/stores/dashboard'

export interface DashboardRouteDefinition {
  name: Exclude<DashboardRouteName, 'unknown'>
  number: string
  label: string
  eyebrow: string
  description: string
}

export interface DashboardSelection {
  project?: string
  task?: string
  agent?: string
  role?: string
  filter_role?: string
  q?: string
  runtime?: string
  model?: string
  state?: string
  kind?: string
  actor?: string
  event_state?: 'all' | 'pending' | 'applied'
  cursor?: string
  range?: DashboardTimeRange
  live?: 'paused'
}

export const DASHBOARD_TIME_RANGES = ['24h', '7d', '30d'] as const
export type DashboardTimeRange = (typeof DASHBOARD_TIME_RANGES)[number]

export interface DashboardLocation {
  name: DashboardRouteName
  path: string
  selection: DashboardSelection
  invalidSelection: boolean
}

export const DASHBOARD_ROUTES: readonly DashboardRouteDefinition[] = [
  {
    name: 'overview',
    number: '01',
    label: 'Overview',
    eyebrow: 'Decision surface',
    description: 'Workspace attention, current movement, and the project portfolio at a glance.',
  },
  {
    name: 'work',
    number: '02',
    label: 'Work',
    eyebrow: 'Task portfolio',
    description: 'Status and estimated work for the selected project.',
  },
  {
    name: 'agents',
    number: '03',
    label: 'Agents',
    eyebrow: 'Execution fleet',
    description: 'Live worker evidence and measured model-token intensity.',
  },
  {
    name: 'team',
    number: '04',
    label: 'Team',
    eyebrow: 'Routing policy',
    description: 'Role authority, runtime selection, scope, skills, and capacity.',
  },
  {
    name: 'activity',
    number: '05',
    label: 'Activity',
    eyebrow: 'Journal inbox',
    description: 'Append-only activity, refusals, findings, and owner reconciliation evidence.',
  },
  {
    name: 'delivery',
    number: '06',
    label: 'Delivery',
    eyebrow: 'Dependency evidence',
    description: 'The selected project graph, schedule, and recorded critical path.',
  },
] as const

const routeNames = new Set(DASHBOARD_ROUTES.map((route) => route.name))
const identityKeys = [
  'project',
  'task',
  'agent',
  'role',
  'filter_role',
  'runtime',
  'model',
  'state',
  'kind',
  'actor',
  'event_state',
  'cursor',
] as const
const selectionKeys = [...identityKeys, 'q', 'range', 'live'] as const
const safeIdentity = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/
const safeSearch = /^[^\u0000-\u001f\u007f]{1,128}$/
const timeRanges = new Set<string>(DASHBOARD_TIME_RANGES)

function validSelection(key: (typeof selectionKeys)[number], value: string): boolean {
  if (key === 'q') return safeSearch.test(value)
  if (key === 'range') return timeRanges.has(value)
  if (key === 'live') return value === 'paused'
  if (key === 'event_state') return ['all', 'pending', 'applied'].includes(value)
  return safeIdentity.test(value)
}

export function parseDashboardHash(hash: string): DashboardLocation {
  const raw = hash.replace(/^#\/?/, '')
  const [rawPath = '', rawQuery = ''] = raw.split('?', 2)
  const path = rawPath || 'overview'
  const name = routeNames.has(path as DashboardRouteDefinition['name'])
    ? (path as DashboardRouteDefinition['name'])
    : 'unknown'
  const query = new URLSearchParams(rawQuery)
  const selection: DashboardSelection = {}
  let invalidSelection = false
  for (const key of selectionKeys) {
    const value = query.get(key)
    if (!value) continue
    if (!validSelection(key, value)) {
      invalidSelection = true
      continue
    }
    if (key === 'range') selection.range = value as DashboardTimeRange
    else if (key === 'live') selection.live = 'paused'
    else if (key === 'event_state')
      selection.event_state = value as DashboardSelection['event_state']
    else selection[key] = value
  }
  return { name, path, selection, invalidSelection }
}

export function dashboardHref(
  name: Exclude<DashboardRouteName, 'unknown'>,
  selection: DashboardSelection = {},
): string {
  const query = new URLSearchParams()
  for (const key of selectionKeys) {
    const value = selection[key]
    if (value && validSelection(key, value)) query.set(key, value)
  }
  const encoded = query.toString()
  const suffix = encoded ? `?${encoded}` : ''
  return `#/${name}${suffix}`
}

export function useDashboardRoute() {
  const location = ref(parseDashboardHash(window.location.hash))

  const current = computed(() =>
    DASHBOARD_ROUTES.find((candidate) => candidate.name === location.value.name),
  )

  function sync(): void {
    location.value = parseDashboardHash(window.location.hash)
  }

  function replaceSelection(selection: DashboardSelection): void {
    if (location.value.name === 'unknown') return
    const href = dashboardHref(location.value.name, selection)
    window.history.replaceState(null, '', href)
    sync()
  }

  function pushSelection(selection: DashboardSelection): void {
    if (location.value.name === 'unknown') return
    const href = dashboardHref(location.value.name, selection)
    if (window.location.hash === href) return
    window.history.pushState(null, '', href)
    sync()
  }

  onMounted(() => {
    if (!window.location.hash) {
      window.history.replaceState(null, '', dashboardHref('overview'))
      sync()
    }
    window.addEventListener('hashchange', sync)
    window.addEventListener('popstate', sync)
  })
  onUnmounted(() => {
    window.removeEventListener('hashchange', sync)
    window.removeEventListener('popstate', sync)
  })

  return { location, current, replaceSelection, pushSelection }
}
