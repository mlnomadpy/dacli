<script setup lang="ts">
import { computed } from 'vue'
import type { Burndown } from '@/types'
import { Card } from '@/components/ui/card'

// The per-day "what closed each day" bar sparkline (DESIGN.md §7.2). This is
// ADDITIVE landed-points-per-day, NOT a classic descending-ideal burndown — the
// snapshot only carries landed-per-day, so the honest visualization is a bar
// series captioned as such. Bars are server-sorted chronologically; never
// re-sort. Empty when `per_day` is empty, but the numeric summary still shows.
const props = defineProps<{ burndown: Burndown }>()

const maxPoints = computed(() => props.burndown.per_day.reduce((m, d) => Math.max(m, d.points), 0))

/** Bar height as a percentage of the tallest bar; a floor of 4% keeps a tiny
 * non-zero day visible. Guarded against an all-zero series (no divide by zero). */
function barPct(points: number): number {
  if (maxPoints.value <= 0) return 0
  return Math.max(4, (points / maxPoints.value) * 100)
}

const summary = computed(() => {
  const b = props.burndown
  let s = `${b.done_points.toFixed(1)} done · ${b.remaining_points.toFixed(1)} remaining pts`
  if (b.unestimated > 0) s += ` · ${b.unestimated} unestimated`
  return s
})
</script>

<template>
  <Card class="burndown mt-3 gap-2.5 rounded-lg px-3.5 py-3">
    <div class="flex flex-wrap items-baseline gap-2">
      <span class="text-[10px] font-semibold uppercase tracking-[0.05em] text-muted-foreground">
        Burndown
      </span>
      <span class="text-xs text-muted-foreground">{{ summary }}</span>
    </div>
    <p v-if="burndown.per_day.length === 0" class="m-0 text-xs text-muted-foreground">
      nothing completed yet — burndown starts when the first task closes
    </p>
    <div
      v-else
      class="chart flex h-14 items-end gap-1"
      role="img"
      :aria-label="`per-day landed points across ${burndown.per_day.length} day(s)`"
    >
      <div v-for="d in burndown.per_day" :key="d.day" class="flex h-full min-w-1 flex-1 items-end">
        <div
          class="bar min-h-0.5 w-full rounded-t-sm bg-success"
          :style="{ height: barPct(d.points) + '%' }"
          :title="`${d.day}: ${d.points.toFixed(1)} pts`"
          :aria-label="`${d.day}: ${d.points.toFixed(1)} points`"
        />
      </div>
    </div>
  </Card>
</template>
