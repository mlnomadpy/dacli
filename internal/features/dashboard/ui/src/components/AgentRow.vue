<script setup lang="ts">
import { computed } from 'vue'
import type { Agent } from '@/types'
import { ago, duration, freshness } from '@/composables/useRelativeTime'

// One live agent row (DESIGN.md §7.3). The activity dot's color encodes the
// freshness of `last_activity` — fresh (<60s, pulsing) / idle (<5m) / stale
// (older, static) — "still moving vs. possibly hung," the swarm's whole point.
// Its meaning is ALSO available as text (the last-activity column + the dot's
// title), so color is never the only signal.
//
// The state badge is the server's HONEST verdict on what the agent is doing,
// derived from the transcript (thinking | acting | waiting | stalled). It is
// distinct from the freshness dot: the dot is pure recency, the badge is what
// the transcript actually shows. The badge word is always the label, so its
// color is decorative, never the only signal. The transcript / diff links are
// read-only views of the same run — the adopter's "presence vs. artifact" fix.
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
const stateTitle = computed(() => {
  switch (props.agent.state) {
    case 'thinking':
      return 'reasoning — last transcript line is prose'
    case 'acting':
      return 'running a tool — last transcript line is a [tool: X] marker'
    case 'waiting':
      return 'no transcript output yet — fresh spawn or a runtime that buffers to exit'
    case 'stalled':
      return 'transcript frozen while alive — possibly hung'
    default:
      return props.agent.state
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
    <td>
      <span class="badge" :class="agent.state" :title="stateTitle">{{ agent.state }}</span>
    </td>
    <td class="mono">{{ agent.pid || '—' }}</td>
    <td>{{ duration(agent.runtime_secs) }}</td>
    <td :title="agent.last_activity">{{ ago(agent.last_activity) }}</td>
    <td class="links">
      <a :href="agent.transcript_url" target="_blank" rel="noopener">transcript</a>
      <a :href="agent.diff_url" target="_blank" rel="noopener">diff</a>
    </td>
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
/* State badge: a bordered pill whose word is always shown (color is decorative,
 * never the only signal). `thinking` reads active-blue, `acting` done-green,
 * `waiting` muted, `stalled` danger-red. */
.badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  border: 1px solid var(--border);
  color: var(--muted);
}
.badge.thinking {
  color: var(--active);
  border-color: var(--active);
}
.badge.acting {
  color: var(--ok);
  border-color: var(--ok);
}
.badge.waiting {
  color: var(--muted);
  border-color: var(--muted);
}
.badge.stalled {
  color: var(--blocked);
  border-color: var(--blocked);
}
.links {
  display: flex;
  gap: 10px;
}
.links a {
  color: var(--accent);
  text-decoration: none;
  font-size: 12px;
}
.links a:hover {
  text-decoration: underline;
}
/* The identifying `run` column stays pinned while the metrics scroll on mobile
 * (DESIGN.md §9). Only one column can hold left:0; `run` is the primary id. */
.run {
  position: sticky;
  left: 0;
  background: var(--panel);
}
</style>
