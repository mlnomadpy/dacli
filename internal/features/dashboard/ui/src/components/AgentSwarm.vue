<script setup lang="ts">
import { computed } from 'vue'
import type { Agent, Phase } from '@/types'
import { sectionState } from '@/composables/useSectionState'
import AgentRow from '@/components/AgentRow.vue'
import AgentCard from '@/components/AgentCard.vue'
import EmptyPanel from '@/components/EmptyPanel.vue'
import ErrorPanel from '@/components/ErrorPanel.vue'
import SkeletonBlock from '@/components/SkeletonBlock.vue'
import { Table, TableBody, TableHead, TableHeader, TableRow } from '@/components/ui/table'

// Owns the swarm's four states (DESIGN.md §7.3). Empty ("no live agents") is the
// common resting state and reads calm, never as an error. Live is a real
// <table> with header scope for screen readers; rows keep the server's
// newest-first order (no client sort in v1). The table is the one element
// allowed to scroll horizontally on mobile, inside its own wrapper.
const props = defineProps<{
  agents: Agent[]
  phase: Phase
  hasSnapshot: boolean
  error: string | null
}>()
const emit = defineEmits<{ retry: [] }>()

const state = computed(() =>
  sectionState(props.phase, props.hasSnapshot, props.agents.length === 0),
)

const headClass = 'h-auto py-2 text-[10px] uppercase tracking-[0.05em]'
</script>

<template>
  <div v-if="state === 'loading'" class="skeleton-table flex flex-col gap-1.5" aria-hidden="true">
    <SkeletonBlock v-for="n in 4" :key="n" height="32px" />
  </div>
  <ErrorPanel
    v-else-if="state === 'error'"
    :message="`couldn't load agents — ${error ?? 'unknown error'}`"
    @retry="emit('retry')"
  />
  <EmptyPanel v-else-if="state === 'empty'">no live agents</EmptyPanel>
  <div v-else>
    <div data-layout="mobile-cards" class="grid gap-3 md:hidden">
      <AgentCard v-for="agent in agents" :key="agent.run_id" :agent="agent" />
    </div>
    <div
      data-layout="desktop-table"
      class="hidden overflow-hidden rounded-lg border border-border md:block"
    >
      <Table class="bg-card">
        <TableHeader>
          <TableRow>
            <TableHead scope="col" :class="[headClass, 'run-h sticky left-0 bg-card']"
              >run</TableHead
            >
            <TableHead scope="col" :class="headClass">child</TableHead>
            <TableHead scope="col" :class="headClass">task</TableHead>
            <TableHead scope="col" :class="headClass">role</TableHead>
            <TableHead scope="col" :class="headClass">runtime</TableHead>
            <TableHead scope="col" :class="headClass">state</TableHead>
            <TableHead scope="col" :class="headClass">pid</TableHead>
            <TableHead scope="col" :class="headClass">uptime</TableHead>
            <TableHead scope="col" :class="headClass">last activity</TableHead>
            <TableHead scope="col" :class="headClass">detail</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <AgentRow v-for="a in agents" :key="a.run_id" :agent="a" />
        </TableBody>
      </Table>
    </div>
  </div>
</template>
