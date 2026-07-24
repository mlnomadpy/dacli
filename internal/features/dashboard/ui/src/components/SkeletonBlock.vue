<script setup lang="ts">
// A single loading placeholder — a `--panel` block with a subtle shimmer. The
// shimmer is an animation, so `prefers-reduced-motion` disables it globally
// (tokens.css) and it becomes a static block (DESIGN.md §7, §8). Decorative.
withDefaults(defineProps<{ height?: string; width?: string }>(), {
  height: '16px',
  width: '100%',
})
</script>

<template>
  <span class="skeleton" :style="{ height, width }" aria-hidden="true" />
</template>

<style scoped>
.skeleton {
  display: block;
  border-radius: 6px;
  background: var(--panel);
  border: 1px solid var(--border);
  position: relative;
  overflow: hidden;
}
.skeleton::after {
  content: '';
  position: absolute;
  inset: 0;
  transform: translateX(-100%);
  background: linear-gradient(90deg, transparent, rgba(230, 233, 239, 0.06), transparent);
  animation: shimmer 1.4s infinite;
}
@keyframes shimmer {
  100% {
    transform: translateX(100%);
  }
}
</style>
