<script setup lang="ts">
import { computed } from 'vue'
import type { Burn } from '@/types'
import { Badge } from '@/components/ui/badge'

// The burn-rate surface (task 144). Unlike a passive burndown line, this chart
// YELLS: when the current burn rate reaches `alert_at`× the calibrated ceiling
// the whole panel turns danger-red and raises an assertive live-region banner —
// the one signal all four discovery segments asked for, to catch overspend
// before it becomes a silent, expensive failure. Data-only prop, like the other
// chart leaves; the store owns the poll and hands down the latest `burn`.
const props = defineProps<{ burn: Burn }>()

const hasSeries = computed(() => props.burn.series.length > 0)

/** Scale bars and the ceiling line against the tallest of the two, so the
 * ceiling is always on-canvas even when every bar sits below it. 0-safe. */
const scaleMax = computed(() => {
  const peak = props.burn.series.reduce((m, p) => Math.max(m, p.per_run), 0)
  return Math.max(peak, props.burn.ceiling)
})

/** Bar height as a percentage of the scale; a 4% floor keeps a tiny non-zero
 * day visible, and an all-zero series collapses to 0 (no divide by zero). */
function barPct(perRun: number): number {
  if (scaleMax.value <= 0) return 0
  return Math.max(4, (perRun / scaleMax.value) * 100)
}

/** Where the ceiling reference line sits, as a percentage from the bottom.
 * Hidden (0) when there is no calibrated ceiling to draw. */
const ceilingPct = computed(() =>
  scaleMax.value > 0 ? (props.burn.ceiling / scaleMax.value) * 100 : 0,
)

/** A single day is "hot" when it alone breaches the alert threshold — that bar
 * is painted danger even if the headline rate (the latest day) has cooled. */
function isHot(perRun: number): boolean {
  return props.burn.ceiling > 0 && perRun >= props.burn.alert_at * props.burn.ceiling
}

const ratioText = computed(() => props.burn.ratio.toFixed(1))
const rateText = computed(() => Math.round(props.burn.rate).toLocaleString())
const ceilingText = computed(() => Math.round(props.burn.ceiling).toLocaleString())

const alertMessage = computed(
  () =>
    `burning ${ratioText.value}× the calibrated ceiling of ${ceilingText.value} ${props.burn.unit}/run`,
)
</script>

<template>
  <section
    aria-labelledby="burn-h"
    class="rounded-lg border bg-card px-4 py-3.5 transition-colors"
    :class="
      burn.alert
        ? 'alert border-destructive shadow-[0_0_0_1px_var(--destructive)]'
        : 'border-border'
    "
  >
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h2
        id="burn-h"
        class="m-0 text-[13px] font-semibold uppercase tracking-[0.06em] text-muted-foreground"
      >
        Burn rate
      </h2>
      <Badge
        v-if="burn.ceiling > 0"
        :variant="burn.alert ? 'destructive' : 'outline'"
        class="font-normal"
        :class="{ 'font-semibold': burn.alert }"
        :title="`current rate ÷ calibrated ceiling`"
      >
        {{ ratioText }}× ceiling
      </Badge>
    </div>

    <!-- The yell: an assertive live region only present when overspend trips. -->
    <p
      v-if="burn.alert"
      class="mt-2.5 mb-0 text-[13px] font-semibold text-destructive"
      role="alert"
      aria-live="assertive"
    >
      <span aria-hidden="true">⚠ </span>{{ alertMessage }}
    </p>

    <p v-if="!hasSeries" class="mt-2.5 mb-0 text-xs text-muted-foreground">
      no usage recorded yet — burn starts when a run reports tokens
    </p>

    <template v-else>
      <div class="mt-3 mb-2.5 flex flex-wrap gap-6">
        <span class="flex items-baseline gap-1.5">
          <span class="text-[10px] font-semibold uppercase tracking-[0.05em] text-muted-foreground"
            >rate</span
          >
          <span
            class="text-xl font-semibold tabular-nums"
            :class="{ 'text-destructive': burn.alert }"
            >{{ rateText }}</span
          >
          <span class="text-[11px] text-muted-foreground">{{ burn.unit }}/run</span>
        </span>
        <span v-if="burn.ceiling > 0" class="flex items-baseline gap-1.5">
          <span class="text-[10px] font-semibold uppercase tracking-[0.05em] text-muted-foreground"
            >ceiling</span
          >
          <span class="text-xl font-semibold tabular-nums">{{ ceilingText }}</span>
          <span class="text-[11px] text-muted-foreground">{{ burn.unit }}/run</span>
        </span>
      </div>

      <div
        class="relative flex h-16 items-end gap-1"
        role="img"
        :aria-label="`burn rate across ${burn.series.length} day(s); ${
          burn.alert ? 'ALERT: ' + alertMessage : 'within the calibrated ceiling'
        }`"
      >
        <div
          v-if="ceilingPct > 0"
          class="ceiling-line pointer-events-none absolute right-0 left-0 h-0 border-t border-dashed border-muted-foreground"
          :style="{ bottom: ceilingPct + '%' }"
          aria-hidden="true"
        />
        <div v-for="d in burn.series" :key="d.day" class="flex h-full min-w-1 flex-1 items-end">
          <div
            class="bar min-h-0.5 w-full rounded-t-sm"
            :class="isHot(d.per_run) ? 'hot bg-destructive' : 'bg-primary'"
            :style="{ height: barPct(d.per_run) + '%' }"
            :title="`${d.day}: ${Math.round(d.per_run).toLocaleString()} ${burn.unit}/run · ${d.runs} run(s)`"
          />
        </div>
      </div>
      <p class="mt-2 mb-0 text-[11px] text-muted-foreground">
        {{ burn.series[burn.series.length - 1].day }} · latest of {{ burn.series.length }} day(s),
        ceiling from {{ burn.bands.length }} calibrated band(s)
      </p>
    </template>
  </section>
</template>
