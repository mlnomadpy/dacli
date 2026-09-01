<script setup lang="ts">
defineProps<{ path: string }>()
</script>

<template>
  <section
    role="alert"
    class="rounded-xl border border-warning/40 bg-card p-6 shadow-[0_24px_70px_-58px_rgba(243,201,121,0.7)]"
  >
    <p class="m-0 font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-warning">
      Unknown workspace area
    </p>
    <h2 id="route-heading" tabindex="-1" class="mt-2 mb-0 text-xl font-semibold focus:outline-none">
      This dashboard route does not exist
    </h2>
    <p class="mt-3 mb-0 text-sm text-muted-foreground">
      <span class="font-mono text-foreground">/{{ path }}</span> was rejected without loading a
      hidden workspace surface.
    </p>
    <a
      href="#/overview"
      class="mt-5 inline-flex min-h-11 items-center rounded-md bg-primary px-4 text-xs font-semibold text-primary-foreground no-underline"
    >
      Open overview
    </a>
  </section>
</template>
