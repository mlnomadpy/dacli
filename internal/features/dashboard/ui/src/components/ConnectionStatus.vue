<script setup lang="ts">
import { computed } from 'vue'
import type { Phase } from '@/types'
import { Button } from '@/components/ui/button'

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

const dotClass = computed(() => ({
  'live bg-success animate-[pulse_2s_infinite]': props.phase === 'live',
  'error bg-destructive': props.phase === 'error',
  'bg-muted-foreground': props.phase === 'loading',
}))
</script>

<template>
  <div
    class="flex items-center gap-1.5 text-xs text-muted-foreground"
    role="status"
    aria-live="polite"
  >
    <span class="dot inline-block size-[7px] rounded-full" :class="dotClass" :title="label" />
    <span>{{ label }}</span>
    <span v-if="phase === 'live' && pendingEvents > 0">
      · {{ pendingEvents }} pending event{{ pendingEvents === 1 ? '' : 's' }}
    </span>
    <Button
      v-if="phase === 'error'"
      variant="outline"
      size="sm"
      class="retry ml-2 h-6 px-2"
      @click="emit('retry')"
    >
      Retry
    </Button>
  </div>
</template>
