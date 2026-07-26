<script setup lang="ts">
import { computed } from 'vue'
import type { Status } from '@/types'
import { statusColor } from '@/composables/useStatusTheme'
import { Card } from '@/components/ui/card'

// One status column of the task board (DESIGN.md §7.2). The board is
// COUNT-driven — /api/state carries per-status counts, not task rows — so the
// chips visualize magnitude, not identity. The header count is always the true
// total; the chip run is capped so a 500-task column can't blow out the layout.
// A zero column still renders its header and an em-dash, never a blank gap.
const CHIP_CAP = 24

const props = defineProps<{ status: Status; count: number }>()

const chips = computed(() => Math.min(props.count, CHIP_CAP))
const overflow = computed(() => Math.max(0, props.count - CHIP_CAP))
</script>

<template>
  <Card
    role="group"
    :aria-label="`${status} — ${count} tasks`"
    class="min-w-0 gap-2.5 rounded-lg p-3"
  >
    <div class="flex items-center gap-1.5">
      <i
        class="size-2 shrink-0 rounded-full"
        :style="{ background: statusColor(status) }"
        aria-hidden="true"
      />
      <span class="text-[10px] font-semibold uppercase tracking-[0.05em] text-muted-foreground">{{
        status
      }}</span>
      <span class="count ml-auto text-sm font-semibold">{{ count }}</span>
    </div>
    <div v-if="count > 0" class="flex flex-wrap items-center gap-1" aria-hidden="true">
      <i
        v-for="n in chips"
        :key="n"
        class="chip size-2.5 rounded-sm border border-border bg-secondary"
      />
      <span v-if="overflow > 0" class="more ml-0.5 text-[11px] text-muted-foreground"
        >+{{ overflow }} more</span
      >
    </div>
    <div v-else class="none text-[13px] text-muted-foreground" aria-hidden="true">—</div>
  </Card>
</template>
