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
  <label class="inline-flex">
    <span class="sr-only">Project</span>
    <select
      :value="selectedSlug"
      class="min-h-8 rounded-md border border-border bg-secondary px-2 py-1 text-sm text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      @change="onChange"
    >
      <option v-for="p in projects" :key="p.slug" :value="p.slug">
        {{ p.title || p.slug }}
      </option>
    </select>
  </label>
</template>
