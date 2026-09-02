<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import AppHeader from '@/components/AppHeader.vue'
import OverviewSection from '@/components/OverviewSection.vue'
import BurnRate from '@/components/BurnRate.vue'
import LoopOperations from '@/components/LoopOperations.vue'
import OutcomeAnalytics from '@/components/OutcomeAnalytics.vue'
import BoardSection from '@/components/BoardSection.vue'
import DagSection from '@/components/DagSection.vue'
import DeliveryTimeline from '@/components/DeliveryTimeline.vue'
import AgentSwarmSection from '@/components/AgentSwarmSection.vue'
import RoleRosterSection from '@/components/RoleRosterSection.vue'
import RoleInspector from '@/components/RoleInspector.vue'
import AgentInspector from '@/components/AgentInspector.vue'
import TaskInspector from '@/components/TaskInspector.vue'
import OperatorPulse from '@/components/OperatorPulse.vue'
import SectionNav from '@/components/SectionNav.vue'
import RouteIntro from '@/components/RouteIntro.vue'
import RouteNotFound from '@/components/RouteNotFound.vue'
import ObservabilityToolbar from '@/components/ObservabilityToolbar.vue'
import ActivitySection from '@/components/ActivitySection.vue'
import ErrorPanel from '@/components/ErrorPanel.vue'
import SkeletonBlock from '@/components/SkeletonBlock.vue'
import { useDashboardRoute } from '@/composables/useDashboardRoute'
import {
  filterAgents,
  filterBurn,
  filterProjects,
  filterRoles,
  filterTasks,
} from '@/composables/useObservabilityFilters'
import { useDashboardStore } from '@/stores/dashboard'

// The URL owns navigation/selection; the Pinia store owns observations. App is
// the only seam joining them and the only component that starts polling. This
// keeps the read-only dashboard from becoming a second workflow state machine
// (issue #941).
const store = useDashboardStore()
const route = useDashboardRoute()
const {
  phase,
  error,
  projects,
  agents,
  roles,
  pendingEvents,
  generated,
  burn,
  selectedSlug,
  overviewSurface,
  projectsSurface,
  tasksSurface,
  agentsSurface,
  rolesSurface,
  burnSurface,
  operationSurface,
  outcomeSurface,
  outcomeRange,
  graphSurface,
  timelineSurface,
  deliveryAttentionSurface,
  agentDetailSurface,
  taskDetailSurface,
  taskEventsSurface,
  activitySurface,
  selectedTaskRef,
  selectedAgentID,
} = storeToRefs(store)

const overviewReady = computed(
  () => overviewSurface.value.lastOk !== null && projectsSurface.value.lastOk !== null,
)
const overviewLoading = computed(
  () => overviewSurface.value.phase === 'loading' || projectsSurface.value.phase === 'loading',
)
const selectedRoleName = computed(() =>
  route.location.value.name === 'team' ? (route.location.value.selection.role ?? '') : '',
)
const selectedRole = computed(
  () => roles.value.find((candidate) => candidate.name === selectedRoleName.value) ?? null,
)
const roleReturnTarget = ref<HTMLElement | null>(null)
const agentReturnTarget = ref<HTMLElement | null>(null)
const taskReturnTarget = ref<HTMLElement | null>(null)
const selectedAgentName = computed(() =>
  route.location.value.name === 'agents' ? (route.location.value.selection.agent ?? '') : '',
)
const selectedAgentIsLive = computed(() =>
  agents.value.some((candidate) => candidate.child === selectedAgentName.value),
)
const selectedTaskName = computed(() =>
  route.location.value.name === 'work' || route.location.value.name === 'delivery'
    ? (route.location.value.selection.task ?? '')
    : '',
)
const filteredProjects = computed(() =>
  filterProjects(projects.value, route.location.value.selection),
)
const filteredAgents = computed(() => filterAgents(agents.value, route.location.value.selection))
const filteredRoles = computed(() => filterRoles(roles.value, route.location.value.selection))
const filteredBurn = computed(() => filterBurn(burn.value, route.location.value.selection))
const filteredTasks = computed(() =>
  filterTasks(tasksSurface.value.data, route.location.value.selection.q),
)
const observationResult = computed(() => {
  switch (route.location.value.name) {
    case 'overview':
      return `${filteredProjects.value.length} of ${projects.value.length} projects`
    case 'work':
      return `${filteredTasks.value.length} of ${tasksSurface.value.data.length} tasks · ${selectedSlug.value || 'no project'}`
    case 'delivery':
      return selectedSlug.value
        ? `1 of ${projects.value.length} projects · ${selectedSlug.value}`
        : `0 of ${projects.value.length} projects`
    case 'agents':
      return `${filteredAgents.value.length} of ${agents.value.length} live agents · ${selectedSlug.value || 'no project'}`
    case 'team':
      return `${filteredRoles.value.length} of ${roles.value.length} roles`
    case 'activity':
      return activitySurface.value.data
        ? `${activitySurface.value.data.events.length} events on this page`
        : 'activity not observed yet'
    default:
      return 'No observable route'
  }
})

