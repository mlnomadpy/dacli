<script setup lang="ts">
import { ref } from 'vue'

defineProps<{
  eyebrow: string
  title: string
  description: string
}>()

const heading = ref<HTMLHeadingElement | null>(null)
defineExpose({ focus: () => heading.value?.focus() })
</script>

<template>
  <header
    class="route-intro flex flex-wrap items-end justify-between gap-4 border-b border-border pb-4"
  >
    <div class="min-w-0">
      <p class="m-0 font-mono text-[10px] font-semibold uppercase tracking-[0.16em] text-primary">
        {{ eyebrow }}
      </p>
      <h2
        id="route-heading"
        ref="heading"
        tabindex="-1"
        class="mt-1 mb-0 text-[clamp(1.35rem,2.4vw,1.9rem)] leading-tight font-semibold tracking-[-0.035em] focus:outline-none"
      >
        {{ title }}
      </h2>
      <p class="mt-2 mb-0 max-w-3xl text-sm leading-relaxed text-muted-foreground">
        {{ description }}
      </p>
    </div>
    <span
      class="rounded-sm border border-border bg-card px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground"
    >
      read-only
    </span>
  </header>
</template>
