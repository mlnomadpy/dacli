<script setup lang="ts">
import { computed, ref } from 'vue'
import type { OutcomeAnalyticsResponse, OutcomeMetric } from '@/types'

const props = defineProps<{
  analytics: OutcomeAnalyticsResponse
  range: 7 | 30 | 90
  stale?: boolean
}>()
const emit = defineEmits<{ range: [days: 7 | 30 | 90] }>()
const selected = ref<string | null>(null)

const featured = computed(() =>
  [
    'throughput',
    'execution_time',
    'current_tree_acceptance',
    'first_pass_review',
    'retry_rate',
    'cost',
  ]
    .map((key) => props.analytics.metrics.find((metric) => metric.key === key))
    .filter((metric): metric is OutcomeMetric => Boolean(metric)),
)
const selectedMetric = computed(
  () => props.analytics.metrics.find((metric) => metric.key === selected.value) ?? null,
)
const maxSeries = computed(() =>
  Math.max(1, ...props.analytics.series.map((point) => Math.max(point.completed, point.runs))),
)
const visibleBreakdowns = computed(() => props.analytics.breakdowns.slice(0, 10))

function format(metric: OutcomeMetric, previous = false): string {
  const measure = previous ? metric.previous : metric.current
  if (measure.value === null) return 'Unknown'
  if (measure.unit === 'percent') return `${measure.value.toFixed(0)}%`
  if (measure.unit === 'USD')
    return measure.value.toLocaleString(undefined, {
      style: 'currency',
      currency: 'USD',
      maximumFractionDigits: 3,
    })
  if (measure.unit === 'hours') return `${measure.value.toFixed(measure.value < 10 ? 1 : 0)}h`
  if (measure.unit === 'tokens') return Math.round(measure.value).toLocaleString()
  return measure.value.toFixed(measure.value % 1 ? 1 : 0)
}
</script>

