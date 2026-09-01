<script setup lang="ts">
import { computed } from 'vue'
import type { LoopOperationResponse, LoopTokenAmount } from '@/types'
import { Badge } from '@/components/ui/badge'

const props = defineProps<{ operation: LoopOperationResponse }>()

const stateLabel = computed(() => props.operation.state.value.replace(/-/g, ' '))
const stateTone = computed(() => {
  if (['running', 'completed'].includes(props.operation.state.value)) return 'ok'
  if (['corrupt', 'halted-policy', 'externally-unknown'].includes(props.operation.state.value))
    return 'danger'
  return 'warn'
})
const refusedPreflight = computed(() =>
  props.operation.preflight.filter((phase) => phase.classification !== 'pass'),
)
const operationalPreflight = computed(() =>
  props.operation.preflight.filter((phase) =>
    [
      'cycle-wip',
      'implementation-wip',
      'reviewer-runtime',
      'implementation-runtime',
      'verification-command',
    ].includes(phase.phase),
  ),
)

function tokens(value?: number | null): string {
  return value === null || value === undefined ? 'not enforceable' : value.toLocaleString()
}

function budgetPct(amount: LoopTokenAmount): number {
  if (amount.limit <= 0) return 0
  return Math.min(100, ((amount.spent + amount.reserved) / amount.limit) * 100)
}

function taskHref(task: string): string {
  const query = new URLSearchParams({ project: props.operation.project, task })
  return `#/work?${query.toString()}`
}
</script>

