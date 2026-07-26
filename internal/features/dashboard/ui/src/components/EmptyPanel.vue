<script setup lang="ts">
// A calm, bordered "nothing here yet" panel — the shared empty state for every
// surface (DESIGN.md §7). Muted text, never a danger color: an empty swarm is
// the common resting state and must read as normal, not as an error.
import { Card } from '@/components/ui/card'
</script>

<template>
  <Card class="gap-0 rounded-lg py-0 text-[13px] text-muted-foreground shadow-none">
    <p class="m-0 p-4"><slot /></p>
  </Card>
</template>
