<script setup lang="ts">
import { computed } from 'vue'
import type { Burndown } from '@/types'

// The single stacked done/remaining bar on a ProjectCard (DESIGN.md §7.1).
// 0-safe: when nothing is estimated (`total === 0`) it renders the empty border
// track at full width and captions "no estimated points yet" — never a divide
// by zero, never a NaN width.
const props = defineProps<{ burndown: Burndown }>()

const total = computed(() => props.burndown.done_points + props.burndown.remaining_points)
const donePct = computed(() =>
  total.value > 0 ? (props.burndown.done_points / total.value) * 100 : 0,
)

const caption = computed(() => {
  const b = props.burndown
  if (total.value === 0) return 'no estimated points yet'
  let s = `${b.done_points.toFixed(1)} done · ${b.remaining_points.toFixed(1)} remaining pts`
  if (b.unestimated > 0) s += ` · ${b.unestimated} unestimated`
  return s
})

const ariaLabel = computed(() =>
  total.value === 0
    ? 'no estimated points yet'
    : `burndown: ${props.burndown.done_points.toFixed(1)} done of ${total.value.toFixed(1)} points`,
)
</script>

<template>
  <div>
    <div
      class="mb-1.5 flex h-2 overflow-hidden rounded-full bg-border"
      role="img"
      :aria-label="ariaLabel"
    >
      <template v-if="total > 0">
        <div class="done-seg bg-success" :style="{ width: donePct + '%' }" />
        <div class="rem-seg bg-primary" :style="{ width: 100 - donePct + '%' }" />
      </template>
    </div>
    <p class="m-0 text-[11px] text-muted-foreground">{{ caption }}</p>
  </div>
</template>
