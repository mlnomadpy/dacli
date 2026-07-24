<script setup lang="ts">
import type { Phase, Project } from '@/types'
import ProjectGrid from '@/components/ProjectGrid.vue'

// The Overview section shell (DESIGN.md §4–§5): a labelled <section> wrapping
// ProjectGrid, which owns the state. Overview always shows ALL projects.
defineProps<{
  projects: Project[]
  phase: Phase
  hasSnapshot: boolean
  error: string | null
}>()
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <section aria-labelledby="overview-h">
    <h2 id="overview-h">Overview</h2>
    <ProjectGrid
      :projects="projects"
      :phase="phase"
      :has-snapshot="hasSnapshot"
      :error="error"
      @retry="emit('retry')"
    />
  </section>
</template>
