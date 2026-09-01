<script setup lang="ts">
import BrandMark from '@/components/BrandMark.vue'
import ConnectionStatus from '@/components/ConnectionStatus.vue'
import type { Phase } from '@/types'

// Compact command bar: brand, product identity and connection truth occupy one
// 56px band so route evidence begins near the viewport top (issue #953).
defineProps<{
  phase: Phase
  generated: string | null
  pendingEvents: number
  error: string | null
}>()
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <header
    class="app-header flex min-h-14 flex-wrap items-center justify-between gap-x-6 gap-y-2 border-b border-border pb-2"
  >
    <div class="flex min-w-0 items-center gap-3">
      <p
        class="m-0 flex shrink-0 items-center gap-2 font-mono text-[9px] font-semibold uppercase tracking-[0.14em] text-primary"
      >
        <BrandMark /> dacli
      </p>
      <span class="h-5 w-px bg-border" aria-hidden="true" />
      <div class="min-w-0 md:flex md:items-baseline md:gap-3">
        <h1 class="m-0 text-lg leading-none font-semibold tracking-[-0.035em]">Delivery control</h1>
        <p class="mt-1 mb-0 truncate text-[11px] text-muted-foreground md:m-0">
          governed workspace record
        </p>
      </div>
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
