<script setup lang="ts">
import type { Project } from '@/types'

// Client-only project selection for the Board + Burndown section (DESIGN.md
// §5). Rendered only when projects.length > 1. A native <select> so it is
// keyboard-reachable with a visible focus ring; the choice is never persisted
// server-side (the UI mutates nothing).
defineProps<{ projects: Project[]; selectedSlug: string }>()
const emit = defineEmits<{ 'update:selectedSlug': [slug: string] }>()

function onChange(e: Event) {
  emit('update:selectedSlug', (e.target as HTMLSelectElement).value)
}
</script>

<template>
  <label class="switcher">
    <span class="sr-only">Project</span>
    <select :value="selectedSlug" @change="onChange">
      <option v-for="p in projects" :key="p.slug" :value="p.slug">
        {{ p.title || p.slug }}
      </option>
    </select>
  </label>
</template>

<style scoped>
.switcher {
  display: inline-flex;
}
select {
  min-height: 32px;
  font: inherit;
  color: var(--text);
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 4px 8px;
}
select:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
</style>
