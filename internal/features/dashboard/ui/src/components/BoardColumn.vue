<script setup lang="ts">
import { computed } from 'vue'
import type { Status } from '@/types'
import { statusColor } from '@/composables/useStatusTheme'

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
  <div class="col" role="group" :aria-label="`${status} — ${count} tasks`">
    <div class="head">
      <i class="dot" :style="{ background: statusColor(status) }" aria-hidden="true" />
      <span class="label">{{ status }}</span>
      <span class="count">{{ count }}</span>
    </div>
    <div v-if="count > 0" class="chips" aria-hidden="true">
      <i v-for="n in chips" :key="n" class="chip" />
      <span v-if="overflow > 0" class="more">+{{ overflow }} more</span>
    </div>
    <div v-else class="none" aria-hidden="true">—</div>
  </div>
</template>

<style scoped>
.col {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 12px;
  min-width: 0;
}
.head {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 10px;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex: none;
}
.label {
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-size: 10px;
  font-weight: 600;
  color: var(--muted);
}
.count {
  margin-left: auto;
  font-size: 14px;
  font-weight: 600;
}
.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}
.chip {
  width: 10px;
  height: 10px;
  border-radius: 2px;
  background: var(--surface-2);
  border: 1px solid var(--border);
}
.more {
  font-size: 11px;
  color: var(--muted);
  margin-left: 2px;
}
.none {
  color: var(--muted);
  font-size: 13px;
}
</style>
