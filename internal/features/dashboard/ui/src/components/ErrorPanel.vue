<script setup lang="ts">
// The shared COLD-error panel (DESIGN.md §7): danger-tinted, a message, and a
// Retry button that re-polls. Only shown when a poll failed AND no snapshot has
// ever arrived; a later failure with a retained snapshot never reaches here.
import { Button } from '@/components/ui/button'

defineProps<{ message: string | null }>()
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <div
    role="alert"
    class="flex flex-wrap items-center gap-3 rounded-lg border border-destructive bg-card px-4 py-3.5 text-[13px] text-foreground"
  >
    <span>{{ message ?? 'unknown error' }}</span>
    <Button variant="outline" size="sm" class="retry min-w-11" @click="emit('retry')">
      Retry
    </Button>
  </div>
</template>