function applyRouteSelection(refreshActivity = true): void {
  const project = route.location.value.selection.project
  if (project && project !== selectedSlug.value) void store.selectProject(project)
  if (route.location.value.name === 'delivery' || route.location.value.name === 'work') {
    void store.selectTask(route.location.value.selection.task ?? '')
  } else if (selectedTaskRef.value) {
    void store.selectTask('')
  }
  if (route.location.value.name === 'agents') {
    const requestedRange = route.location.value.selection.outcome_range
    if (requestedRange === '7d' || requestedRange === '30d' || requestedRange === '90d') {
      void store.setOutcomeRange(Number.parseInt(requestedRange, 10) as 7 | 30 | 90)
    }
    void store.selectAgent(route.location.value.selection.agent ?? '')
  } else if (selectedAgentID.value) {
    void store.selectAgent('')
  }
  if (route.location.value.name === 'activity') {
    const selection = route.location.value.selection
    void store.configureActivity(
      {
        project: selection.project,
        task: selection.task,
        kind: selection.kind,
        actor: selection.actor,
        state: selection.event_state ?? 'all',
        range: selection.range ?? '7d',
        cursor: selection.cursor,
      },
      fetch,
      refreshActivity,
    )
  }
}

function updateProject(slug: string): void {
  void store.selectProject(slug)
  const selection = { ...route.location.value.selection, project: slug }
  delete selection.task
  route.replaceSelection(selection)
}

function updateOutcomeRange(days: 7 | 30 | 90): void {
  const selection = {
    ...route.location.value.selection,
    outcome_range: `${days}d` as '7d' | '30d' | '90d',
  }
  delete selection.day
  delete selection.metric
  route.replaceSelection(selection)
  void store.setOutcomeRange(days)
}

function inspectTask(task: string, trigger?: HTMLElement): void {
  if (trigger) taskReturnTarget.value = trigger
  void store.selectTask(task)
  route.pushSelection({ ...route.location.value.selection, task })
}

function closeTask(): void {
  if (taskReturnTarget.value && selectedTaskName.value) {
    window.history.back()
    return
  }
  void store.selectTask('')
  const selection = { ...route.location.value.selection }
  delete selection.task
  route.replaceSelection(selection)
  void nextTick(() => document.querySelector<HTMLElement>('#route-heading')?.focus())
}

function updateObservation(selection: Parameters<typeof route.replaceSelection>[0]): void {
  route.replaceSelection(selection)
}

function inspectRole(name: string, trigger: HTMLElement): void {
  roleReturnTarget.value = trigger
  route.pushSelection({ ...route.location.value.selection, role: name })
}

function inspectAgent(id: string, trigger?: HTMLElement): void {
  if (trigger) agentReturnTarget.value = trigger
  void store.selectAgent(id)
  route.pushSelection({ ...route.location.value.selection, agent: id })
}

function selectionWithoutAgent() {
  const selection = { ...route.location.value.selection }
  delete selection.agent
  return selection
}

function closeAgent(): void {
  if (agentReturnTarget.value && selectedAgentName.value) {
    window.history.back()
    return
  }
  void store.selectAgent('')
  route.replaceSelection(selectionWithoutAgent())
  void nextTick(() => document.querySelector<HTMLElement>('#route-heading')?.focus())
}

