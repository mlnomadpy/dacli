<script setup lang="ts">
import { TooltipProvider, type TooltipProviderProps } from 'reka-ui'

const props = withDefaults(defineProps<TooltipProviderProps>(), {
  delayDuration: 0,
})
</script>

<template>
  <TooltipProvider data-slot="tooltip-provider" v-bind="props">
    <slot />
  </TooltipProvider>
</template>
