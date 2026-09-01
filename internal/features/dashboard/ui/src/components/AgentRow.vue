<script setup lang="ts">
import { computed } from 'vue'
import type { Agent } from '@/types'
import { ago, duration, freshness } from '@/composables/useRelativeTime'
import { TableCell, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'

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
const emit = defineEmits<{ inspect: [agent: string, trigger: HTMLElement] }>()

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
    case 'blocked':
      return "the agent's task has an outstanding `dacli ask` — waiting on a human"
    case 'silent':
      return 'a text runtime has stayed quiet past the stall window — worth a look'
    default:
      return props.agent.state
  }
})

// The activity dot's color bucket. The freshness value doubles as the marker
// class (`fresh`/`idle`/`stale`); the pulse is the one ambient animation, keyed
// to tokens.css so `prefers-reduced-motion` freezes it.
const dotClass = computed(() => ({
  'fresh bg-success animate-[pulse_2s_infinite]': fresh.value === 'fresh',
  'idle bg-muted-foreground': fresh.value === 'idle',
  'stale bg-destructive': fresh.value === 'stale',
}))

// State badge color. The word is always shown (color is decorative): `thinking`
// reads active-blue, `acting` done-green, `waiting` muted, and the three
// "needs a look" states (`stalled`/`blocked`/`silent`) share danger-red — the
// same grouping `dacli agents` uppercases for without needing --tail.
const badgeClass = computed(() => {
  switch (props.agent.state) {
    case 'thinking':
      return 'thinking border-primary text-primary'
    case 'acting':
      return 'acting border-success text-success'
    case 'stalled':
      return 'stalled border-destructive text-destructive'
    case 'blocked':
      return 'blocked border-destructive text-destructive'
    case 'silent':
      return 'silent border-destructive text-destructive'
    default:
      return 'waiting border-muted-foreground text-muted-foreground'
  }
})
</script>

<template>
  <TableRow>
    <TableCell class="run sticky left-0 bg-card font-mono text-xs">
      <i
        class="dot mr-1.5 inline-block size-2 shrink-0 rounded-full"
        :class="dotClass"
        :title="dotTitle"
      />
      {{ agent.run_id.slice(0, 10) }}
    </TableCell>
    <TableCell class="text-xs">{{ agent.child || '—' }}</TableCell>
    <TableCell class="text-xs">{{ agent.task || '—' }}</TableCell>
    <TableCell class="text-xs">{{ agent.role || '—' }}</TableCell>
    <TableCell class="text-xs">{{ agent.runtime || '—' }}</TableCell>
    <TableCell class="text-xs">
      <Badge
        variant="outline"
        class="badge rounded-full text-[10px] font-semibold uppercase tracking-[0.04em]"
        :class="badgeClass"
        :title="stateTitle"
        >{{ agent.state }}</Badge
      >
    </TableCell>
    <TableCell class="font-mono text-xs">{{ agent.pid || '—' }}</TableCell>
    <TableCell class="text-xs">{{ duration(agent.runtime_secs) }}</TableCell>
    <TableCell class="text-xs" :title="agent.last_activity">{{
      ago(agent.last_activity)
    }}</TableCell>
    <TableCell class="links flex items-center gap-2.5 text-xs">
      <button
        type="button"
        class="rounded-sm border border-border px-2 py-1 font-semibold text-foreground hover:bg-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        :aria-label="`Inspect agent ${agent.child}`"
        @click="emit('inspect', agent.child, $event.currentTarget as HTMLElement)"
      >
        inspect
      </button>
      <a
        class="text-primary no-underline hover:underline"
        :href="agent.transcript_url"
        target="_blank"
        rel="noopener"
        >transcript</a
      >
      <a
        class="text-primary no-underline hover:underline"
        :href="agent.diff_url"
        target="_blank"
        rel="noopener"
        >diff</a
      >
    </TableCell>
  </TableRow>
</template>
