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
  <header class="mb-2 flex flex-wrap items-start justify-between gap-x-6 gap-y-3">
    <div class="min-w-0">
      <h1 class="m-0 mb-1 flex items-center gap-2 text-lg font-semibold">
        <BrandMark /> dacli dashboard
      </h1>
      <p class="m-0 text-xs text-muted-foreground">mission control — the live agent swarm</p>
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
