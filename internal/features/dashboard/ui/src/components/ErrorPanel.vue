<script setup lang="ts">
// The shared COLD-error panel (DESIGN.md §7): danger-tinted, a message, and a
// Retry button that re-polls. Only shown when a poll failed AND no snapshot has
// ever arrived; a later failure with a retained snapshot never reaches here.
defineProps<{ message: string | null }>()
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <div class="error-panel" role="alert">
    <span>{{ message ?? 'unknown error' }}</span>
    <button type="button" class="retry" @click="emit('retry')">Retry</button>
  </div>
</template>

<style scoped>
.error-panel {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  color: var(--text);
  font-size: 13px;
  padding: 14px 16px;
  background: var(--panel);
  border: 1px solid var(--danger);
  border-radius: 8px;
}
.retry {
  min-height: 32px;
  min-width: 44px;
  padding: 4px 12px;
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
