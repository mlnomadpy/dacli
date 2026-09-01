<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
  containerLabel?: string
}>()
</script>

<template>
  <div
    data-slot="table-container"
    :role="props.containerLabel ? 'region' : undefined"
    :aria-label="props.containerLabel"
    :tabindex="props.containerLabel ? 0 : undefined"
    class="relative w-full overflow-x-auto focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-ring"
  >
    <table data-slot="table" :class="cn('w-full caption-bottom text-sm', props.class)">
      <slot />
    </table>
  </div>
</template>
