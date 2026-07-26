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
    <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
      <h2
        id="swarm-h"
        class="m-0 text-[13px] font-semibold uppercase tracking-[0.06em] text-muted-foreground"
      >
        Live agent swarm
      </h2>
      <span
        v-if="props.agents.length > 0"
        class="flex items-center gap-1.5 text-xs text-muted-foreground"
      >
        <i
          class="inline-block size-2 animate-[pulse_2s_infinite] rounded-full bg-success"
          aria-hidden="true"
        />{{ props.agents.length }}
        running
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
