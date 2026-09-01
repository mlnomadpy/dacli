<script setup lang="ts">
import { computed } from 'vue'
import {
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
} from 'reka-ui'
import type { Agent, Phase, Role } from '@/types'
import SkeletonBlock from '@/components/SkeletonBlock.vue'

const props = defineProps<{
  open: boolean
  selectedName: string
  role: Role | null
  rolesPhase: Phase
  rolesHasSnapshot: boolean
  agents: Agent[]
  agentsPhase: Phase
  agentsHasSnapshot: boolean
  agentsError: string | null
}>()
const emit = defineEmits<{ close: [] }>()

const assignedAgents = computed(() =>
  props.agents.filter((agent) => agent.role === props.selectedName),
)

function listOrNone(values: string[], empty = 'none recorded'): string[] {
  return values.length > 0 ? values : [empty]
}

function onOpenChange(open: boolean): void {
  if (!open) emit('close')
}
</script>

<template>
  <DialogRoot :open="open" @update:open="onOpenChange">
    <DialogPortal>
      <DialogOverlay
        class="fixed inset-0 z-40 bg-background/76 backdrop-blur-[2px] data-[state=closed]:opacity-0 data-[state=open]:opacity-100"
      />
      <DialogContent
        class="role-inspector fixed inset-y-0 right-0 z-50 flex w-full max-w-[580px] flex-col border-l border-border bg-card shadow-[-28px_0_90px_-48px_#000] focus:outline-none data-[state=closed]:translate-x-full data-[state=open]:translate-x-0"
      >
        <header
          class="flex items-start justify-between gap-5 border-b border-border px-5 py-5 sm:px-7"
        >
          <div class="min-w-0">
            <p
              class="m-0 font-mono text-[10px] font-semibold uppercase tracking-[0.16em] text-primary"
            >
              Team authority record
            </p>
            <DialogTitle class="mt-2 text-2xl leading-tight font-semibold tracking-[-0.035em]">
              {{ role?.name ?? selectedName }}
            </DialogTitle>
            <DialogDescription class="mt-2 text-sm leading-relaxed text-muted-foreground">
              {{
                role?.summary ||
                'This exact role is no longer present in the latest observed roster.'
              }}
            </DialogDescription>
          </div>
          <DialogClose
            aria-label="Close role details"
            class="grid size-11 shrink-0 place-items-center rounded-md border border-border bg-background text-lg text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            <span aria-hidden="true">×</span>
          </DialogClose>
        </header>

        <div class="min-h-0 flex-1 overflow-y-auto px-5 py-5 sm:px-7">
          <SkeletonBlock
            v-if="rolesPhase === 'loading' && !rolesHasSnapshot"
            height="240px"
            aria-label="Loading role authority"
          />

          <section
            v-else-if="!role"
            role="status"
            class="rounded-lg border border-warning/40 bg-background p-5"
          >
            <h3 class="m-0 text-base font-semibold">Role no longer observed</h3>
            <p class="mt-2 mb-0 text-sm leading-relaxed text-muted-foreground">
              <span class="font-mono text-foreground">{{ selectedName }}</span> was not replaced by
              another role. Close this sheet and choose a role from the current roster.
            </p>
          </section>

          <template v-else>
            <section aria-labelledby="role-capacity-h">
              <p
                class="m-0 font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-primary"
              >
                Authority and capacity
              </p>
              <h3 id="role-capacity-h" class="sr-only">Authority and capacity</h3>
              <dl
                class="mt-3 grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-border bg-border sm:grid-cols-3"
              >
                <div class="bg-background p-3">
                  <dt class="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
                    Kind
                  </dt>
                  <dd class="mt-1 mb-0 text-sm font-semibold">{{ role.kind || 'any phase' }}</dd>
                </div>
                <div class="bg-background p-3">
                  <dt class="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
                    Grant
                  </dt>
                  <dd class="mt-1 mb-0 font-mono text-sm font-semibold">
                    {{ role.grant || 'ro' }}
                  </dd>
                </div>
                <div class="bg-background p-3">
                  <dt class="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">WIP</dt>
                  <dd
                    class="mt-1 mb-0 text-sm font-semibold"
                    :class="role.wip_exceeded ? 'text-destructive' : ''"
                  >
                    {{ role.active_agents }} / {{ role.wip > 0 ? role.wip : 'uncapped' }}
                  </dd>
                </div>
                <div class="bg-background p-3">
                  <dt class="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
                    Runtime
                  </dt>
                  <dd class="mt-1 mb-0 font-mono text-xs font-semibold">
                    {{ role.runtime || 'inherited' }}
                  </dd>
                </div>
                <div class="bg-background p-3">
                  <dt class="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
                    Model
                  </dt>
                  <dd class="mt-1 mb-0 font-mono text-xs font-semibold">
                    {{ role.model || 'inherited' }}
                  </dd>
                </div>
                <div class="bg-background p-3">
                  <dt class="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
                    Max task
                  </dt>
                  <dd class="mt-1 mb-0 text-sm font-semibold">
                    {{ role.max_points > 0 ? `Te ${role.max_points}` : 'uncapped' }}
                  </dd>
                </div>
              </dl>
              <p
                class="mt-3 mb-0 rounded-md border border-border bg-background px-3 py-2 text-xs text-muted-foreground"
              >
                Standing instructions:
                <strong class="text-foreground">{{
                  role.has_prompt ? 'defined' : 'metadata only'
                }}</strong>
              </p>
            </section>

            <section aria-labelledby="role-occupancy-h" class="mt-7 border-t border-border pt-6">
              <div class="flex items-center justify-between gap-4">
                <div>
                  <p
                    class="m-0 font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-primary"
                  >
                    Current occupancy
                  </p>
                  <h3 id="role-occupancy-h" class="mt-1 mb-0 text-base font-semibold">
                    {{ assignedAgents.length }} live member{{
                      assignedAgents.length === 1 ? '' : 's'
                    }}
                  </h3>
                </div>
                <span
                  class="rounded-full border border-border bg-background px-2.5 py-1 font-mono text-xs"
                  >{{ assignedAgents.length }}</span
                >
              </div>
              <p
                v-if="agentsPhase === 'error' && !agentsHasSnapshot"
                role="alert"
                class="mt-3 mb-0 text-xs text-destructive"
              >
                Live-member evidence is unavailable: {{ agentsError ?? 'unknown error' }}
              </p>
              <p
                v-else-if="assignedAgents.length === 0"
                class="mt-3 mb-0 text-sm text-muted-foreground"
              >
                No live agents are assigned to this role in the current observation.
              </p>
              <ul v-else class="mt-3 mb-0 list-none space-y-2 p-0">
                <li
                  v-for="agent in assignedAgents"
                  :key="agent.run_id"
                  class="grid grid-cols-[minmax(0,1fr)_auto] gap-3 rounded-md border border-border bg-background px-3 py-2.5"
                >
                  <span class="min-w-0">
                    <strong class="block truncate font-mono text-xs">{{ agent.child }}</strong>
                    <small class="mt-0.5 block text-[11px] text-muted-foreground"
                      >task {{ agent.task }} · {{ agent.runtime }}</small
                    >
                  </span>
                  <span
                    class="self-center rounded-full border border-border px-2 py-0.5 text-[10px] uppercase tracking-[0.06em] text-muted-foreground"
                    >{{ agent.state }}</span
                  >
                </li>
              </ul>
            </section>

            <section aria-labelledby="role-boundaries-h" class="mt-7 border-t border-border pt-6">
              <p
                class="m-0 font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-primary"
              >
                Boundaries and knowledge
              </p>
              <h3 id="role-boundaries-h" class="sr-only">Boundaries and knowledge</h3>
              <div class="mt-3 grid gap-5 sm:grid-cols-2">
                <div>
                  <h4 class="m-0 text-xs font-semibold uppercase tracking-[0.06em]">In scope</h4>
                  <ul
                    class="mt-2 mb-0 list-none space-y-1 p-0 font-mono text-xs text-muted-foreground"
                  >
                    <li
                      v-for="item in listOrNone(role.scope, 'everywhere')"
                      :key="item"
                      class="break-all"
                    >
                      {{ item }}
                    </li>
                  </ul>
                </div>
                <div>
                  <h4 class="m-0 text-xs font-semibold uppercase tracking-[0.06em]">
                    Out of scope
                  </h4>
                  <ul
                    class="mt-2 mb-0 list-none space-y-1 p-0 font-mono text-xs text-muted-foreground"
                  >
                    <li v-for="item in listOrNone(role.out_of_scope)" :key="item" class="break-all">
                      {{ item }}
                    </li>
                  </ul>
                </div>
                <div>
                  <h4 class="m-0 text-xs font-semibold uppercase tracking-[0.06em]">Skills</h4>
                  <ul class="mt-2 mb-0 list-none space-y-1 p-0 text-xs text-muted-foreground">
                    <li v-for="item in listOrNone(role.skills)" :key="item">{{ item }}</li>
                  </ul>
                </div>
                <div>
                  <h4 class="m-0 text-xs font-semibold uppercase tracking-[0.06em]">Shortcuts</h4>
                  <ul
                    class="mt-2 mb-0 list-none space-y-1 p-0 font-mono text-xs text-muted-foreground"
                  >
                    <li v-for="item in listOrNone(role.shortcuts)" :key="item">{{ item }}</li>
                  </ul>
                </div>
              </div>
              <div class="mt-5 rounded-md border border-border bg-background p-3">
                <h4 class="m-0 text-xs font-semibold uppercase tracking-[0.06em]">Escalates to</h4>
                <p class="mt-2 mb-0 text-xs text-muted-foreground">
                  {{ listOrNone(role.escalate_to).join(', ') }}
                </p>
              </div>
            </section>
          </template>
        </div>

        <footer
          class="border-t border-border px-5 py-3 font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground sm:px-7"
        >
          projection only · no edit or spawn authority
        </footer>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style scoped>
.role-inspector,
[data-reka-dialog-overlay] {
  transition:
    transform 180ms cubic-bezier(0.22, 0.61, 0.36, 1),
    opacity 150ms ease;
}

@media (prefers-reduced-motion: reduce) {
  .role-inspector,
  [data-reka-dialog-overlay] {
    transition: none;
  }
}
</style>
