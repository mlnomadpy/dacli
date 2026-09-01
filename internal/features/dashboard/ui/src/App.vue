<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, watch } from 'vue'
import { storeToRefs } from 'pinia'
import AppHeader from '@/components/AppHeader.vue'
import OverviewSection from '@/components/OverviewSection.vue'
import BurnRate from '@/components/BurnRate.vue'
import BoardSection from '@/components/BoardSection.vue'
import DagSection from '@/components/DagSection.vue'
import AgentSwarmSection from '@/components/AgentSwarmSection.vue'
import RoleRosterSection from '@/components/RoleRosterSection.vue'
import OperatorPulse from '@/components/OperatorPulse.vue'
import SectionNav from '@/components/SectionNav.vue'
import RouteIntro from '@/components/RouteIntro.vue'
import RouteNotFound from '@/components/RouteNotFound.vue'
import ActivitySection from '@/components/ActivitySection.vue'
import ErrorPanel from '@/components/ErrorPanel.vue'
import SkeletonBlock from '@/components/SkeletonBlock.vue'
import { useDashboardRoute } from '@/composables/useDashboardRoute'
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

function applyRouteSelection(): void {
  const project = route.location.value.selection.project
  if (project && project !== selectedSlug.value) void store.selectProject(project)
}

function updateProject(slug: string): void {
  void store.selectProject(slug)
  route.replaceSelection({ ...route.location.value.selection, project: slug })
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

onMounted(() => {
  applyRouteSelection()
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
    <SectionNav :current="route.location.value.name" :selected-project="selectedSlug" />

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
            :projects="projects"
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
          <BurnRate v-if="burnSurface.lastOk !== null" :burn="burn" />
          <SkeletonBlock v-else-if="burnSurface.phase === 'loading'" height="140px" />
          <ErrorPanel
            v-else
            :message="`couldn't load burn rate — ${burnSurface.error ?? 'unknown error'}`"
            @retry="store.pollBurn()"
          />
          <AgentSwarmSection
            :agents="agents"
            :phase="agentsSurface.phase"
            :has-snapshot="agentsSurface.lastOk !== null"
            :error="agentsSurface.error"
            @retry="store.pollAgents()"
          />
        </div>

        <div v-else-if="route.location.value.name === 'team'" class="mt-6 min-w-0">
          <RoleRosterSection
            :roles="roles"
            :phase="rolesSurface.phase"
            :has-snapshot="rolesSurface.lastOk !== null"
            :error="rolesSurface.error"
            @retry="store.pollRoles()"
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

    <footer
      class="mt-12 flex flex-wrap justify-between gap-3 border-t border-border py-5 font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground"
    >
      <span>one durable workspace</span><span>projection only · no mutation authority</span>
    </footer>
  </div>
</template>
