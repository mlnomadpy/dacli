<script setup lang="ts">
import { computed } from 'vue'
import type { Phase } from '@/types'

// The always-visible header tell (DESIGN.md §6.2). Pure: props in, no fetching.
// An aria-live="polite" region so a screen reader announces phase transitions
// without hijacking focus (DESIGN.md §8).
const props = defineProps<{
  phase: Phase
  generated: string | null
  pendingEvents: number
  error: string | null
}>()

const emit = defineEmits<{ retry: [] }>()

const clock = computed(() => {
  if (!props.generated) return ''
  const d = new Date(props.generated)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleTimeString()
})

const label = computed(() => {
  switch (props.phase) {
    case 'loading':
      return 'connecting…'
    case 'live':
      return `live · updated ${clock.value}`
    case 'error':
      return `connection lost: ${props.error ?? 'unknown error'}`
  }
  return ''
})
</script>

<template>
  <div class="status" role="status" aria-live="polite">
    <span class="dot" :class="phase" :title="label" />
    <span class="text">{{ label }}</span>
    <span v-if="phase === 'live' && pendingEvents > 0" class="pill">
      · {{ pendingEvents }} pending event{{ pendingEvents === 1 ? '' : 's' }}
    </span>
    <button v-if="phase === 'error'" type="button" class="retry" @click="emit('retry')">Retry</button>
  </div>
</template>

<style scoped>
.status {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--muted);
  font-size: 12px;
}
.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  display: inline-block;
  background: var(--muted);
}
.dot.live {
  background: var(--ok);
  animation: pulse 2s infinite;
}
.dot.error {
  background: var(--danger);
}
.pill {
  color: var(--muted);
}
.retry {
  margin-left: 8px;
  min-height: 24px;
  padding: 2px 8px;
  font: inherit;
  color: var(--text);
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 4px;
  cursor: pointer;
}
.retry:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}
</style>
