<script setup lang="ts">
// The hexagon-cluster mark (four nested hexagons: one center, three satellites)
// — "many coordinated units, not chaos" (DESIGN.md §3.3). Decorative, so
// aria-hidden; inlined SVG with no new static-asset route.
</script>

<template>
  <svg class="mark" viewBox="0 0 64 64" fill="currentColor" aria-hidden="true">
    <polygon points="32,20 42.39,26 42.39,38 32,44 21.61,38 21.61,26" />
    <polygon points="32,5 38.06,8.5 38.06,15.5 32,19 25.94,15.5 25.94,8.5" />
    <polygon points="14.4,35 20.46,38.5 20.46,45.5 14.4,49 8.34,45.5 8.34,38.5" />
    <polygon points="49.6,35 55.66,38.5 55.66,45.5 49.6,49 43.54,45.5 43.54,38.5" />
  </svg>
</template>

<style scoped>
.mark {
  width: 20px;
  height: 20px;
  flex: none;
  color: var(--accent);
}
</style>
