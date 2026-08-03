<script setup lang="ts">
import { computed } from 'vue'
import type { Phase, Role } from '@/types'
import { sectionState } from '@/composables/useSectionState'
import RoleRow from '@/components/RoleRow.vue'
import EmptyPanel from '@/components/EmptyPanel.vue'
import ErrorPanel from '@/components/ErrorPanel.vue'
import SkeletonBlock from '@/components/SkeletonBlock.vue'
import { Table, TableBody, TableHead, TableHeader, TableRow } from '@/components/ui/table'

// Owns the roster's four states, the same contract every other section follows
// (DESIGN.md §6.3). Empty is calm — a workspace can legitimately have no roles
// yet — never an error. Rows keep the server's name order (no client sort).
const props = defineProps<{
  roles: Role[]
  phase: Phase
  hasSnapshot: boolean
  error: string | null
}>()
const emit = defineEmits<{ retry: [] }>()

const state = computed(() => sectionState(props.phase, props.hasSnapshot, props.roles.length === 0))

const headClass = 'h-auto py-2 text-[10px] uppercase tracking-[0.05em]'
</script>

<template>
  <div v-if="state === 'loading'" class="skeleton-table flex flex-col gap-1.5" aria-hidden="true">
    <SkeletonBlock v-for="n in 4" :key="n" height="32px" />
  </div>
  <ErrorPanel
    v-else-if="state === 'error'"
    :message="`couldn't load roles — ${error ?? 'unknown error'}`"
    @retry="emit('retry')"
  />
  <EmptyPanel v-else-if="state === 'empty'">no roles defined yet</EmptyPanel>
  <div v-else class="overflow-hidden rounded-lg border border-border">
    <Table class="bg-card">
      <TableHeader>
        <TableRow>
          <TableHead scope="col" :class="[headClass, 'name-h sticky left-0 bg-card']"
            >role</TableHead
          >
          <TableHead scope="col" :class="headClass">summary</TableHead>
          <TableHead scope="col" :class="headClass">kind</TableHead>
          <TableHead scope="col" :class="headClass">grant</TableHead>
          <TableHead scope="col" :class="headClass">runtime / model</TableHead>
          <TableHead scope="col" :class="headClass" title="active agents / WIP cap"
            >wip</TableHead
          >
          <TableHead scope="col" :class="headClass">scope</TableHead>
          <TableHead scope="col" :class="headClass">skills</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <RoleRow v-for="r in roles" :key="r.name" :role="r" />
      </TableBody>
    </Table>
  </div>
</template>