<template>
  <section aria-labelledby="loop-operations-heading" class="space-y-3">
    <article
      class="overflow-hidden rounded-lg border bg-card"
      :class="{
        'border-destructive/60': stateTone === 'danger',
        'border-warning/50': stateTone === 'warn',
        'border-primary/35': stateTone === 'ok',
      }"
    >
      <div class="grid gap-4 px-4 py-4 md:grid-cols-[minmax(0,1.5fr)_minmax(260px,0.8fr)]">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <p class="m-0 font-mono text-[10px] uppercase tracking-[0.12em] text-primary">
              Loop operation · {{ operation.project }}
            </p>
            <Badge variant="outline" class="capitalize" :data-tone="stateTone">{{
              stateLabel
            }}</Badge>
            <Badge variant="outline">{{ operation.state.freshness }}</Badge>
          </div>
          <h2 id="loop-operations-heading" class="mt-2 mb-0 text-xl font-semibold tracking-tight">
            {{ operation.state.next_action }}
          </h2>
          <p
            v-if="operation.state.reason"
            class="mt-2 mb-0 max-w-3xl text-xs text-muted-foreground"
          >
            {{ operation.state.reason }}
          </p>
        </div>
        <dl class="m-0 grid grid-cols-2 gap-px overflow-hidden rounded-md border bg-border text-xs">
          <div class="bg-background px-3 py-2.5">
            <dt class="text-muted-foreground">Cycle / generation</dt>
            <dd class="mt-1 font-mono font-semibold">
              {{ operation.state.cycle }} / {{ operation.state.generation || '—' }}
            </dd>
          </div>
          <div class="bg-background px-3 py-2.5">
            <dt class="text-muted-foreground">Current phase</dt>
            <dd class="mt-1 font-mono font-semibold">{{ operation.state.phase || '—' }}</dd>
          </div>
          <div class="bg-background px-3 py-2.5">
            <dt class="text-muted-foreground">Wave</dt>
            <dd class="mt-1 font-mono font-semibold">
              {{ operation.wave.live_width }} live · {{ operation.wave.allocated_width }}/{{
                operation.wave.requested_width
              }}
              allocated
            </dd>
          </div>
          <div class="bg-background px-3 py-2.5">
            <dt class="text-muted-foreground">Harness boundary</dt>
            <dd class="mt-1 font-mono font-semibold">
              {{ operation.harness.mode }} ·
              {{ operation.harness.allowed.join(', ') || 'unobserved' }}
            </dd>
          </div>
        </dl>
      </div>
      <div
        v-if="operation.state.halt_class"
        class="border-t border-border bg-muted/35 px-4 py-2.5 text-xs"
      >
        <span class="font-semibold">{{ operation.state.halt_class }}</span>
        <span class="text-muted-foreground">
          ·
          {{
            operation.state.retryable
              ? 'retryable after evidence changes'
              : 'policy answer — do not retry unchanged'
          }}</span
        >
      </div>
    </article>

    <div class="grid gap-3 lg:grid-cols-2">
      <article
        v-for="entry in [
          { label: 'Cycle budget', amount: operation.budget.cycle },
          { label: 'Rolling window', amount: operation.budget.rolling },
        ]"
        :key="entry.label"
        class="rounded-lg border border-border bg-card px-4 py-3.5"
      >
        <div class="flex items-center justify-between gap-3">
          <h3
            class="m-0 text-[11px] font-semibold uppercase tracking-[0.07em] text-muted-foreground"
          >
            {{ entry.label }}
          </h3>
          <Badge variant="outline">{{ operation.budget.mode || 'unknown' }}</Badge>
        </div>
        <div class="mt-3 grid grid-cols-3 gap-3 text-xs">
          <span
            ><small class="block text-muted-foreground">spent</small
            ><b class="font-mono">{{ tokens(entry.amount.spent) }}</b></span
          >
          <span
            ><small class="block text-muted-foreground">reserved</small
            ><b class="font-mono">{{ tokens(entry.amount.reserved) }}</b></span
          >
          <span
            ><small class="block text-muted-foreground">remaining</small
            ><b class="font-mono">{{ tokens(entry.amount.remaining) }}</b></span
          >
        </div>
        <div
          v-if="operation.budget.mode === 'enforceable' && entry.amount.limit > 0"
          class="mt-3 h-1.5 overflow-hidden rounded-full bg-muted"
          role="progressbar"
          :aria-label="`${entry.label} spent and reserved`"
          :aria-valuenow="budgetPct(entry.amount)"
          aria-valuemin="0"
          aria-valuemax="100"
        >
          <div class="h-full bg-primary" :style="{ width: `${budgetPct(entry.amount)}%` }" />
        </div>
      </article>
    </div>

    <p
      class="m-0 rounded-md border border-border bg-muted/30 px-3 py-2 text-[11px] text-muted-foreground"
    >
      {{ operation.budget.accounting_boundary }}
      <template v-if="operation.budget.window_reset_at">
        · resets {{ operation.budget.window_reset_at }}</template
      >
      · review reserve {{ operation.budget.review_reservation.toLocaleString() }} · delivery
      recovery reserve {{ operation.budget.recovery_reserve.toLocaleString() }}
      <template v-if="operation.budget.unknown_usage_runs.length">
        · {{ operation.budget.unknown_usage_runs.length }} run(s) have unknown usage</template
      >
    </p>

    <div class="grid gap-3 xl:grid-cols-[minmax(0,1.35fr)_minmax(300px,0.65fr)]">
      <article class="min-w-0 rounded-lg border border-border bg-card p-4">
        <div class="flex items-center justify-between gap-3">
          <h3 class="m-0 text-sm font-semibold">Current wave</h3>
          <span class="font-mono text-[10px] text-muted-foreground"
            >{{ operation.tasks.length }} task(s)</span
          >
        </div>
        <p v-if="operation.tasks.length === 0" class="mt-3 mb-0 text-xs text-muted-foreground">
          No durable planned or active task is recorded.
        </p>
        <ul v-else class="mt-3 grid list-none gap-2 p-0">
          <li
            v-for="task in operation.tasks"
            :key="task.task"
            class="rounded-md border border-border bg-background px-3 py-3"
          >
            <div class="flex flex-wrap items-start justify-between gap-2">
              <a
                :href="taskHref(task.task)"
                class="font-mono text-xs font-semibold text-primary underline-offset-4 hover:underline"
                >{{ task.task }}</a
              >
              <Badge variant="outline">{{ task.phase }}</Badge>
            </div>
            <p class="mt-2 mb-0 text-xs">
              <b>{{ task.role || 'unrouted' }}</b
              ><span class="text-muted-foreground">
                · {{ task.runtime || 'runtime unknown' }} / {{ task.model || 'model unknown' }} ·
                {{ task.grant || 'grant unknown' }}</span
              >
            </p>
            <p class="mt-1 mb-0 text-[11px] text-muted-foreground">
              {{ task.claim_count }} scoped claim(s)
              <template v-if="task.capacity_fit !== undefined">
                · capacity {{ task.capacity_fit ? 'fits' : 'refused' }}
                <span v-if="task.role_limit"
                  >({{ task.task_points }}/{{ task.role_limit }} points)</span
                ></template
              >
            </p>
            <p
              v-if="task.verification_argv"
              class="mt-2 mb-0 break-words rounded bg-muted px-2 py-1.5 font-mono text-[10px]"
            >
              {{ task.verification_cwd || '.' }} · {{ task.verification_argv }}
            </p>
            <p v-if="task.override" class="mt-2 mb-0 text-[11px] text-warning">
              {{ task.override }}
            </p>
          </li>
        </ul>
      </article>

      <div class="space-y-3">
        <article class="rounded-lg border border-border bg-card p-4">
          <h3 class="m-0 text-sm font-semibold">Routing evidence</h3>
          <p v-if="operation.routing.length === 0" class="mt-3 mb-0 text-xs text-muted-foreground">
            No current routing decision is recorded.
          </p>
          <details
            v-for="route in operation.routing"
            :key="route.task"
            class="mt-3 rounded-md border border-border px-3 py-2"
          >
            <summary class="cursor-pointer text-xs font-semibold">
              {{ route.task }} · {{ route.selected.role || 'no eligible selection' }}
            </summary>
            <p class="mt-2 mb-0 text-[11px] text-muted-foreground">
              {{ route.source }} · {{ route.freshness }}
            </p>
            <ul class="mt-2 mb-0 list-none space-y-1 p-0 text-[11px]">
              <li
                v-for="candidate in route.candidates"
                :key="candidate.role"
                class="flex items-start justify-between gap-2"
              >
                <span>{{ candidate.role }} · {{ candidate.runtime }}/{{ candidate.model }}</span>
                <span :class="candidate.eligible ? 'text-primary' : 'text-muted-foreground'">{{
                  candidate.eligible ? 'eligible' : candidate.exclusions?.join('; ')
                }}</span>
              </li>
            </ul>
          </details>
        </article>

        <article class="rounded-lg border border-border bg-card p-4">
          <div class="flex items-center justify-between gap-3">
            <h3 class="m-0 text-sm font-semibold">Preflight</h3>
            <Badge variant="outline">{{ refusedPreflight.length }} attention</Badge>
          </div>
          <ul class="mt-3 mb-0 list-none space-y-2 p-0 text-xs">
            <li
              v-for="phase in refusedPreflight"
              :key="`${phase.phase}-${phase.task}`"
              class="rounded-md bg-muted px-2.5 py-2"
            >
              <b>{{ phase.phase }}</b> · {{ phase.classification }}
              <p v-if="phase.remediation" class="mt-1 mb-0 text-muted-foreground">
                {{ phase.remediation }}
              </p>
            </li>
            <li v-if="refusedPreflight.length === 0" class="text-muted-foreground">
              No refused preflight phase is recorded.
            </li>
          </ul>
          <details v-if="operationalPreflight.length" class="mt-3 border-t border-border pt-3">
            <summary class="cursor-pointer text-[11px] font-semibold text-muted-foreground">
              Capacity, runtime, and verification evidence
            </summary>
            <ul class="mt-2 mb-0 list-none space-y-1.5 p-0 text-[11px]">
              <li
                v-for="phase in operationalPreflight"
                :key="`evidence-${phase.phase}-${phase.task}`"
              >
                <b>{{ phase.phase }}</b
                ><template v-if="phase.task"> · {{ phase.task }}</template
                ><span class="text-muted-foreground"> · {{ phase.evidence || phase.verdict }}</span>
              </li>
            </ul>
          </details>
        </article>
      </div>
    </div>

    <div
      v-if="operation.warnings.length"
      role="alert"
      class="rounded-md border border-destructive/45 bg-card px-4 py-3 text-xs text-destructive"
    >
      <b>Evidence warnings</b>
      <ul class="mt-1 mb-0 pl-4">
        <li v-for="warning in operation.warnings" :key="warning">{{ warning }}</li>
      </ul>
    </div>
  </section>
</template>

<style scoped>
[data-tone='danger'] {
  border-color: color-mix(in srgb, var(--destructive) 65%, transparent);
  color: var(--destructive);
}
[data-tone='warn'] {
  border-color: color-mix(in srgb, var(--warning) 65%, transparent);
  color: var(--warning);
}
[data-tone='ok'] {
  border-color: color-mix(in srgb, var(--primary) 55%, transparent);
  color: var(--primary);
}
summary::marker {
  color: var(--primary);
}
</style>
