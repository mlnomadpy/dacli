<script setup lang="ts">
import { computed } from 'vue'
import type { Agent, Phase } from '@/types'
import { sectionState } from '@/composables/useSectionState'
import AgentRow from '@/components/AgentRow.vue'
import EmptyPanel from '@/components/EmptyPanel.vue'
import ErrorPanel from '@/components/ErrorPanel.vue'
import SkeletonBlock from '@/components/SkeletonBlock.vue'

// Owns the swarm's four states (DESIGN.md §7.3). Empty ("no live agents") is the
// common resting state and reads calm, never as an error. Live is a real
// <table> with header scope for screen readers; rows keep the server's
// newest-first order (no client sort in v1). The table is the one element
// allowed to scroll horizontally on mobile, inside its own wrapper.
const props = defineProps<{
  agents: Agent[]
  phase: Phase
  hasSnapshot: boolean
  error: string | null
}>()
const emit = defineEmits<{ retry: [] }>()

const state = computed(() =>
  sectionState(props.phase, props.hasSnapshot, props.agents.length === 0),
)
</script>

<template>
  <div v-if="state === 'loading'" class="skeleton-table" aria-hidden="true">
    <SkeletonBlock v-for="n in 4" :key="n" height="32px" />
  </div>
  <ErrorPanel
    v-else-if="state === 'error'"
    :message="`couldn't load agents — ${error ?? 'unknown error'}`"
    @retry="emit('retry')"
  />
  <EmptyPanel v-else-if="state === 'empty'">no live agents</EmptyPanel>
  <div v-else class="table-wrap">
    <table>
      <thead>
        <tr>
          <th scope="col" class="run-h">run</th>
          <th scope="col">child</th>
          <th scope="col">task</th>
          <th scope="col">role</th>
          <th scope="col">runtime</th>
          <th scope="col">pid</th>
          <th scope="col">uptime</th>
          <th scope="col">last activity</th>
        </tr>
      </thead>
      <tbody>
        <AgentRow v-for="a in agents" :key="a.run_id" :agent="a" />
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.table-wrap {
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: 8px;
}
table {
  width: 100%;
  border-collapse: collapse;
  background: var(--panel);
}
th {
  text-align: left;
  padding: 8px 12px;
  color: var(--muted);
  font-weight: 600;
  text-transform: uppercase;
  font-size: 10px;
  letter-spacing: 0.05em;
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}
.run-h {
  position: sticky;
  left: 0;
  background: var(--panel);
}
table :deep(tr:last-child td) {
  border-bottom: none;
}
.skeleton-table {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
</style>
