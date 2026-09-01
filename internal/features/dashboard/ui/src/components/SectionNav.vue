<script setup lang="ts">
import {
  DASHBOARD_ROUTES,
  dashboardHref,
  type DashboardSelection,
} from '@/composables/useDashboardRoute'
import type { DashboardRouteName } from '@/stores/dashboard'

const props = defineProps<{
  current: DashboardRouteName
  selection: DashboardSelection
}>()

function href(name: (typeof DASHBOARD_ROUTES)[number]['name']): string {
  // Investigation context survives route changes. A route that cannot apply a
  // filter says so explicitly in the observation strip instead of silently
  // dropping it and making Back/Forward disagree (issue #950).
  return dashboardHref(name, props.selection)
}
</script>

<template>
  <nav
    aria-label="Workspace areas"
    class="section-nav sticky top-3 z-20 my-3 grid grid-cols-3 gap-1 rounded-lg border border-border bg-background/92 p-1 shadow-[0_14px_36px_-28px_#000] backdrop-blur-md md:grid-cols-6 lg:my-0 lg:grid-cols-1 lg:p-1.5"
  >
    <a
      v-for="route in DASHBOARD_ROUTES"
      :key="route.name"
      :href="href(route.name)"
      :aria-current="current === route.name ? 'page' : undefined"
    >
      <span>{{ route.number }}</span
      >{{ route.label }}
    </a>
  </nav>
</template>

<style scoped>
.section-nav a {
  display: inline-flex;
  min-height: 40px;
  min-width: 0;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border: 1px solid transparent;
  border-radius: 0.3rem;
  padding: 0 0.6rem;
  color: var(--muted-foreground);
  font-size: 0.72rem;
  font-weight: 600;
  text-decoration: none;
  transition:
    background-color 150ms ease,
    border-color 150ms ease,
    color 150ms ease;
}
.section-nav a span {
  color: var(--primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.58rem;
}
.section-nav a:hover {
  background: var(--secondary);
  color: var(--foreground);
}
.section-nav a[aria-current='page'] {
  border-color: color-mix(in srgb, var(--primary) 36%, transparent);
  background: color-mix(in srgb, var(--primary) 12%, var(--card));
  color: var(--foreground);
}
.section-nav a:focus-visible {
  outline: 2px solid var(--ring);
  outline-offset: 2px;
}

@media (max-width: 420px) {
  .section-nav a {
    min-height: 44px;
    padding-inline: 0.35rem;
    font-size: 0.68rem;
  }
}

@media (min-width: 1024px) {
  .section-nav a {
    min-height: 52px;
    flex-direction: column;
    gap: 0.1rem;
    padding-inline: 0.4rem;
    font-size: 0.66rem;
  }
  .section-nav a span {
    font-size: 0.52rem;
  }
}
</style>