function selectionWithoutRole() {
  const selection = { ...route.location.value.selection }
  delete selection.role
  return selection
}

function closeRole(): void {
  if (roleReturnTarget.value && selectedRoleName.value) {
    window.history.back()
    return
  }
  route.replaceSelection(selectionWithoutRole())
  void nextTick(() => document.querySelector<HTMLElement>('#route-heading')?.focus())
}

watch(
  () => route.location.value.name,
  async (name, previous) => {
    if (name === 'activity') {
      applyRouteSelection(false)
      store.activateRoute(name)
    } else {
      store.activateRoute(name)
      applyRouteSelection()
    }
    if (previous !== undefined) {
      await nextTick()
      document.querySelector<HTMLElement>('#route-heading')?.focus()
    }
  },
)

watch(
  () => route.location.value.selection.project,
  () => applyRouteSelection(),
)

watch(
  () => route.location.value.selection.task,
  () => applyRouteSelection(),
)

watch(
  () => route.location.value.selection.agent,
  () => applyRouteSelection(),
)

watch(
  () => [
    route.location.value.selection.kind,
    route.location.value.selection.actor,
    route.location.value.selection.event_state,
    route.location.value.selection.cursor,
    route.location.value.selection.range,
  ],
  () => {
    if (route.location.value.name !== 'activity') return
    const selection = route.location.value.selection
    void store.configureActivity({
      project: selection.project,
      task: selection.task,
      kind: selection.kind,
      actor: selection.actor,
      state: selection.event_state ?? 'all',
      range: selection.range ?? '7d',
      cursor: selection.cursor,
    })
  },
)

watch(
  () => route.location.value.selection.live,
  (live) => store.setPaused(live === 'paused'),
)

watch(selectedSlug, (slug) => {
  const name = route.location.value.name
  if (
    (name === 'work' || name === 'delivery') &&
    slug &&
    route.location.value.selection.project !== slug
  ) {
    route.replaceSelection({ ...route.location.value.selection, project: slug })
  }
})

watch(selectedRoleName, async (name, previous) => {
  if (name || !previous) return
  await nextTick()
  const target = roleReturnTarget.value
  if (target?.isConnected) target.focus()
  else document.querySelector<HTMLElement>('#route-heading')?.focus()
  roleReturnTarget.value = null
})

watch(selectedAgentName, async (name, previous) => {
  if (name || !previous) return
  await nextTick()
  const target = agentReturnTarget.value
  if (target?.isConnected) target.focus()
  else document.querySelector<HTMLElement>('#route-heading')?.focus()
  agentReturnTarget.value = null
})

watch(selectedTaskName, async (name, previous) => {
  if (name || !previous) return
  await nextTick()
  const target = taskReturnTarget.value
  if (target?.isConnected) target.focus()
  else document.querySelector<HTMLElement>('#route-heading')?.focus()
  taskReturnTarget.value = null
})

onMounted(() => {
  applyRouteSelection(false)
  store.setPaused(route.location.value.selection.live === 'paused')
  store.start(fetch, route.location.value.name)
})
onUnmounted(() => store.stop())
</script>

