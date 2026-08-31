<script setup lang="ts">
import { computed } from 'vue'
import type { Phase, Project } from '@/types'
import { emptyGraph } from '@/types'
import ProjectSwitcher from '@/components/ProjectSwitcher.vue'
import DependencyGraph from '@/components/DependencyGraph.vue'
import SkeletonBlock from '@/components/SkeletonBlock.vue'
import ErrorPanel from '@/components/ErrorPanel.vue'

// The dependency-graph section. Mirrors BoardSection: it renders the SELECTED
// project's DAG and shares the same project switcher semantics, so switching a
// project drives the board and the graph together. Pure/read-only — the store
// owns the poll; this only picks which project's embedded graph to draw.
const props = defineProps<{
  projects: Project[]
  selectedSlug: string
  phase: Phase
  hasSnapshot: boolean
  error: string | null
}>()
const emit = defineEmits<{ 'update:selectedSlug': [slug: string]; retry: [] }>()

const selectedProject = computed<Project | null>(
  () => props.projects.find((p) => p.slug === props.selectedSlug) ?? props.projects[0] ?? null,
)

// A project resolved before its graph field arrives (or an older snapshot)
// falls back to the zero-safe empty graph so the leaf never binds to undefined.
const graph = computed(() => selectedProject.value?.graph ?? emptyGraph())
</script>

<template>
  <section aria-labelledby="dag-section-h">
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <h2
        id="dag-section-h"
        class="m-0 text-[13px] font-semibold uppercase tracking-[0.06em] text-muted-foreground"
      >
        Task dependencies
      </h2>
      <ProjectSwitcher
        v-if="projects.length > 1"
        :projects="projects"
        :selected-slug="selectedProject?.slug ?? ''"
        @update:selected-slug="emit('update:selectedSlug', $event)"
      />
    </div>

    <DependencyGraph v-if="selectedProject && hasSnapshot" :graph="graph" />
    <SkeletonBlock v-else-if="phase === 'loading'" height="120px" />
    <ErrorPanel
      v-else-if="phase === 'error'"
      :message="`couldn't load task dependencies — ${error ?? 'unknown error'}`"
      @retry="emit('retry')"
    />
    <p v-else class="text-xs text-muted-foreground">no projects yet</p>
  </section>
</template>
