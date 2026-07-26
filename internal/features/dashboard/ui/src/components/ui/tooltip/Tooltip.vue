<script setup lang="ts">
import { TooltipRoot, type TooltipRootProps, useForwardPropsEmits } from 'reka-ui'
import type { TooltipRootEmits } from 'reka-ui'
import TooltipProvider from './TooltipProvider.vue'

const props = defineProps<TooltipRootProps>()
const emits = defineEmits<TooltipRootEmits>()

const forwarded = useForwardPropsEmits(props, emits)
</script>

<template>
  <TooltipProvider>
    <TooltipRoot data-slot="tooltip" v-bind="forwarded">
      <slot />
    </TooltipRoot>
  </TooltipProvider>
</template>
