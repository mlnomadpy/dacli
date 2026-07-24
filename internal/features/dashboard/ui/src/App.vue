<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import AppHeader from '@/components/AppHeader.vue'
import OverviewSection from '@/components/OverviewSection.vue'
import BurnRate from '@/components/BurnRate.vue'
import BoardSection from '@/components/BoardSection.vue'
import DagSection from '@/components/DagSection.vue'
import AgentSwarmSection from '@/components/AgentSwarmSection.vue'
import { useDashboardStore } from '@/stores/dashboard'

// `App` is the ONLY thing that touches the network, via the Pinia store's poll
// loop (DESIGN.md §5). Data flows down as props; every section is pure and
// read-only. The store owns `phase`; a dropped poll keeps the last good snapshot
// on screen (dimmed) rather than blanking.
const store = useDashboardStore()
const { phase, error, projects, agents, pendingEvents, generated, hasSnapshot, burn } =
  storeToRefs(store)

// Project selection for the Board section is client-only, not persisted
// (DESIGN.md §5). BoardSection falls back to the first project when this slug
// is absent, so a project disappearing between polls never renders a stale slug.
const selectedSlug = ref('')

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <div class="shell">
    <AppHeader
      :phase="phase"
      :generated="generated"
      :pending-events="pendingEvents"
      :error="error"
      @retry="store.retry()"
    />

    <main :class="{ stale: phase === 'error' && hasSnapshot }">
      <OverviewSection
        :projects="projects"
        :phase="phase"
        :has-snapshot="hasSnapshot"
        :error="error"
        @retry="store.retry()"
      />
      <BurnRate :burn="burn" />
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
      <AgentSwarmSection
        :agents="agents"
        :phase="phase"
        :has-snapshot="hasSnapshot"
        :error="error"
        @retry="store.retry()"
      />
    </main>
  </div>
</template>

<style scoped>
.shell {
  max-width: 1280px;
  margin: 0 auto;
}
/* A stale-but-retained read is dimmed and inert, visually distinct from live —
 * honesty about freshness, no fabricated data (DESIGN.md §6.2). */
main.stale {
  opacity: 0.6;
  pointer-events: none;
}
main :deep(section) {
  margin-top: 24px;
}
main :deep(h2) {
  font-size: 13px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--muted);
  font-weight: 600;
  margin: 0 0 12px;
}
</style>
