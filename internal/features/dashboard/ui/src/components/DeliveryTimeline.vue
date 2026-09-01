<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import type { DeliveryAttempt, DeliveryTimelineResponse, GraphNode } from '@/types'
import { dashboardHref } from '@/composables/useDashboardRoute'

const props = defineProps<{
  tasks: GraphNode[]
  selectedTask: string
  timeline: DeliveryTimelineResponse | null
}>()
const emit = defineEmits<{ select: [task: string] }>()
const phaseButtons = ref<HTMLElement[]>([])

const attempts = computed(() => [...(props.timeline?.attempts ?? [])].reverse())

function duration(value: number | null): string {
  if (value === null) return 'unknown duration'
  if (value < 1_000) return `${value} ms`
  if (value < 60_000) return `${Math.round(value / 1_000)} s`
  return `${Math.round(value / 60_000)} min`
}

function movePhase(event: KeyboardEvent, index: number): void {
  if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key))
    return
  event.preventDefault()
  let next = index
  if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') next = Math.max(0, index - 1)
  if (event.key === 'ArrowRight' || event.key === 'ArrowDown')
    next = Math.min(phaseButtons.value.length - 1, index + 1)
  if (event.key === 'Home') next = 0
  if (event.key === 'End') next = phaseButtons.value.length - 1
  void nextTick(() => phaseButtons.value[next]?.focus())
}

function rememberPhase(el: unknown, index: number): void {
  if (el instanceof HTMLElement) phaseButtons.value[index] = el
}

function diagnosisTone(value: DeliveryAttempt['diagnosis']['class']): string {
  if (value === 'accepted-on-current-tree') return 'border-success/50 bg-success/10 text-success'
  if (value === 'pending' || value === 'merged-not-accepted')
    return 'border-warning/50 bg-warning/10 text-warning'
  return 'border-destructive/50 bg-destructive/10 text-destructive'
}

function diagnosisLabel(value: DeliveryAttempt['diagnosis']['class']): string {
  return value.replace(/-/g, ' ')
}

function shortSHA(value = ''): string {
  return value ? value.slice(0, 9) : ''
}

function observedLabel(value = ''): string {
  if (!value) return 'observation time unknown'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? 'observation time unknown' : date.toLocaleString()
}
</script>

