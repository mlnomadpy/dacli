<script setup lang="ts">
import type { StatusCounts } from '@/types'
import { useStatusTheme } from '@/composables/useStatusTheme'
import { Badge } from '@/components/ui/badge'

// Four dot+count chips in canonical order (DESIGN.md §7.1). Color is never the
// only signal: every dot is paired with its text label, and a missing count
// key resolves to 0 through `count()` — never `undefined` (DESIGN.md §0, §8).
defineProps<{ counts: StatusCounts }>()
const { statuses, statusColor, count } = useStatusTheme()
</script>

<template>
  <ul class="counts m-0 mb-2.5 flex list-none flex-wrap gap-x-2.5 gap-y-1.5 p-0">
    <li v-for="s in statuses" :key="s">
      <Badge variant="outline" class="gap-1.5 text-xs font-normal">
        <i
          class="dot size-2 shrink-0 rounded-full"
          :style="{ background: statusColor(s) }"
          aria-hidden="true"
        />
        <span>{{ s }} {{ count(counts, s) }}</span>
      </Badge>
    </li>
  </ul>
</template>
