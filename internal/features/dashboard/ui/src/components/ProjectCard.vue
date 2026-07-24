<script setup lang="ts">
import { computed } from 'vue'
import type { Project } from '@/types'
import StatusCounts from '@/components/StatusCounts.vue'
import BurndownBar from '@/components/BurndownBar.vue'

// One project's overview card (DESIGN.md §7.1). An <article> whose title is its
// accessible name; pure — no fetching. Stage renders an em-dash when empty.
const props = defineProps<{ project: Project }>()

const headingId = computed(() => `project-${props.project.slug}`)
const displayTitle = computed(() => props.project.title || props.project.slug)
</script>

<template>
  <article class="card" :aria-labelledby="headingId">
    <h3 :id="headingId">{{ displayTitle }}</h3>
    <p class="stage">{{ project.slug }} · stage: {{ project.stage || '—' }}</p>
    <StatusCounts :counts="project.counts" />
    <BurndownBar :burndown="project.burndown" />
  </article>
</template>

<style scoped>
.card {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 14px 16px;
}
h3 {
  margin: 0 0 8px;
  font-size: 14px;
  font-weight: 600;
}
.stage {
  color: var(--muted);
  font-size: 12px;
  margin: 0 0 10px;
}
</style>
