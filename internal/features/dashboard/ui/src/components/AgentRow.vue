<script setup lang="ts">
import { computed } from 'vue'
import type { Agent } from '@/types'
import { ago, duration, freshness } from '@/composables/useRelativeTime'

// One live agent row (DESIGN.md §7.3). The activity dot's color encodes the
// freshness of `last_activity` — fresh (<60s, pulsing) / idle (<5m) / stale
// (older, static) — "still moving vs. possibly hung," the swarm's whole point.
// Its meaning is ALSO available as text (the last-activity column + the dot's
// title), so color is never the only signal.
const props = defineProps<{ agent: Agent }>()

const fresh = computed(() => freshness(props.agent.last_activity))
const dotTitle = computed(() => {
  switch (fresh.value) {
    case 'fresh':
      return 'active <60s'
    case 'idle':
      return 'idle <5m'
    default:
      return 'stale — possibly hung'
  }
})
</script>

<template>
  <tr>
    <td class="run mono">
      <i class="dot" :class="fresh" :title="dotTitle" />
      {{ agent.run_id.slice(0, 10) }}
    </td>
    <td>{{ agent.child || '—' }}</td>
    <td class="task">{{ agent.task || '—' }}</td>
    <td>{{ agent.role || '—' }}</td>
    <td>{{ agent.runtime || '—' }}</td>
    <td class="mono">{{ agent.pid || '—' }}</td>
    <td>{{ duration(agent.runtime_secs) }}</td>
    <td :title="agent.last_activity">{{ ago(agent.last_activity) }}</td>
  </tr>
</template>

<style scoped>
td {
  text-align: left;
  padding: 8px 12px;
  font-size: 12px;
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
  margin-right: 6px;
  flex: none;
  background: var(--muted);
}
.dot.fresh {
  background: var(--ok);
  animation: pulse 2s infinite;
}
.dot.idle {
  background: var(--muted);
}
.dot.stale {
  background: var(--blocked);
}
/* The identifying `run` column stays pinned while the metrics scroll on mobile
 * (DESIGN.md §9). Only one column can hold left:0; `run` is the primary id. */
.run {
  position: sticky;
  left: 0;
  background: var(--panel);
}
</style>
