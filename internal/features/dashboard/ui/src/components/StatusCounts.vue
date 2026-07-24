<script setup lang="ts">
import type { StatusCounts } from '@/types'
import { useStatusTheme } from '@/composables/useStatusTheme'

// Four dot+count chips in canonical order (DESIGN.md §7.1). Color is never the
// only signal: every dot is paired with its text label, and a missing count
// key resolves to 0 through `count()` — never `undefined` (DESIGN.md §0, §8).
defineProps<{ counts: StatusCounts }>()
const { statuses, statusColor, count } = useStatusTheme()
</script>

<template>
  <ul class="counts">
    <li v-for="s in statuses" :key="s">
      <i class="dot" :style="{ background: statusColor(s) }" aria-hidden="true" />
      <span>{{ s }} {{ count(counts, s) }}</span>
    </li>
  </ul>
</template>

<style scoped>
.counts {
  list-style: none;
  margin: 0 0 10px;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px 10px;
  font-size: 12px;
}
.counts li {
  display: flex;
  align-items: center;
  gap: 4px;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
  flex: none;
}
</style>
