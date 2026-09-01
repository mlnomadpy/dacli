<script setup lang="ts">
import type { Status, TaskSummary } from '@/types'
import { statusColor } from '@/composables/useStatusTheme'
import { Card } from '@/components/ui/card'

defineProps<{ status: Status; count: number; tasks: TaskSummary[] }>()
const emit = defineEmits<{ inspect: [task: string, trigger: HTMLElement] }>()
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
    <p v-if="tasks.length !== count" class="text-[10px] text-muted-foreground">
      {{ tasks.length }} matching {{ count }} total
    </p>
    <ul v-if="tasks.length" class="max-h-72 space-y-1.5 overflow-y-auto pr-1">
      <li v-for="task in tasks" :key="task.id">
        <button
          type="button"
          class="w-full rounded-md border border-border/70 bg-background px-2.5 py-2 text-left hover:border-primary/50 hover:bg-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          :aria-label="`Inspect ${task.id}: ${task.title}`"
          @click="emit('inspect', task.id, $event.currentTarget as HTMLElement)"
        >
          <span class="block truncate text-xs font-semibold"
            >{{ String(task.seq).padStart(3, '0') }} · {{ task.title }}</span
          >
          <span class="mt-1 block truncate font-mono text-[9px] text-muted-foreground">
            {{ task.owner || 'unowned' }} ·
            {{ task.estimated ? `Te ${task.points}` : 'unestimated' }}
          </span>
        </button>
      </li>
    </ul>
    <div v-else class="none text-[11px] text-muted-foreground">
      {{ count === 0 ? 'no tasks' : 'no matching tasks' }}
    </div>
  </Card>
</template>
