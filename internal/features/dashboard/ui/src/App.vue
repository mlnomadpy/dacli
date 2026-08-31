<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
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
import ErrorPanel from '@/components/ErrorPanel.vue'
import SkeletonBlock from '@/components/SkeletonBlock.vue'
import { useDashboardStore } from '@/stores/dashboard'

// `App` is the ONLY thing that starts the network lifecycle. Each canonical
// surface now owns independent freshness/error state (issue #932), so a failed
// graph read cannot dim a healthy live-agent or roster projection.
const store = useDashboardStore()
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

const pulseHasSnapshot = computed(
  () =>
    overviewSurface.value.lastOk !== null &&
    projectsSurface.value.lastOk !== null &&
    agentsSurface.value.lastOk !== null &&
    rolesSurface.value.lastOk !== null &&
    burnSurface.value.lastOk !== null &&
    (projects.value.length === 0 || graphSurface.value.lastOk !== null),
)
const pulseLoading = computed(() =>
  [
    overviewSurface.value,
    projectsSurface.value,
    agentsSurface.value,
    rolesSurface.value,
    burnSurface.value,
    graphSurface.value,
  ].some((surface) => surface.phase === 'loading'),
)

onMounted(() => store.start())
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
      @retry="store.retryAll()"
    />
    <SectionNav />

    <main id="dashboard-main" class="space-y-10">
      <div id="pulse" class="scroll-mt-20 space-y-6" aria-label="Workspace pulse">
        <OperatorPulse
          v-if="pulseHasSnapshot"
          :projects="projects"
          :agents="agents"
          :roles="roles"
          :burn="burn"
          :pending-events="pendingEvents"
        />
        <SkeletonBlock
          v-else-if="pulseLoading"
          height="260px"
          aria-label="Loading operator pulse"
        />
        <ErrorPanel
          v-else
          :message="`couldn't assemble the operator pulse — ${error ?? 'unknown error'}`"
          @retry="store.retryAll()"
        />
        <OverviewSection
          :projects="projects"
          :phase="projectsSurface.phase"
          :has-snapshot="projectsSurface.lastOk !== null"
          :error="projectsSurface.error"
          @retry="store.pollProjects()"
        />
      </div>

      <div id="delivery" class="scroll-mt-20 space-y-6" aria-label="Delivery">
        <BoardSection
          :projects="projects"
          :selected-slug="selectedSlug"
          :phase="projectsSurface.phase"
          :has-snapshot="projectsSurface.lastOk !== null"
          :error="projectsSurface.error"
          @update:selected-slug="store.selectProject($event)"
          @retry="store.pollProjects()"
        />
        <DagSection
          :projects="projects"
          :selected-slug="selectedSlug"
          :phase="graphSurface.phase"
          :has-snapshot="graphSurface.lastOk !== null"
          :error="graphSurface.error"
          @update:selected-slug="store.selectProject($event)"
          @retry="store.pollGraph()"
        />
      </div>

      <div id="agents" class="scroll-mt-20 space-y-6" aria-label="Agents and spend">
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
      <!-- The roster sits below the swarm: the swarm answers "who is running
           now", the roster answers "who COULD run, and what may they touch"
           (dacli 226). -->
      <div id="team" class="scroll-mt-20" aria-label="Team">
        <RoleRosterSection
          :roles="roles"
          :phase="rolesSurface.phase"
          :has-snapshot="rolesSurface.lastOk !== null"
          :error="rolesSurface.error"
          @retry="store.pollRoles()"
        />
      </div>
    </main>

    <footer
      class="mt-12 flex flex-wrap justify-between gap-3 border-t border-border py-5 font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground"
    >
      <span>one durable workspace</span><span>projection only · no mutation authority</span>
    </footer>
  </div>
</template>
