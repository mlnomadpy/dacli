<script setup lang="ts">
import { computed, ref, watch } from 'vue'
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
  graphMode: 'operational' | 'history'
  graphStatuses: string[]
  graphFocus: string
  graphPage: number
}>()
const emit = defineEmits<{
  'update:selectedSlug': [slug: string]
  retry: []
  inspect: [task: string, trigger?: HTMLElement]
  query: [
    query: Partial<{
      mode: 'operational' | 'history'
      statuses: string[]
      focus: string
      page: number
    }>,
  ]
}>()

const focusInput = ref(props.graphFocus)
watch(
  () => props.graphFocus,
  (value) => {
    focusInput.value = value
  },
)
const statuses = ['open', 'active', 'blocked'] as const

function toggleStatus(status: string): void {
  const next = props.graphStatuses.includes(status)
    ? props.graphStatuses.filter((item) => item !== status)
    : [...props.graphStatuses, status]
  emit('query', { statuses: next, focus: '', page: 1 })
}

function submitFocus(): void {
  emit('query', { focus: focusInput.value.trim(), mode: 'operational', page: 1 })
}

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

    <div class="mb-3 flex flex-wrap items-end gap-3 rounded-lg border border-border bg-card p-3">
      <div class="min-w-[220px] flex-1">
        <label
          for="graph-focus"
          class="mb-1 block text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground"
          >Exact task focus</label
        >
        <form class="flex gap-2" @submit.prevent="submitFocus">
          <input
            id="graph-focus"
            v-model="focusInput"
            type="search"
            placeholder="Task ID, sequence, or slug"
            class="h-9 min-w-0 flex-1 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
          <button
            type="submit"
            class="h-9 rounded-md bg-primary px-3 text-xs font-semibold text-primary-foreground"
          >
            Focus
          </button>
        </form>
      </div>
      <fieldset v-if="graphMode === 'operational'" class="flex flex-wrap gap-2">
        <legend
          class="mb-1 text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground"
        >
          Statuses
        </legend>
        <button
          v-for="status in statuses"
          :key="status"
          type="button"
          class="rounded-md border px-2.5 py-1.5 text-xs"
          :class="
            graphStatuses.includes(status)
              ? 'border-primary bg-primary/10 text-primary'
              : 'border-border text-muted-foreground'
          "
          :aria-pressed="graphStatuses.includes(status)"
          @click="toggleStatus(status)"
        >
          {{ status }}
        </button>
      </fieldset>
      <button
        type="button"
        class="h-9 rounded-md border border-border px-3 text-xs font-semibold"
        @click="
          emit('query', {
            mode: graphMode === 'history' ? 'operational' : 'history',
            focus: '',
            page: 1,
          })
        "
      >
        {{ graphMode === 'history' ? 'Operational view' : 'Show completed history' }}
      </button>
      <div v-if="graphMode === 'history'" class="flex items-center gap-2 text-xs">
        <button
          type="button"
          class="rounded border border-border px-2 py-1 disabled:opacity-40"
          :disabled="graphPage <= 1"
          @click="emit('query', { page: graphPage - 1 })"
        >
          Previous
        </button>
        <span>Page {{ graphPage }}</span>
        <button
          type="button"
          class="rounded border border-border px-2 py-1"
          :disabled="!graph.projection.has_more"
          @click="emit('query', { page: graphPage + 1 })"
        >
          Next
        </button>
      </div>
    </div>

    <DependencyGraph
      v-if="selectedProject && hasSnapshot"
      :graph="graph"
      @inspect="(task, trigger) => emit('inspect', task, trigger)"
    />
    <SkeletonBlock v-else-if="phase === 'loading'" height="120px" />
    <ErrorPanel
      v-else-if="phase === 'error'"
      :message="`couldn't load task dependencies — ${error ?? 'unknown error'}`"
      @retry="emit('retry')"
    />
    <p v-else class="text-xs text-muted-foreground">no projects yet</p>
  </section>
</template>