<template>
  <section
    aria-labelledby="outcomes-heading"
    class="overflow-hidden rounded-xl border border-border bg-card"
  >
    <p
      v-if="stale"
      class="m-0 border-b border-amber-500/30 bg-amber-500/10 px-5 py-2 text-xs font-medium text-amber-800 dark:text-amber-200"
      role="status"
    >
      Stale analytics snapshot — the latest refresh failed; values below retain their prior
      observation time.
    </p>
    <header
      class="flex flex-col gap-4 border-b border-border bg-gradient-to-r from-primary/[0.08] via-transparent to-transparent px-5 py-5 lg:flex-row lg:items-end lg:justify-between"
    >
      <div class="max-w-2xl">
        <p class="m-0 text-[10px] font-bold uppercase tracking-[0.14em] text-primary">
          Outcome intelligence
        </p>
        <h2 id="outcomes-heading" class="mt-1.5 mb-0 text-xl font-semibold tracking-tight">
          Is the delivery system improving?
        </h2>
        <p class="mt-1.5 mb-0 text-sm leading-6 text-muted-foreground">
          Comparable windows, exact evidence membership, and visible data gaps. No agent rankings or
          causal claims.
        </p>
      </div>
      <div
        class="inline-flex w-fit rounded-lg border border-border bg-background/80 p-1"
        aria-label="Analytics window"
      >
        <button
          v-for="days in [7, 30, 90] as const"
          :key="days"
          type="button"
          class="rounded-md px-3 py-1.5 text-xs font-medium transition"
          :class="
            range === days
              ? 'bg-foreground text-background shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          "
          :aria-pressed="range === days"
          @click="emit('range', days)"
        >
          {{ days }}d
        </button>
      </div>
    </header>

    <div class="grid gap-px bg-border sm:grid-cols-2 xl:grid-cols-3">
      <button
        v-for="metric in featured"
        :key="metric.key"
        type="button"
        class="group min-w-0 bg-card p-4 text-left transition hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
        @click="selected = selected === metric.key ? null : metric.key"
      >
        <div class="flex items-start justify-between gap-3">
          <span class="text-xs font-medium text-muted-foreground">{{ metric.label }}</span>
          <span
            class="rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide"
            :class="
              metric.current.state === 'complete'
                ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
                : metric.current.state === 'advisory'
                  ? 'bg-violet-500/10 text-violet-700 dark:text-violet-300'
                  : metric.current.state === 'partial'
                    ? 'bg-amber-500/10 text-amber-700 dark:text-amber-300'
                    : 'bg-muted text-muted-foreground'
            "
            >{{ metric.current.state }}</span
          >
        </div>
        <div class="mt-3 flex items-end justify-between gap-3">
          <strong class="text-2xl font-semibold tabular-nums tracking-tight">{{
            format(metric)
          }}</strong>
          <span class="text-right text-[11px] leading-4 text-muted-foreground"
            >prev {{ format(metric, true) }}<br />{{ metric.current.sample_size }}/{{
              metric.current.eligible
            }}
            observed</span
          >
        </div>
        <div class="mt-3 h-1 overflow-hidden rounded-full bg-muted">
          <div
            class="h-full rounded-full bg-primary transition-all"
            :style="{ width: `${Math.max(3, metric.current.coverage * 100)}%` }"
          />
        </div>
      </button>
    </div>

    <div class="grid gap-5 p-5 lg:grid-cols-[minmax(0,1.45fr)_minmax(250px,.55fr)]">
      <div class="min-w-0">
        <div class="mb-3 flex items-baseline justify-between gap-3">
          <h3 class="m-0 text-sm font-semibold">Delivery pulse</h3>
          <span class="text-[11px] text-muted-foreground">accepted tasks / recorded runs</span>
        </div>
        <div
          class="flex h-28 items-end gap-[3px]"
          role="img"
          aria-label="Daily accepted tasks and recorded runs"
        >
          <div
            v-for="point in analytics.series"
            :key="point.day"
            class="group relative flex h-full min-w-0 flex-1 items-end gap-px"
            :title="`${point.day}: ${point.completed} accepted, ${point.runs} runs, ${point.tokens.toLocaleString()} tokens`"
          >
            <span
              class="w-1/2 rounded-t-sm bg-primary/80"
              :style="{
                height: `${Math.max(point.completed ? 5 : 0, (point.completed / maxSeries) * 100)}%`,
              }"
            />
            <span
              class="w-1/2 rounded-t-sm bg-sky-400/45"
              :style="{
                height: `${Math.max(point.runs ? 5 : 0, (point.runs / maxSeries) * 100)}%`,
              }"
            />
          </div>
        </div>
        <div class="mt-2 flex items-center gap-4 text-[11px] text-muted-foreground">
          <span><i class="mr-1 inline-block h-2 w-2 rounded-sm bg-primary/80" />accepted</span
          ><span><i class="mr-1 inline-block h-2 w-2 rounded-sm bg-sky-400/60" />runs</span>
        </div>
      </div>
      <aside class="rounded-lg border border-border bg-muted/25 p-4">
        <h3 class="m-0 text-sm font-semibold">Evidence health</h3>
        <dl class="mt-3 grid grid-cols-2 gap-3 text-xs">
          <div>
            <dt class="text-muted-foreground">Tasks scanned</dt>
            <dd class="mt-0.5 text-lg font-semibold tabular-nums">
              {{ analytics.performance.tasks_scanned }}
            </dd>
          </div>
          <div>
            <dt class="text-muted-foreground">Runs scanned</dt>
            <dd class="mt-0.5 text-lg font-semibold tabular-nums">
              {{ analytics.performance.runs_scanned }}
            </dd>
          </div>
          <div>
            <dt class="text-muted-foreground">Build</dt>
            <dd class="mt-0.5 font-medium tabular-nums">{{ analytics.performance.build_ms }}ms</dd>
          </div>
          <div>
            <dt class="text-muted-foreground">Drill-down cap</dt>
            <dd class="mt-0.5 font-medium tabular-nums">
              {{ analytics.performance.evidence_cap }}
            </dd>
          </div>
        </dl>
        <p
          class="mt-3 mb-0 border-t border-border pt-3 text-[11px] leading-5 text-muted-foreground"
        >
          {{ analytics.notes[1] }}
        </p>
      </aside>
    </div>

    <div v-if="visibleBreakdowns.length" class="border-t border-border px-5 py-4">
      <div class="mb-3 flex items-baseline justify-between gap-3">
        <h3 class="m-0 text-sm font-semibold">Comparable cohorts</h3>
        <span class="text-[11px] text-muted-foreground"
          >size context retained · no people ranking</span
        >
      </div>
      <div class="overflow-x-auto rounded-lg border border-border">
        <table class="w-full min-w-[620px] border-collapse text-left text-xs">
          <thead class="bg-muted/50 text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
            <tr>
              <th class="px-3 py-2 font-semibold">Dimension</th>
              <th class="px-3 py-2 font-semibold">Cohort</th>
              <th class="px-3 py-2 font-semibold">Size</th>
              <th class="px-3 py-2 font-semibold">Current</th>
              <th class="px-3 py-2 font-semibold">Previous</th>
              <th class="px-3 py-2 font-semibold">Confidence</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in visibleBreakdowns"
              :key="`${row.dimension}/${row.key}/${row.size_band}`"
              class="border-t border-border"
            >
              <td class="px-3 py-2 text-muted-foreground">{{ row.dimension.replace('_', ' ') }}</td>
              <td class="max-w-56 truncate px-3 py-2 font-medium" :title="row.key">
                {{ row.key }}
              </td>
              <td class="px-3 py-2 text-muted-foreground">{{ row.size_band || 'unknown' }}</td>
              <td class="px-3 py-2 tabular-nums">{{ row.current.sample_size }}</td>
              <td class="px-3 py-2 tabular-nums">{{ row.previous.sample_size }}</td>
              <td class="px-3 py-2">
                <span
                  :class="
                    row.comparable
                      ? 'text-emerald-700 dark:text-emerald-300'
                      : 'text-amber-700 dark:text-amber-300'
                  "
                  >{{ row.comparable ? 'comparable' : 'descriptive' }}</span
                >
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div
      v-if="selectedMetric"
      class="border-t border-border bg-muted/20 px-5 py-4"
      aria-live="polite"
    >
      <div class="flex items-start justify-between gap-4">
        <div>
          <h3 class="m-0 text-sm font-semibold">{{ selectedMetric.label }} evidence</h3>
          <p class="mt-1 mb-0 text-xs leading-5 text-muted-foreground">
            {{ selectedMetric.current.provenance
            }}<template v-if="selectedMetric.current.caveat">
              · {{ selectedMetric.current.caveat }}</template
            >
          </p>
        </div>
        <button
          type="button"
          class="text-xs text-muted-foreground hover:text-foreground"
          @click="selected = null"
        >
          Close
        </button>
      </div>
      <div class="mt-3 flex flex-wrap gap-1.5">
        <code
          v-for="task in selectedMetric.current.evidence.tasks"
          :key="task"
          class="rounded bg-background px-2 py-1 text-[10px]"
          >task {{ task }}</code
        ><code
          v-for="run in selectedMetric.current.evidence.runs"
          :key="run"
          class="rounded bg-background px-2 py-1 text-[10px]"
          >run {{ run }}</code
        ><span
          v-if="
            selectedMetric.current.evidence.tasks.length +
              selectedMetric.current.evidence.runs.length ===
            0
          "
          class="text-xs text-muted-foreground"
          >No exact evidence in this window.</span
        >
      </div>
    </div>
  </section>
</template>
