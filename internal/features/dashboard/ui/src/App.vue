<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
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
import { useDashboardStore } from '@/stores/dashboard'

// `App` is the ONLY thing that touches the network, via the Pinia store's poll
// loop (DESIGN.md §5). Data flows down as props; every section is pure and
// read-only. The store owns `phase`; a dropped poll keeps the last good snapshot
// on screen (dimmed) rather than blanking.
const store = useDashboardStore()
const { phase, error, projects, agents, roles, pendingEvents, generated, hasSnapshot, burn } =
  storeToRefs(store)

// Project selection for the Board section is client-only, not persisted
// (DESIGN.md §5). BoardSection falls back to the first project when this slug
// is absent, so a project disappearing between polls never renders a stale slug.
const selectedSlug = ref('')

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
      @retry="store.retry()"
    />
    <SectionNav />

    <!-- A stale-but-retained read is dimmed and inert, visually distinct from
         live — honesty about freshness, no fabricated data (DESIGN.md §6.2). -->
    <main
      id="dashboard-main"
      class="space-y-10"
      :class="{ 'pointer-events-none opacity-60': phase === 'error' && hasSnapshot }"
    >
      <div id="pulse" class="scroll-mt-20 space-y-6" aria-label="Workspace pulse">
        <OperatorPulse
          :projects="projects"
          :agents="agents"
          :roles="roles"
          :burn="burn"
          :pending-events="pendingEvents"
        />
        <OverviewSection
          :projects="projects"
          :phase="phase"
          :has-snapshot="hasSnapshot"
          :error="error"
          @retry="store.retry()"
        />
      </div>

      <div id="delivery" class="scroll-mt-20 space-y-6" aria-label="Delivery">
        <BoardSection
          :projects="projects"
          :selected-slug="selectedSlug"
          :phase="phase"
          :has-snapshot="hasSnapshot"
          :error="error"
          @update:selected-slug="selectedSlug = $event"
          @retry="store.retry()"
        />
        <DagSection
          :projects="projects"
          :selected-slug="selectedSlug"
          :phase="phase"
          @update:selected-slug="selectedSlug = $event"
        />
      </div>

      <div id="agents" class="scroll-mt-20 space-y-6" aria-label="Agents and spend">
        <BurnRate :burn="burn" />
        <AgentSwarmSection
          :agents="agents"
          :phase="phase"
          :has-snapshot="hasSnapshot"
          :error="error"
          @retry="store.retry()"
        />
      </div>
      <!-- The roster sits below the swarm: the swarm answers "who is running
           now", the roster answers "who COULD run, and what may they touch"
           (dacli 226). -->
      <div id="team" class="scroll-mt-20" aria-label="Team">
        <RoleRosterSection
          :roles="roles"
          :phase="phase"
          :has-snapshot="hasSnapshot"
          :error="error"
          @retry="store.retry()"
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
