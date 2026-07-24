<script setup lang="ts">
import type { Agent, Phase } from '@/types'
import AgentSwarm from '@/components/AgentSwarm.vue'

// The Live agent swarm section shell (DESIGN.md §4–§5, §7.3): a labelled
// <section> with a live-count in the header. The 2s poll (owned by the store)
// keeps this live — a running loop's agents appear without a restart.
const props = defineProps<{
  agents: Agent[]
  phase: Phase
  hasSnapshot: boolean
  error: string | null
}>()
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <section aria-labelledby="swarm-h">
    <div class="section-head">
      <h2 id="swarm-h">Live agent swarm</h2>
      <span v-if="props.agents.length > 0" class="live-count">
        <i class="dot" aria-hidden="true" />{{ props.agents.length }} running
      </span>
    </div>
    <AgentSwarm
      :agents="props.agents"
      :phase="props.phase"
      :has-snapshot="props.hasSnapshot"
      :error="props.error"
      @retry="emit('retry')"
    />
  </section>
</template>

<style scoped>
.section-head {
  display: flex;
  align-items: center;
  gap: 12px;
  justify-content: space-between;
  flex-wrap: wrap;
}
.live-count {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--muted);
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--ok);
  display: inline-block;
  animation: pulse 2s infinite;
}
</style>
