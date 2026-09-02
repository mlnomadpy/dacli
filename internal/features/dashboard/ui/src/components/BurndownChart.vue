<script setup lang="ts">
import { computed } from 'vue'
import ChartFrame from '@/components/ChartFrame.vue'
import type { Burndown, ChartContract } from '@/types'
import { Card } from '@/components/ui/card'

// The per-day "what closed each day" bar sparkline (DESIGN.md §7.2). This is
// ADDITIVE landed-points-per-day, NOT a classic descending-ideal burndown — the
// snapshot only carries landed-per-day, so the honest visualization is a bar
// series captioned as such. Bars are server-sorted chronologically; never
// re-sort. Empty when `per_day` is empty, but the numeric summary still shows.
const props = defineProps<{ burndown: Burndown; project: string; focusDay?: string }>()

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
const selectedPoint = computed(() =>
  props.burndown.per_day.find((point) => point.day === props.focusDay),
)
const contract = computed<ChartContract>(() => ({
  id: 'burndown-completions',
  title: 'Landed points by day',
  metric_definition: 'Estimated task points completed on each recorded UTC day',
  unit: 'PERT expected points',
  window: props.burndown.per_day.length
    ? `${props.burndown.per_day[0].day} → ${props.burndown.per_day[props.burndown.per_day.length - 1].day}`
    : 'no observed window',
  source: 'task completion Log + three-point estimate',
  freshness: 'current project snapshot',
  coverage: `${props.burndown.unestimated} unestimated task(s) excluded`,
  state: props.burndown.per_day.length
    ? props.burndown.unestimated
      ? 'partial'
      : 'live'
    : 'empty',
  state_detail: props.burndown.unestimated
    ? 'Tasks without a three-point estimate remain visible in the board but cannot contribute zero points.'
    : '',
  summary: summary.value,
  points: props.burndown.per_day.map((point) => ({
    id: point.day,
    label: point.day,
    value: point.points,
    display: `${point.points.toFixed(1)} points`,
    href: `#/work?project=${encodeURIComponent(props.project)}&day=${point.day}`,
    evidence_count: point.task_ids?.length ?? 0,
  })),
  hidden_resolution: props.burndown.hidden_days
    ? `${props.burndown.hidden_days} intermediate day point(s) hidden; first, last, minimum, and maximum retained`
    : undefined,
}))
</script>

<template>
  <Card class="burndown mt-3 gap-2.5 rounded-lg px-3.5 py-3">
    <ChartFrame :contract="contract">
      <p v-if="burndown.per_day.length === 0" class="m-0 text-xs text-muted-foreground">
        nothing completed yet — burndown starts when the first task closes
      </p>
      <div
        v-else
        class="chart flex h-14 items-end gap-1"
        role="list"
        aria-label="Daily landed-point evidence"
      >
        <a
          v-for="d in burndown.per_day"
          :key="d.day"
          :href="`#/work?project=${encodeURIComponent(project)}&day=${d.day}`"
          class="flex h-full min-w-1 flex-1 items-end rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          role="listitem"
          :aria-label="`${d.day}: ${d.points.toFixed(1)} points from ${d.task_ids?.length ?? 0} exact tasks`"
        >
          <span
            class="bar min-h-0.5 w-full rounded-t-sm bg-success"
            :style="{ height: barPct(d.points) + '%' }"
            :title="`${d.day}: ${d.points.toFixed(1)} pts`"
            :aria-label="`${d.day}: ${d.points.toFixed(1)} points`"
          />
        </a>
      </div>
    </ChartFrame>
    <div v-if="selectedPoint" class="rounded-md border border-border bg-muted/30 p-3">
      <p class="m-0 text-xs font-semibold">{{ selectedPoint.day }} exact tasks</p>
      <div class="mt-2 flex flex-wrap gap-1.5">
        <a
          v-for="task in selectedPoint.task_ids ?? []"
          :key="task"
          :href="`#/work?project=${encodeURIComponent(project)}&task=${encodeURIComponent(task)}`"
          class="rounded bg-background px-2 py-1 font-mono text-[10px] text-primary hover:underline"
          >{{ task }}</a
        ><span
          v-if="(selectedPoint.task_ids?.length ?? 0) === 0"
          class="text-xs text-muted-foreground"
          >No exact task identity was retained for this point.</span
        >
      </div>
    </div>
  </Card>
</template>
