<script setup lang="ts">
import type { Role } from '@/types'

const props = defineProps<{ role: Role }>()
const emit = defineEmits<{ inspect: [name: string, trigger: HTMLElement] }>()

function inspect(event: MouseEvent): void {
  emit('inspect', props.role.name, event.currentTarget as HTMLElement)
}
</script>

<template>
  <article class="rounded-lg border border-border bg-card p-4" :aria-label="`Role ${role.name}`">
    <header class="flex items-start justify-between gap-3">
      <div class="min-w-0">
        <p class="m-0 font-mono text-[10px] uppercase tracking-[0.1em] text-primary">
          {{ role.kind || 'any phase' }} · {{ role.grant || 'ro' }}
        </p>
        <h3 class="mt-1 mb-0 truncate text-base font-semibold">{{ role.name }}</h3>
        <p class="mt-1 mb-0 text-xs text-muted-foreground">
          {{ role.summary || 'No summary recorded.' }}
        </p>
      </div>
      <span
        :class="role.wip_exceeded ? 'text-destructive' : 'text-muted-foreground'"
        class="font-mono text-xs"
      >
        {{ role.active_agents }}/{{ role.wip || '∞' }} WIP
      </span>
    </header>
    <dl class="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
      <div>
        <dt>Runtime</dt>
        <dd>{{ role.runtime || 'default' }}</dd>
      </div>
      <div>
        <dt>Model</dt>
        <dd>{{ role.model || 'default' }}</dd>
      </div>
      <div class="col-span-2">
        <dt>Scope</dt>
        <dd class="truncate" :title="role.scope.join(', ')">
          {{ role.scope.join(', ') || 'everywhere' }}
        </dd>
      </div>
    </dl>
    <button type="button" :aria-label="`Inspect ${role.name}`" @click="inspect">
      Inspect role
    </button>
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
}
button {
  width: 100%;
  min-height: 44px;
  margin-top: 1rem;
  border: 1px solid var(--border);
  border-radius: 0.35rem;
  background: var(--background);
  color: var(--foreground);
  font-size: 0.72rem;
  font-weight: 650;
}
</style>
