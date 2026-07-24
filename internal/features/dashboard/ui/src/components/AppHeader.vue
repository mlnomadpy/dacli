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
  <header class="app-header">
    <div class="title">
      <h1><BrandMark /> dacli dashboard</h1>
      <p class="tagline">mission control — the live agent swarm</p>
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

<style scoped>
.app-header {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px 24px;
  margin-bottom: 8px;
}
.title {
  min-width: 0;
}
h1 {
  font-size: 18px;
  font-weight: 600;
  margin: 0 0 4px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.tagline {
  color: var(--muted);
  font-size: 12px;
  margin: 0;
}
</style>
