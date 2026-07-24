<script setup lang="ts">
// A calm, bordered "nothing here yet" panel — the shared empty state for every
// surface (DESIGN.md §7). Muted text, never a danger color: an empty swarm is
// the common resting state and must read as normal, not as an error.
</script>

<template>
  <p class="empty-panel"><slot /></p>
</template>

<style scoped>
.empty-panel {
  margin: 0;
  color: var(--muted);
  font-size: 13px;
  padding: 16px;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 8px;
}
</style>
