<script setup lang="ts">
import type { ChartContract } from '@/types'

defineProps<{ contract: ChartContract }>()
</script>

<template>
  <section :aria-labelledby="`${contract.id}-title`" class="chart-contract min-w-0">
    <header class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
      <div class="min-w-0">
        <p class="m-0 text-[10px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
          {{ contract.metric_definition }}
        </p>
        <h3 :id="`${contract.id}-title`" class="mt-1 mb-0 text-sm font-semibold">
          {{ contract.title }}
        </h3>
      </div>
      <span
        class="w-fit rounded-full bg-muted px-2 py-1 text-[10px] font-semibold uppercase tracking-wide text-muted-foreground"
      >
        {{ contract.state }}
      </span>
    </header>

    <dl class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-[10px] leading-4 text-muted-foreground">
      <div>
        <dt class="inline font-semibold text-foreground">Unit</dt>
        <dd class="inline">{{ contract.unit }}</dd>
      </div>
      <div>
        <dt class="inline font-semibold text-foreground">Window</dt>
        <dd class="inline">{{ contract.window }}</dd>
      </div>
      <div>
        <dt class="inline font-semibold text-foreground">Source</dt>
        <dd class="inline">{{ contract.source }}</dd>
      </div>
      <div>
        <dt class="inline font-semibold text-foreground">Freshness</dt>
        <dd class="inline">{{ contract.freshness }}</dd>
      </div>
      <div>
        <dt class="inline font-semibold text-foreground">Coverage</dt>
        <dd class="inline">{{ contract.coverage }}</dd>
      </div>
      <div v-if="contract.comparison">
        <dt class="inline font-semibold text-foreground">Comparison</dt>
        <dd class="inline">{{ contract.comparison }}</dd>
      </div>
    </dl>

    <p
      v-if="
        contract.state === 'stale' || contract.state === 'partial' || contract.state === 'error'
      "
      class="mt-2 mb-0 rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-800 dark:text-amber-200"
      role="status"
    >
      {{ contract.state_detail }}
    </p>
    <p class="sr-only" :id="`${contract.id}-summary`">{{ contract.summary }}</p>

    <div class="mt-3" :aria-describedby="`${contract.id}-summary`">
      <slot />
    </div>

    <details
      v-if="contract.points.length"
      class="mt-3 rounded-md border border-border bg-background/50 text-xs"
    >
      <summary class="cursor-pointer px-3 py-2 font-medium">
        Accessible data table · {{ contract.points.length }} visible point(s)<template
          v-if="contract.hidden_resolution"
        >
          · {{ contract.hidden_resolution }}</template
        >
      </summary>
      <div class="max-h-56 overflow-auto border-t border-border">
        <table class="w-full min-w-[480px] border-collapse text-left">
          <thead class="sticky top-0 bg-muted">
            <tr>
              <th class="px-3 py-2">Point</th>
              <th class="px-3 py-2">Value</th>
              <th class="px-3 py-2">Evidence</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="point in contract.points" :key="point.id" class="border-t border-border">
              <th class="px-3 py-2 font-medium">{{ point.label }}</th>
              <td class="px-3 py-2 tabular-nums">
                {{ point.value === null ? 'Missing' : point.display }}
              </td>
              <td class="px-3 py-2">
                <a
                  v-if="point.href"
                  :href="point.href"
                  class="font-medium text-primary underline-offset-2 hover:underline"
                  >Open {{ point.evidence_count }} record(s)</a
                ><span v-else class="text-muted-foreground">No exact evidence link</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </details>
  </section>
</template>
