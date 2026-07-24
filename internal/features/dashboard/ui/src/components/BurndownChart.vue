<script setup lang="ts">
import { computed } from 'vue'
import type { Burndown } from '@/types'

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
  <div class="burndown">
    <div class="summary">
      <span class="tag">Burndown</span>
      <span class="nums">{{ summary }}</span>
    </div>
    <p v-if="burndown.per_day.length === 0" class="empty-note">
      nothing completed yet — burndown starts when the first task closes
    </p>
    <div
      v-else
      class="chart"
      role="img"
      :aria-label="`per-day landed points across ${burndown.per_day.length} day(s)`"
    >
      <div v-for="d in burndown.per_day" :key="d.day" class="bar-wrap">
        <div
          class="bar"
          :style="{ height: barPct(d.points) + '%' }"
          :title="`${d.day}: ${d.points.toFixed(1)} pts`"
          :aria-label="`${d.day}: ${d.points.toFixed(1)} points`"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.burndown {
  margin-top: 12px;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 12px 14px;
}
.summary {
  display: flex;
  align-items: baseline;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 10px;
}
.tag {
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-size: 10px;
  font-weight: 600;
  color: var(--muted);
}
.nums {
  font-size: 12px;
  color: var(--muted);
}
.empty-note {
  margin: 0;
  color: var(--muted);
  font-size: 12px;
}
.chart {
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 56px;
}
.bar-wrap {
  flex: 1 1 0;
  min-width: 4px;
  height: 100%;
  display: flex;
  align-items: flex-end;
}
.bar {
  width: 100%;
  background: var(--done);
  border-radius: 2px 2px 0 0;
  min-height: 2px;
}
</style>
