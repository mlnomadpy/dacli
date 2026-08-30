<script setup lang="ts">
import BrandMark from '@/components/BrandMark.vue'
import ConnectionStatus from '@/components/ConnectionStatus.vue'
import type { Phase } from '@/types'

// The page masthead: the one <h1>, the decorative mark, and the always-visible
// ConnectionStatus tell (DESIGN.md §4–§5). Pure — it forwards Retry up to App,
// the sole owner of the network.
defineProps<{
  phase: Phase
  generated: string | null
  pendingEvents: number
  error: string | null
}>()
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <header class="flex flex-wrap items-end justify-between gap-x-8 gap-y-4 pt-2">
    <div class="min-w-0">
      <p
        class="m-0 mb-2 flex items-center gap-2 font-mono text-[10px] font-medium uppercase tracking-[0.14em] text-primary"
      >
        <BrandMark /> dacli / workspace record
      </p>
      <h1
        class="m-0 text-[clamp(1.65rem,3vw,2.6rem)] leading-none font-semibold tracking-[-0.045em]"
      >
        Delivery control
      </h1>
      <p class="mt-2 mb-0 max-w-2xl text-sm text-muted-foreground">
        What is moving, what needs attention, and what the governed loop can do next.
      </p>
    </div>
    <ConnectionStatus
      :phase="phase"
      :generated="generated"
      :pending-events="pendingEvents"
      :error="error"
      @retry="emit('retry')"
    />
  </header>
</template>
