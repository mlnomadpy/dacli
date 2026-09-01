<script setup lang="ts">
import { computed } from 'vue'
import type { Role } from '@/types'
import { TableCell, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'

// One role on the roster (dacli 226). Every column is a MECHANICAL fact about
// what the role can do — scope, grant, cost, caps — never a self-description,
// because a role that changes none of these is cosplay (internal/team's rule).
//
// The WIP cell is the one that yells: `active/cap` reads danger-red when the cap
// is reached, because that is the state in which the next spawn is refused, and
// an operator staring at a stalled queue needs to see the reason. The number is
// always the label, so color is never the only signal.
const props = defineProps<{ role: Role }>()
const emit = defineEmits<{ inspect: [name: string, trigger: HTMLElement] }>()

const wipLabel = computed(() =>
  props.role.wip > 0
    ? `${props.role.active_agents}/${props.role.wip}`
    : `${props.role.active_agents}`,
)
const wipTitle = computed(() =>
  props.role.wip > 0
    ? props.role.wip_exceeded
      ? `at the WIP cap — another spawn into ${props.role.name} is refused`
      : `${props.role.active_agents} of ${props.role.wip} slots in use`
    : `${props.role.active_agents} active — this role is uncapped`,
)
const wipClass = computed(() =>
  props.role.wip_exceeded ? 'text-destructive font-semibold' : 'text-muted-foreground',
)

/** Scope globs joined for the cell; an empty scope is a DECLARED absence of a
 * boundary (permissive by design), so it reads "everywhere" rather than "—". */
const scopeLabel = computed(() =>
  props.role.scope.length > 0 ? props.role.scope.join(' ') : 'everywhere',
)
const scopeTitle = computed(() => {
  const parts = [
    props.role.scope.length > 0 ? `in scope: ${props.role.scope.join(', ')}` : 'no declared scope',
  ]
  if (props.role.out_of_scope.length > 0) {
    parts.push(`out of scope: ${props.role.out_of_scope.join(', ')}`)
  }
  return parts.join('\n')
})

const skillsLabel = computed(() =>
  props.role.skills.length > 0 ? props.role.skills.join(', ') : '—',
)
/** Cost routing: a role with neither a runtime nor a model inherits the default. */
const costLabel = computed(
  () => [props.role.runtime, props.role.model].filter(Boolean).join(' / ') || '—',
)

function inspect(event: MouseEvent): void {
  emit('inspect', props.role.name, event.currentTarget as HTMLElement)
}
</script>

<template>
  <TableRow>
    <TableCell class="name sticky left-0 bg-card text-xs font-semibold">
      {{ role.name }}
      <span
        v-if="!role.has_prompt"
        class="ml-1.5 text-[10px] font-normal text-muted-foreground"
        title="metadata only — this role file carries no standing instructions, so its agents get the generic prompt"
        >(no prompt)</span
      >
    </TableCell>
    <TableCell class="text-xs text-muted-foreground">{{ role.summary || '—' }}</TableCell>
    <TableCell class="text-xs">
      <Badge
        v-if="role.kind"
        variant="outline"
        class="kind rounded-full text-[10px] font-semibold uppercase tracking-[0.04em]"
        title="the lifecycle phase gate acts on this"
        >{{ role.kind }}</Badge
      >
      <span v-else class="text-muted-foreground" title="no kind — works in any phase">any</span>
    </TableCell>
    <TableCell
      class="grant text-xs"
      :title="`spawns into this role default to ${role.grant || 'ro'}`"
      >{{ role.grant || 'ro' }}</TableCell
    >
    <TableCell class="text-xs">{{ costLabel }}</TableCell>
    <TableCell class="wip font-mono text-xs" :class="wipClass" :title="wipTitle">{{
      wipLabel
    }}</TableCell>
    <TableCell class="scope max-w-[22ch] truncate font-mono text-xs" :title="scopeTitle">{{
      scopeLabel
    }}</TableCell>
    <TableCell class="skills max-w-[18ch] truncate text-xs" :title="skillsLabel">{{
      skillsLabel
    }}</TableCell>
    <TableCell class="text-right">
      <button
        type="button"
        class="min-h-9 rounded-md border border-border bg-background px-3 text-[11px] font-semibold text-foreground transition-colors hover:border-primary/50 hover:bg-secondary"
        :aria-label="`Inspect ${role.name}`"
        @click="inspect"
      >
        Inspect
      </button>
    </TableCell>
  </TableRow>
</template>
