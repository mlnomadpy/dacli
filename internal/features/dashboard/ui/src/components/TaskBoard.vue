<script setup lang="ts">
import { computed } from 'vue'
import type { Phase, Project } from '@/types'
import { STATUSES } from '@/types'
import { count } from '@/composables/useStatusTheme'
import { sectionState } from '@/composables/useSectionState'
import BoardColumn from '@/components/BoardColumn.vue'
import EmptyPanel from '@/components/EmptyPanel.vue'
import ErrorPanel from '@/components/ErrorPanel.vue'
import SkeletonBlock from '@/components/SkeletonBlock.vue'

// Four fixed status columns for the selected project (DESIGN.md §7.2). The
// column set and order are canonical (open, active, blocked, done) and never
// change — the board's shape is fixed so the eye learns it. `project` is null
// when no project exists at all. Mobile: 2×2 grid preserving canonical order.
const props = defineProps<{
  project: Project | null
  phase: Phase
  hasSnapshot: boolean
  error: string | null
}>()
const emit = defineEmits<{ retry: [] }>()

const isEmpty = computed(() => !props.project || props.project.total === 0)
const state = computed(() => sectionState(props.phase, props.hasSnapshot, isEmpty.value))
const statuses = STATUSES

const boardClass = 'grid grid-cols-2 gap-2 sm:grid-cols-4'
</script>

<template>
  <div v-if="state === 'loading'" :class="boardClass" aria-hidden="true">
    <div
      v-for="n in 4"
      :key="n"
      class="flex flex-col gap-2 rounded-lg border border-border bg-card p-3"
    >
      <SkeletonBlock width="50%" height="10px" />
      <SkeletonBlock width="30%" height="18px" />
    </div>
  </div>
  <ErrorPanel
    v-else-if="state === 'error'"
    :message="`couldn't load the board — ${error ?? 'unknown error'}`"
    @retry="emit('retry')"
  />
  <EmptyPanel v-else-if="state === 'empty'">no tasks in this project</EmptyPanel>
  <div v-else :class="boardClass">
    <BoardColumn v-for="s in statuses" :key="s" :status="s" :count="count(project!.counts, s)" />
  </div>
</template>
