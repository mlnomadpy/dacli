<script setup lang="ts">
import { computed } from 'vue'
import type { Project } from '@/types'
import StatusCounts from '@/components/StatusCounts.vue'
import BurndownBar from '@/components/BurndownBar.vue'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

// One project's overview card (DESIGN.md §7.1). A Card whose title is its
// accessible name; pure — no fetching. Stage renders an em-dash when empty.
const props = defineProps<{ project: Project }>()

const headingId = computed(() => `project-${props.project.slug}`)
const displayTitle = computed(() => props.project.title || props.project.slug)
</script>

<template>
  <Card :aria-labelledby="headingId" class="gap-0 rounded-lg py-3.5">
    <CardHeader class="gap-1 px-4">
      <CardTitle :id="headingId" class="text-sm">{{ displayTitle }}</CardTitle>
      <p class="m-0 text-xs text-muted-foreground">
        {{ project.slug }} · stage: {{ project.stage || '—' }}
      </p>
    </CardHeader>
    <CardContent class="mt-2.5 px-4">
      <StatusCounts :counts="project.counts" />
      <BurndownBar :burndown="project.burndown" />
    </CardContent>
  </Card>
</template>
