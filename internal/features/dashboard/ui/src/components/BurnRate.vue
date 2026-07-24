<script setup lang="ts">
import { computed } from 'vue'
import type { Burn } from '@/types'

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
  <section aria-labelledby="burn-h" :class="{ alert: burn.alert }">
    <div class="section-head">
      <h2 id="burn-h">Burn rate</h2>
      <span
        v-if="burn.ceiling > 0"
        class="ratio-chip"
        :class="{ hot: burn.alert }"
        :title="`current rate ÷ calibrated ceiling`"
      >
        {{ ratioText }}× ceiling
      </span>
    </div>

    <!-- The yell: an assertive live region only present when overspend trips. -->
    <p v-if="burn.alert" class="yell" role="alert" aria-live="assertive">
      <span aria-hidden="true">⚠ </span>{{ alertMessage }}
    </p>

    <p v-if="!hasSeries" class="empty-note">
      no usage recorded yet — burn starts when a run reports tokens
    </p>

    <template v-else>
      <div class="stat-row">
        <span class="stat">
          <span class="stat-label">rate</span>
          <span class="stat-val" :class="{ hot: burn.alert }">{{ rateText }}</span>
          <span class="stat-unit">{{ burn.unit }}/run</span>
        </span>
        <span v-if="burn.ceiling > 0" class="stat">
          <span class="stat-label">ceiling</span>
          <span class="stat-val">{{ ceilingText }}</span>
          <span class="stat-unit">{{ burn.unit }}/run</span>
        </span>
      </div>

      <div
        class="chart"
        role="img"
        :aria-label="`burn rate across ${burn.series.length} day(s); ${
          burn.alert ? 'ALERT: ' + alertMessage : 'within the calibrated ceiling'
        }`"
      >
        <div
          v-if="ceilingPct > 0"
          class="ceiling-line"
          :style="{ bottom: ceilingPct + '%' }"
          aria-hidden="true"
        />
        <div v-for="d in burn.series" :key="d.day" class="bar-wrap">
          <div
            class="bar"
            :class="{ hot: isHot(d.per_run) }"
            :style="{ height: barPct(d.per_run) + '%' }"
            :title="`${d.day}: ${Math.round(d.per_run).toLocaleString()} ${burn.unit}/run · ${d.runs} run(s)`"
          />
        </div>
      </div>
      <p class="caption">
        {{ burn.series[burn.series.length - 1].day }} · latest of {{ burn.series.length }} day(s),
        ceiling from {{ burn.bands.length }} calibrated band(s)
      </p>
    </template>
  </section>
</template>

<style scoped>
section {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 14px 16px;
  background: var(--panel);
  transition: border-color 0.2s;
}
/* The yell: overspend repaints the whole panel danger-red, not a passive line. */
section.alert {
  border-color: var(--danger);
  box-shadow: 0 0 0 1px var(--danger);
}
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.ratio-chip {
  font-size: 12px;
  color: var(--muted);
  border: 1px solid var(--border);
  border-radius: 999px;
  padding: 2px 8px;
}
.ratio-chip.hot {
  color: var(--danger);
  border-color: var(--danger);
  font-weight: 600;
}
.yell {
  margin: 10px 0 0;
  color: var(--danger);
  font-size: 13px;
  font-weight: 600;
}
.empty-note {
  margin: 10px 0 0;
  color: var(--muted);
  font-size: 12px;
}
.stat-row {
  display: flex;
  gap: 24px;
  margin: 12px 0 10px;
  flex-wrap: wrap;
}
.stat {
  display: flex;
  align-items: baseline;
  gap: 6px;
}
.stat-label {
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-size: 10px;
  font-weight: 600;
  color: var(--muted);
}
.stat-val {
  font-size: 20px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.stat-val.hot {
  color: var(--danger);
}
.stat-unit {
  font-size: 11px;
  color: var(--muted);
}
.chart {
  position: relative;
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 64px;
}
.ceiling-line {
  position: absolute;
  left: 0;
  right: 0;
  height: 0;
  border-top: 1px dashed var(--muted);
  pointer-events: none;
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
  background: var(--accent);
  border-radius: 2px 2px 0 0;
  min-height: 2px;
}
/* A day that alone breaches the threshold burns danger-red. */
.bar.hot {
  background: var(--danger);
}
.caption {
  margin: 8px 0 0;
  color: var(--muted);
  font-size: 11px;
}
</style>
