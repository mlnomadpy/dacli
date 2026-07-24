<script setup lang="ts">
import { computed } from 'vue'
import type { Phase, Project } from '@/types'
import ProjectSwitcher from '@/components/ProjectSwitcher.vue'
import TaskBoard from '@/components/TaskBoard.vue'
import BurndownChart from '@/components/BurndownChart.vue'
import SkeletonBlock from '@/components/SkeletonBlock.vue'

// The Task board + Burndown section (DESIGN.md §4–§5, §7.2). Renders the
// selected project's board and per-day chart; the switcher appears only with
// more than one project. TaskBoard owns the loading/empty/error views for the
// board itself; the chart renders once a project is resolved.
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
</script>

<template>
  <section aria-labelledby="board-h">
    <div class="section-head">
      <h2 id="board-h">Task board + Burndown</h2>
      <ProjectSwitcher
        v-if="projects.length > 1"
        :projects="projects"
        :selected-slug="selectedProject?.slug ?? ''"
        @update:selected-slug="emit('update:selectedSlug', $event)"
      />
    </div>

    <TaskBoard
      :project="selectedProject"
      :phase="phase"
      :has-snapshot="hasSnapshot"
      :error="error"
      @retry="emit('retry')"
    />

    <BurndownChart v-if="selectedProject" :burndown="selectedProject.burndown" />
    <SkeletonBlock v-else-if="phase === 'loading'" height="80px" />
  </section>
</template>

<style scoped>
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}
</style>