<template>
  <a
    href="#dashboard-main"
    class="fixed top-3 left-3 z-50 -translate-y-20 rounded-sm bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground focus:translate-y-0"
    >Skip to dashboard</a
  >
  <div class="dashboard-shell mx-auto max-w-[1340px]">
    <AppHeader
      :phase="phase"
      :generated="generated"
      :pending-events="pendingEvents"
      :error="error"
      @retry="store.retryCurrent()"
    />
    <div class="dashboard-workspace">
      <SectionNav
        :current="route.location.value.name"
        :selection="route.location.value.selection"
      />

      <main id="dashboard-main" class="min-w-0">
        <RouteNotFound
          v-if="route.location.value.name === 'unknown'"
          :path="route.location.value.path"
        />

        <template v-else-if="route.current.value">
          <RouteIntro
            :eyebrow="route.current.value.eyebrow"
            :title="route.current.value.label"
            :description="route.current.value.description"
          />

          <ObservabilityToolbar
            :route="route.location.value.name"
            :selection="route.location.value.selection"
            :projects="projects"
            :roles="roles"
            :agents="agents"
            :generated="generated"
            :result-label="observationResult"
            :compact="route.location.value.name === 'activity'"
            @change="updateObservation"
          />

          <div
            v-if="route.location.value.invalidSelection"
            role="alert"
            class="mt-4 rounded-md border border-warning/40 bg-card px-4 py-3 text-xs text-warning"
          >
            An unsafe or malformed selection was ignored. The route itself remains available.
          </div>

          <div v-if="route.location.value.name === 'overview'" class="mt-6 space-y-6">
            <OperatorPulse
              v-if="overviewReady && overviewSurface.data"
              :overview="overviewSurface.data"
              :projects="projects"
              :delivery-attention="deliveryAttentionSurface.data?.item"
            />
            <SkeletonBlock
              v-else-if="overviewLoading"
              height="260px"
              aria-label="Loading operator pulse"
            />
            <ErrorPanel
              v-else
              :message="`couldn't assemble the operator pulse — ${error ?? 'unknown error'}`"
              @retry="store.retryCurrent()"
            />
            <OverviewSection
              :projects="filteredProjects"
              :phase="projectsSurface.phase"
              :has-snapshot="projectsSurface.lastOk !== null"
              :error="projectsSurface.error"
              @retry="store.pollProjects()"
            />
          </div>

          <div v-else-if="route.location.value.name === 'work'" class="mt-6">
            <BoardSection
              :projects="projects"
              :selected-slug="selectedSlug"
              :phase="projectsSurface.phase"
              :has-snapshot="projectsSurface.lastOk !== null"
              :error="projectsSurface.error"
              :tasks="tasksSurface.data"
              :tasks-phase="tasksSurface.phase"
              :tasks-has-snapshot="tasksSurface.lastOk !== null"
              :tasks-error="tasksSurface.error"
              :query="route.location.value.selection.q"
              :day="route.location.value.selection.day"
              @update:selected-slug="updateProject"
              @retry="store.pollProjects()"
              @retry-tasks="store.pollTasks()"
              @inspect="inspectTask"
            />
          </div>

          <div v-else-if="route.location.value.name === 'agents'" class="mt-6 space-y-6">
            <LoopOperations
              v-if="operationSurface.lastOk !== null && operationSurface.data"
              :operation="operationSurface.data"
            />
            <SkeletonBlock
              v-else-if="operationSurface.phase === 'loading'"
              height="420px"
              aria-label="Loading loop operations"
            />
            <ErrorPanel
              v-else
              :message="`couldn't load loop operations — ${operationSurface.error ?? 'unknown error'}`"
              @retry="store.pollOperation()"
            />
            <OutcomeAnalytics
              v-if="outcomeSurface.lastOk !== null && outcomeSurface.data"
              :analytics="outcomeSurface.data"
              :range="outcomeRange"
              :stale="outcomeSurface.phase === 'error'"
              :focus-metric="route.location.value.selection.metric"
              :focus-day="route.location.value.selection.day"
              @range="updateOutcomeRange"
            />
            <SkeletonBlock
              v-else-if="outcomeSurface.phase === 'loading'"
              height="420px"
              aria-label="Loading outcome analytics"
            />
            <ErrorPanel
              v-else
              :message="`couldn't load outcome analytics — ${outcomeSurface.error ?? 'unknown error'}`"
              @retry="store.pollOutcomes()"
            />
            <BurnRate
              v-if="burnSurface.lastOk !== null"
              :burn="filteredBurn"
              :project="selectedSlug"
              :generated="burnSurface.generated"
              :stale="burnSurface.phase === 'error'"
              :focus-day="route.location.value.selection.burn_day"
            />
            <SkeletonBlock v-else-if="burnSurface.phase === 'loading'" height="140px" />
            <ErrorPanel
              v-else
              :message="`couldn't load burn rate — ${burnSurface.error ?? 'unknown error'}`"
              @retry="store.pollBurn()"
            />
            <AgentSwarmSection
              :agents="filteredAgents"
              :phase="agentsSurface.phase"
              :has-snapshot="agentsSurface.lastOk !== null"
              :error="agentsSurface.error"
              @retry="store.pollAgents()"
              @inspect="inspectAgent"
            />
          </div>

          <div v-else-if="route.location.value.name === 'team'" class="mt-6 min-w-0">
            <RoleRosterSection
              :roles="filteredRoles"
              :phase="rolesSurface.phase"
              :has-snapshot="rolesSurface.lastOk !== null"
              :error="rolesSurface.error"
              @retry="store.pollRoles()"
              @inspect="inspectRole"
            />
          </div>

          <div v-else-if="route.location.value.name === 'activity'" class="mt-6">
            <ActivitySection
              :activity="activitySurface.data"
              :phase="activitySurface.phase"
              :has-snapshot="activitySurface.lastOk !== null"
              :error="activitySurface.error"
              :selection="route.location.value.selection"
              :projects="projects"
              @change="updateObservation"
              @retry="store.pollActivity()"
            />
          </div>

          <div v-else-if="route.location.value.name === 'delivery'" class="mt-6 space-y-6">
            <DeliveryTimeline
              :tasks="graphSurface.data.nodes"
              :selected-task="selectedTaskRef"
              :timeline="timelineSurface.data"
              @select="inspectTask"
            />
            <DagSection
              :projects="projects"
              :selected-slug="selectedSlug"
              :phase="graphSurface.phase"
              :has-snapshot="graphSurface.lastOk !== null"
              :error="graphSurface.error"
              :graph-mode="store.graphMode"
              :graph-statuses="store.graphStatuses"
              :graph-focus="store.graphFocus"
              :graph-page="store.graphPage"
              @update:selected-slug="updateProject"
              @retry="store.pollGraph()"
              @inspect="inspectTask"
              @query="store.setGraphQuery($event)"
            />
          </div>
        </template>
      </main>
    </div>

    <RoleInspector
      :open="Boolean(selectedRoleName)"
      :selected-name="selectedRoleName"
      :role="selectedRole"
      :roles-phase="rolesSurface.phase"
      :roles-has-snapshot="rolesSurface.lastOk !== null"
      :agents="agents"
      :agents-phase="agentsSurface.phase"
      :agents-has-snapshot="agentsSurface.lastOk !== null"
      :agents-error="agentsSurface.error"
      @close="closeRole"
    />

    <AgentInspector
      :open="Boolean(selectedAgentName)"
      :selected-i-d="selectedAgentName"
      :agent="agentDetailSurface.data"
      :phase="agentDetailSurface.phase"
      :has-snapshot="agentDetailSurface.lastOk !== null"
      :error="agentDetailSurface.error"
      :status="agentDetailSurface.status"
      :live="selectedAgentIsLive"
      @close="closeAgent"
      @retry="store.pollAgentDetail()"
      @navigate-agent="(agent) => inspectAgent(agent)"
    />

    <TaskInspector
      :open="Boolean(selectedTaskName) && route.location.value.name === 'work'"
      :selected-ref="selectedTaskName"
      :task="taskDetailSurface.data"
      :phase="taskDetailSurface.phase"
      :has-snapshot="taskDetailSurface.lastOk !== null"
      :error="taskDetailSurface.error"
      :status="taskDetailSurface.status"
      :events="taskEventsSurface.data"
      :events-phase="taskEventsSurface.phase"
      :events-has-snapshot="taskEventsSurface.lastOk !== null"
      :events-error="taskEventsSurface.error"
      @close="closeTask"
      @retry="store.pollTaskDetail()"
      @navigate-task="(task) => inspectTask(task)"
    />

    <footer
      class="mt-12 flex flex-wrap justify-between gap-3 border-t border-border py-5 font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground"
    >
      <span>one durable workspace</span><span>projection only · no mutation authority</span>
    </footer>
  </div>
</template>

<style scoped>
.dashboard-workspace {
  margin-top: 0.75rem;
}

@media (min-width: 1024px) {
  .dashboard-workspace {
    display: grid;
    grid-template-columns: 104px minmax(0, 1fr);
    gap: 1.1rem;
    align-items: start;
  }
}
</style>