<template>
  <section aria-labelledby="delivery-timeline-heading" class="space-y-3">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <p class="mb-1 font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
          Attempt evidence
        </p>
        <h2 id="delivery-timeline-heading" class="text-sm font-semibold text-foreground">
          Delivery waterfall
        </h2>
      </div>
      <label
        class="grid gap-1 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground"
      >
        Inspect task
        <select
          :value="selectedTask"
          class="min-w-56 rounded-md border border-border bg-background px-3 py-2 text-xs font-medium normal-case tracking-normal text-foreground"
          @change="emit('select', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">Select a task</option>
          <option v-for="task in tasks" :key="task.id" :value="task.id">
            {{ String(task.seq).padStart(3, '0') }} · {{ task.title }}
          </option>
        </select>
      </label>
    </div>

    <div
      v-if="timeline?.refusal"
      role="alert"
      class="rounded-md border border-destructive/40 bg-destructive/5 px-4 py-3 text-xs text-destructive"
    >
      <strong>Evidence refused.</strong> {{ timeline.refusal }}
    </div>

    <p
      v-if="timeline && selectedTask"
      class="rounded-md border border-border/70 bg-muted/25 px-4 py-3 text-xs leading-relaxed text-muted-foreground"
      aria-live="polite"
    >
      {{ timeline.summary }}
    </p>

    <div
      v-if="!selectedTask"
      class="rounded-lg border border-dashed border-border p-6 text-sm text-muted-foreground"
    >
      Select a task to inspect attempt-level execution, verification, review, PR, CI, merge, and
      acceptance evidence.
    </div>
    <div
      v-else-if="timeline && attempts.length === 0"
      class="rounded-lg border border-dashed border-border p-6 text-sm text-muted-foreground"
    >
      {{ timeline.summary }}
    </div>

    <article
      v-for="attempt in attempts"
      v-else
      :key="attempt.run_id"
      class="overflow-hidden rounded-lg border border-border bg-card"
    >
      <header
        class="flex flex-wrap items-start justify-between gap-3 border-b border-border px-4 py-3"
      >
        <div>
          <div class="flex flex-wrap items-center gap-2">
            <span class="font-mono text-[10px] uppercase tracking-[0.1em] text-muted-foreground"
              >Attempt {{ attempt.attempt }}</span
            >
            <span
              class="rounded-full border border-border px-2 py-0.5 text-[10px] font-semibold uppercase"
              >{{ attempt.outcome }}</span
            >
            <span
              v-if="attempt.recovered"
              class="rounded-full bg-warning/10 px-2 py-0.5 text-[10px] font-semibold uppercase text-warning"
              >Recovered</span
            >
          </div>
          <p class="mt-1 text-xs text-foreground">
            {{ attempt.role }} · {{ attempt.runtime
            }}<template v-if="attempt.model"> / {{ attempt.model }}</template>
          </p>
          <p class="mt-1 font-mono text-[10px] text-muted-foreground">{{ attempt.run_id }}</p>
        </div>
        <div class="text-right text-[10px] text-muted-foreground">
          <p v-if="attempt.usage.available">
            {{ attempt.usage.output_tokens.toLocaleString() }} output tokens · ${{
              attempt.usage.cost_usd.toFixed(3)
            }}
          </p>
          <p v-else>provider usage unavailable</p>
          <p class="mt-1">generation {{ attempt.generation }}</p>
        </div>
      </header>

      <div class="grid gap-3 border-b border-border px-4 py-3 md:grid-cols-[auto_minmax(0,1fr)]">
        <span
          class="h-fit rounded-full border px-2.5 py-1 font-mono text-[10px] font-semibold uppercase"
          :class="diagnosisTone(attempt.diagnosis.class)"
        >
          {{ diagnosisLabel(attempt.diagnosis.class) }}
        </span>
        <div class="min-w-0 text-xs">
          <p class="m-0 text-foreground">{{ attempt.diagnosis.detail }}</p>
          <p class="mt-1 mb-0 text-muted-foreground">Next: {{ attempt.diagnosis.next_action }}</p>
          <p class="mt-1 mb-0 break-all font-mono text-[10px] text-muted-foreground">
            {{ attempt.identity.branch }} · commit
            {{ attempt.identity.commit_sha || 'unobserved' }} · tree
            {{ attempt.identity.tree_sha || 'unobserved' }}
          </p>
        </div>
      </div>

      <ol class="hidden grid-cols-10 gap-px bg-border md:grid" aria-label="Delivery phases">
        <li v-for="(span, index) in attempt.spans" :key="span.phase" class="min-w-0 bg-card">
          <button
            :ref="(el) => rememberPhase(el, index)"
            type="button"
            class="phase-cell h-full w-full px-2 py-3 text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-inset"
            :class="`phase-${span.status}`"
            :title="`${span.phase}: ${span.detail} Source: ${span.source}. Freshness: ${span.freshness}. Duration: ${duration(span.duration_ms)}. Next: ${span.next_action}`"
            @keydown="movePhase($event, index)"
          >
            <span class="block truncate font-mono text-[9px] uppercase tracking-[0.08em]">{{
              span.phase
            }}</span>
            <span class="mt-2 block text-[10px] font-semibold uppercase">{{ span.status }}</span>
            <span class="mt-1 block text-[9px] text-muted-foreground">{{
              duration(span.duration_ms)
            }}</span>
            <span
              v-if="span.verdict"
              class="mt-1 block truncate text-[9px] font-medium text-foreground/75"
            >
              {{ span.verdict
              }}<template v-if="span.correction"> · correction {{ span.correction }}</template>
            </span>
          </button>
        </li>
      </ol>

      <ol class="divide-y divide-border md:hidden" aria-label="Delivery phases mobile">
        <li
          v-for="span in attempt.spans"
          :key="span.phase"
          class="flex items-start gap-3 px-4 py-3"
        >
          <span
            class="mt-1 h-2.5 w-2.5 shrink-0 rounded-full"
            :class="`phase-dot-${span.status}`"
          />
          <div class="min-w-0 flex-1">
            <div class="flex items-center justify-between gap-2">
              <span class="font-mono text-[10px] font-semibold uppercase tracking-[0.08em]">{{
                span.phase
              }}</span>
              <span class="text-[10px] uppercase text-muted-foreground">{{ span.status }}</span>
            </div>
            <p class="mt-1 text-xs text-muted-foreground">{{ span.detail }}</p>
            <p v-if="span.verdict || span.contract" class="mt-1 text-[10px] text-foreground/75">
              <template v-if="span.verdict">{{ span.verdict }}</template>
              <template v-if="span.correction"> · correction {{ span.correction }}</template>
              <template v-if="span.contract"> · {{ span.contract }}</template>
            </p>
            <p class="mt-1 text-[10px] text-muted-foreground">
              {{ duration(span.duration_ms) }} · {{ span.source }}
            </p>
          </div>
        </li>
      </ol>

      <footer
        class="flex flex-wrap gap-x-4 gap-y-2 border-t border-border px-4 py-3 text-[10px] font-medium"
      >
        <a
          :href="
            dashboardHref('work', {
              project: timeline?.task.project,
              task: attempt.identity.task_id,
            })
          "
          class="text-primary hover:underline"
          >Task</a
        >
        <a
          :href="dashboardHref('agents', { agent: attempt.agent_id })"
          class="text-primary hover:underline"
          >Agent</a
        >
        <a
          :href="`/api/agents/diff?run=${encodeURIComponent(attempt.run_id)}`"
          target="_blank"
          rel="noreferrer"
          class="text-primary hover:underline"
          >Diff evidence</a
        >
        <a
          :href="dashboardHref('activity', { task: attempt.identity.task_id, kind: 'review' })"
          class="text-primary hover:underline"
          >Review events</a
        >
        <a
          :href="
            dashboardHref('delivery', {
              project: timeline?.task.project,
              task: attempt.identity.task_id,
            })
          "
          class="text-primary hover:underline"
          >Delivery evidence</a
        >
        <span class="ml-auto font-mono text-muted-foreground"
          >tree {{ attempt.identity.tree_sha || 'unobserved' }}</span
        >
      </footer>
      <div
        v-if="attempt.pull_requests?.length"
        class="flex flex-wrap items-center gap-2 border-t border-border/70 bg-muted/20 px-4 py-2 text-[10px]"
      >
        <span class="font-semibold uppercase tracking-[0.08em] text-muted-foreground"
          >PR generations</span
        >
        <a
          v-for="pr in attempt.pull_requests"
          :key="pr.url"
          :href="pr.url"
          target="_blank"
          rel="noreferrer"
          :title="`Durable PR observation: ${observedLabel(pr.observed_at)}`"
          class="rounded-full border border-border px-2 py-0.5 font-mono text-primary hover:underline"
          :class="pr.state.startsWith('superseded') ? 'opacity-55' : ''"
        >
          {{ pr.generation ? `g${pr.generation}` : 'generation unknown' }} · {{ pr.state
          }}<template v-if="pr.merge_sha"> · merge {{ shortSHA(pr.merge_sha) }}</template>
        </a>
      </div>
    </article>
  </section>
</template>

<style scoped>
.phase-complete {
  box-shadow: inset 0 3px 0 hsl(var(--primary));
}
.phase-current {
  background: hsl(var(--primary) / 0.08);
  box-shadow: inset 0 3px 0 hsl(var(--primary));
}
.phase-refused {
  background: hsl(var(--destructive) / 0.07);
  box-shadow: inset 0 3px 0 hsl(var(--destructive));
}
.phase-pending,
.phase-unknown,
.phase-skipped {
  color: hsl(var(--muted-foreground));
}
.phase-dot-complete,
.phase-dot-current {
  background: hsl(var(--primary));
}
.phase-dot-refused {
  background: hsl(var(--destructive));
}
.phase-dot-pending,
.phase-dot-unknown,
.phase-dot-skipped {
  background: hsl(var(--muted-foreground) / 0.45);
}
</style>
