<script setup lang="ts">
defineProps<{ pendingEvents: number }>()
</script>

<template>
  <section
    aria-labelledby="activity-inbox-h"
    class="grid gap-4 rounded-xl border border-border bg-card/90 p-5 shadow-[0_24px_70px_-58px_rgba(121,166,255,0.75)] md:grid-cols-[minmax(0,1fr)_auto] md:items-center"
  >
    <div>
      <p class="m-0 font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-primary">
        Durable event inbox
      </p>
      <h3 id="activity-inbox-h" class="mt-2 mb-0 text-lg font-semibold">
        {{ pendingEvents }} pending event{{ pendingEvents === 1 ? '' : 's' }}
      </h3>
      <p class="mt-2 mb-0 max-w-2xl text-sm leading-relaxed text-muted-foreground">
        This count is the only journal fact projected on this route today. The dashboard does not
        fabricate chronology or mark events applied; reconciliation remains an explicit CLI or MCP
        action.
      </p>
    </div>
    <a
      href="#/overview"
      class="inline-flex min-h-11 items-center justify-center rounded-md border border-border bg-background px-4 text-xs font-semibold text-foreground no-underline transition-colors hover:bg-secondary"
    >
      Return to overview
    </a>
  </section>
</template>
