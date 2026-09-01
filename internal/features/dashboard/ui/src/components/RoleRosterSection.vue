<script setup lang="ts">
import { computed } from 'vue'
import type { Phase, Role } from '@/types'
import RoleRoster from '@/components/RoleRoster.vue'

// The Team roster section shell (dacli 226). Its header counts the roles and, in
// the same breath, how many are AT their WIP cap — the single number that
// explains a queue that has stopped moving, which previously required reading
// `.dacli/roles/*.md` and `.dacli/agents/*.md` by hand to discover.
const props = defineProps<{
  roles: Role[]
  phase: Phase
  hasSnapshot: boolean
  error: string | null
}>()
const emit = defineEmits<{ retry: [] }>()

const cappedOut = computed(() => props.roles.filter((r) => r.wip_exceeded).length)
</script>

<template>
  <section aria-labelledby="roster-h">
    <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
      <h2
        id="roster-h"
        class="m-0 text-[13px] font-semibold uppercase tracking-[0.06em] text-muted-foreground"
      >
        Team roster
      </h2>
      <span v-if="props.roles.length > 0" class="text-xs text-muted-foreground">
        {{ props.roles.length }} roles<template v-if="cappedOut > 0">
          &middot;
          <span class="capped text-destructive">{{ cappedOut }} at WIP cap</span></template
        >
      </span>
    </div>
    <p
      v-if="props.roles.length > 0"
      class="mb-2 text-right font-mono text-[10px] uppercase tracking-[0.08em] text-primary md:hidden"
    >
      scroll table for policy fields →
    </p>
    <RoleRoster
      :roles="props.roles"
      :phase="props.phase"
      :has-snapshot="props.hasSnapshot"
      :error="props.error"
      @retry="emit('retry')"
    />
  </section>
</template>
