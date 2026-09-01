<script setup lang="ts">
import type { Agent } from '@/types'
import { ago, duration } from '@/composables/useRelativeTime'

defineProps<{ agent: Agent }>()
const emit = defineEmits<{ inspect: [agent: string, trigger: HTMLElement] }>()
</script>

<template>
  <article
    class="rounded-lg border border-border bg-card p-4"
    :aria-label="`Agent run ${agent.run_id}`"
  >
    <header class="flex items-start justify-between gap-3">
      <div class="min-w-0">
        <p class="m-0 font-mono text-[10px] uppercase tracking-[0.1em] text-primary">
          run {{ agent.run_id.slice(0, 10) }}
        </p>
        <h3 class="mt-1 mb-0 truncate text-sm font-semibold">
          {{ agent.role || 'unassigned role' }} · task {{ agent.task || '—' }}
        </h3>
      </div>
      <span
        class="rounded-full border border-border px-2 py-0.5 text-[10px] font-semibold uppercase"
      >
        {{ agent.state }}
      </span>
    </header>
    <dl class="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
      <div>
        <dt>Runtime</dt>
        <dd>{{ agent.runtime || '—' }}</dd>
      </div>
      <div>
        <dt>Child</dt>
        <dd>{{ agent.child || '—' }}</dd>
      </div>
      <div>
        <dt>Uptime</dt>
        <dd>{{ duration(agent.runtime_secs) }}</dd>
      </div>
      <div>
        <dt>Last activity</dt>
        <dd :title="agent.last_activity">{{ ago(agent.last_activity) }}</dd>
      </div>
    </dl>
    <div class="mt-4 grid grid-cols-3 gap-2">
      <button
        type="button"
        :aria-label="`Inspect agent ${agent.child}`"
        @click="emit('inspect', agent.child, $event.currentTarget as HTMLElement)"
      >
        Inspect agent
      </button>
      <a :href="agent.transcript_url" target="_blank" rel="noopener">Inspect transcript</a>
      <a :href="agent.diff_url" target="_blank" rel="noopener">Inspect diff</a>
    </div>
  </article>
</template>

<style scoped>
dt {
  color: var(--muted-foreground);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.56rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}
dd {
  margin: 0.12rem 0 0;
  color: var(--foreground);
}
a,
button {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  border-radius: 0.35rem;
  color: var(--primary);
  font-size: 0.68rem;
  font-weight: 650;
  text-decoration: none;
  background: transparent;
}
</style>
