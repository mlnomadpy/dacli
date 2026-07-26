<script setup lang="ts">
// A single loading placeholder — a `bg-card` block with a subtle pulse. The
// pulse is an animation, so `prefers-reduced-motion` disables it globally
// (tokens.css) and it becomes a static block (DESIGN.md §7, §8). Decorative.
withDefaults(defineProps<{ height?: string; width?: string }>(), {
  height: '16px',
  width: '100%',
})
</script>

<template>
  <span
    class="skeleton block animate-pulse rounded-md border border-border bg-card"
    :style="{ height, width }"
    aria-hidden="true"
  />
</template>
