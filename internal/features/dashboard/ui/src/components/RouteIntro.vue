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
    class="route-intro flex min-h-11 flex-wrap items-baseline gap-x-3 gap-y-1 border-b border-border pb-2"
  >
    <div class="flex min-w-0 items-baseline gap-2">
      <p
        class="m-0 shrink-0 font-mono text-[9px] font-semibold uppercase tracking-[0.14em] text-primary"
      >
        {{ eyebrow }}
      </p>
      <h2
        id="route-heading"
        ref="heading"
        tabindex="-1"
        class="m-0 text-lg leading-tight font-semibold tracking-[-0.03em] focus:outline-none"
      >
        {{ title }}
      </h2>
    </div>
    <p class="m-0 min-w-0 flex-1 text-[11px] text-muted-foreground md:truncate">
      {{ description }}
    </p>
  </header>
</template>
