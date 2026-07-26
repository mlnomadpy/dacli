<script setup lang="ts">
import { computed } from 'vue'
import type { Phase, Project } from '@/types'
import { sectionState } from '@/composables/useSectionState'
import ProjectCard from '@/components/ProjectCard.vue'
import EmptyPanel from '@/components/EmptyPanel.vue'
import ErrorPanel from '@/components/ErrorPanel.vue'
import SkeletonBlock from '@/components/SkeletonBlock.vue'

// Owns the Overview surface's four states (DESIGN.md §7.1). Loading shows three
// skeleton cards; empty is a calm "no projects yet"; cold error is a danger
// panel with Retry; live is the auto-fill grid of ProjectCards.
const props = defineProps<{
  projects: Project[]
  phase: Phase
  hasSnapshot: boolean
  error: string | null
}>()
const emit = defineEmits<{ retry: [] }>()

const state = computed(() =>
  sectionState(props.phase, props.hasSnapshot, props.projects.length === 0),
)

const gridClass = 'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(280px,1fr))]'
</script>

<template>
  <div v-if="state === 'loading'" :class="gridClass" aria-hidden="true">
    <div
      v-for="n in 3"
      :key="n"
      class="skeleton-card flex flex-col gap-2.5 rounded-lg border border-border bg-card px-4 py-3.5"
    >
      <SkeletonBlock width="60%" height="14px" />
      <SkeletonBlock width="90%" height="12px" />
      <SkeletonBlock height="8px" />
    </div>
  </div>
  <ErrorPanel
    v-else-if="state === 'error'"
    :message="`couldn't load projects — ${error ?? 'unknown error'}`"
    @retry="emit('retry')"
  />
  <EmptyPanel v-else-if="state === 'empty'">no projects yet</EmptyPanel>
  <div v-else :class="gridClass">
    <ProjectCard v-for="p in projects" :key="p.slug" :project="p" />
  </div>
</template>
