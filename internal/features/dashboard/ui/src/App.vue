<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import AppHeader from '@/components/AppHeader.vue'
import OverviewSection from '@/components/OverviewSection.vue'
import BurnRate from '@/components/BurnRate.vue'
import BoardSection from '@/components/BoardSection.vue'
import DagSection from '@/components/DagSection.vue'
import AgentSwarmSection from '@/components/AgentSwarmSection.vue'
import RoleRosterSection from '@/components/RoleRosterSection.vue'
import RoleInspector from '@/components/RoleInspector.vue'
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
  agentsSurface,
  rolesSurface,
  burnSurface,
  graphSurface,
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
const filteredProjects = computed(() =>
  filterProjects(projects.value, route.location.value.selection),
)
const filteredAgents = computed(() => filterAgents(agents.value, route.location.value.selection))
const filteredRoles = computed(() => filterRoles(roles.value, route.location.value.selection))
const filteredBurn = computed(() => filterBurn(burn.value, route.location.value.selection))
const observationResult = computed(() => {
  switch (route.location.value.name) {
    case 'overview':
      return `${filteredProjects.value.length} of ${projects.value.length} projects`
    case 'work':
    case 'delivery':
      return selectedSlug.value
        ? `1 of ${projects.value.length} projects · ${selectedSlug.value}`
        : `0 of ${projects.value.length} projects`
    case 'agents':
      return `${filteredAgents.value.length} of ${agents.value.length} live agents`
    case 'team':
      return `${filteredRoles.value.length} of ${roles.value.length} roles`
    case 'activity':
      return `${pendingEvents.value} pending events`
    default:
      return 'No observable route'
  }
})

function applyRouteSelection(): void {
  const project = route.location.value.selection.project
  if (project && project !== selectedSlug.value) void store.selectProject(project)
}

function updateProject(slug: string): void {
  void store.selectProject(slug)
  route.replaceSelection({ ...route.location.value.selection, project: slug })
}

function updateObservation(selection: Parameters<typeof route.replaceSelection>[0]): void {
  route.replaceSelection(selection)
}

function inspectRole(name: string, trigger: HTMLElement): void {
  roleReturnTarget.value = trigger
  route.pushSelection({ ...route.location.value.selection, role: name })
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
    store.activateRoute(name)
    applyRouteSelection()
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

onMounted(() => {
  applyRouteSelection()
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
    <SectionNav :current="route.location.value.name" :selection="route.location.value.selection" />

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
            @update:selected-slug="updateProject"
            @retry="store.pollProjects()"
          />
        </div>

        <div v-else-if="route.location.value.name === 'agents'" class="mt-6 space-y-6">
          <BurnRate v-if="burnSurface.lastOk !== null" :burn="filteredBurn" />
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
          <ActivitySection :pending-events="pendingEvents" />
        </div>

        <div v-else-if="route.location.value.name === 'delivery'" class="mt-6">
          <DagSection
            :projects="projects"
            :selected-slug="selectedSlug"
            :phase="graphSurface.phase"
            :has-snapshot="graphSurface.lastOk !== null"
            :error="graphSurface.error"
            @update:selected-slug="updateProject"
            @retry="store.pollGraph()"
          />
        </div>
      </template>
    </main>

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

    <footer
      class="mt-12 flex flex-wrap justify-between gap-3 border-t border-border py-5 font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground"
    >
      <span>one durable workspace</span><span>projection only · no mutation authority</span>
    </footer>
  </div>
</template>
