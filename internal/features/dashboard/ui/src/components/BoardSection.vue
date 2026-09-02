<script setup lang="ts">
import { computed } from 'vue'
import type { Phase, Project, TaskSummary } from '@/types'
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
  tasks: TaskSummary[]
  tasksPhase: Phase
  tasksHasSnapshot: boolean
  tasksError: string | null
  query?: string
  day?: string
}>()
const emit = defineEmits<{
  'update:selectedSlug': [slug: string]
  retry: []
  retryTasks: []
  inspect: [task: string, trigger: HTMLElement]
}>()

const selectedProject = computed<Project | null>(
  () => props.projects.find((p) => p.slug === props.selectedSlug) ?? props.projects[0] ?? null,
)
</script>

<template>
  <section aria-labelledby="board-h">
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <h2
        id="board-h"
        class="m-0 text-[13px] font-semibold uppercase tracking-[0.06em] text-muted-foreground"
      >
        Task board + Burndown
      </h2>
      <ProjectSwitcher
        v-if="projects.length > 1"
        :projects="projects"
        :selected-slug="selectedProject?.slug ?? ''"
        @update:selected-slug="emit('update:selectedSlug', $event)"
      />
    </div>

    <TaskBoard
      :project="selectedProject"
      :tasks="tasks"
      :query="query"
      :phase="tasksPhase"
      :has-snapshot="tasksHasSnapshot"
      :error="tasksError"
      @retry="emit('retryTasks')"
      @inspect="(task, trigger) => emit('inspect', task, trigger)"
    />

    <BurndownChart
      v-if="selectedProject"
      :burndown="selectedProject.burndown"
      :project="selectedProject.slug"
      :focus-day="day"
    />
    <SkeletonBlock v-else-if="phase === 'loading'" height="80px" />
  </section>
</template>
