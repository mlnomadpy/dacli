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
import type { AgentDetail, Phase } from '@/types'
import { dashboardHref } from '@/composables/useDashboardRoute'
import SkeletonBlock from '@/components/SkeletonBlock.vue'

const props = defineProps<{
  open: boolean
  selectedID: string
  agent: AgentDetail | null
  phase: Phase
  hasSnapshot: boolean
  error: string | null
  status: number | null
  live: boolean
}>()
const emit = defineEmits<{
  close: []
  retry: []
  navigateAgent: [agent: string]
}>()

const availability = computed(() => {
  if (props.agent?.retired) return 'retired'
  if (props.live) return 'live now'
  return 'no longer live'
})

const failureTitle = computed(() => {
  if (props.status === 400) return 'Agent identity rejected'
  if (props.status === 404) return 'Agent record unavailable'
  return 'Agent detail unavailable'
})

function onOpenChange(open: boolean): void {
  if (!open) emit('close')
}

function routableTaskRef(ref: string, sequence?: number): string {
  if (sequence) return String(sequence)
  const parts = ref.split('/')
  return parts[parts.length - 1] ?? ''
}
</script>

<template>
  <DialogRoot :open="open" @update:open="onOpenChange">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-40 bg-background/76 backdrop-blur-[2px]" />
      <DialogContent
        class="agent-inspector fixed inset-y-0 right-0 z-50 flex w-full max-w-[640px] flex-col border-l border-border bg-card shadow-[-28px_0_90px_-48px_#000] focus:outline-none"
      >
        <header
          class="flex items-start justify-between gap-5 border-b border-border px-5 py-5 sm:px-7"
        >
          <div class="min-w-0">
            <p
              class="m-0 font-mono text-[10px] font-semibold uppercase tracking-[0.16em] text-primary"
            >
              Agent lineage record
            </p>
            <div class="mt-2 flex flex-wrap items-center gap-2">
              <DialogTitle
                class="break-all font-mono text-xl leading-tight font-semibold tracking-[-0.025em]"
              >
                {{ agent?.id ?? selectedID }}
              </DialogTitle>
              <span
                class="rounded-full border border-border px-2 py-0.5 text-[10px] font-semibold uppercase"
                :class="live ? 'text-success' : 'text-warning'"
                >{{ availability }}</span
              >
            </div>
            <DialogDescription class="mt-2 text-sm leading-relaxed text-muted-foreground">
              Durable identity, authority, owned work, and newest-first run evidence.
            </DialogDescription>
          </div>
          <DialogClose
            aria-label="Close agent details"
            class="grid size-11 shrink-0 place-items-center rounded-md border border-border bg-background text-lg text-muted-foreground hover:bg-secondary hover:text-foreground"
          >
            <span aria-hidden="true">×</span>
          </DialogClose>
        </header>

        <div class="min-h-0 flex-1 overflow-y-auto px-5 py-5 sm:px-7">
          <SkeletonBlock
            v-if="phase === 'loading' && !hasSnapshot"
            height="300px"
            aria-label="Loading agent detail"
          />

          <section
            v-else-if="phase === 'error' && !hasSnapshot"
            role="alert"
            class="rounded-lg border border-destructive/40 bg-background p-5"
          >
            <h3 class="m-0 text-base font-semibold">{{ failureTitle }}</h3>
            <p class="mt-2 text-sm text-muted-foreground">
              <span class="font-mono text-foreground">{{ selectedID }}</span> was not replaced with
              another identity. {{ error }}
            </p>
            <button
              type="button"
              class="mt-4 min-h-11 rounded-md border border-border px-4 text-xs font-semibold text-primary"
              @click="emit('retry')"
            >
              Retry this agent
            </button>
          </section>

          <template v-else-if="agent">
            <div
              v-if="phase === 'error'"
              role="alert"
              class="mb-5 rounded-md border border-warning/40 bg-warning/5 px-4 py-3 text-xs text-warning"
            >
              Detail refresh failed; retained evidence is stale. {{ error }}
              <button type="button" class="ml-2 underline" @click="emit('retry')">Retry</button>
            </div>

            <section aria-labelledby="agent-authority-h">
              <p
                class="m-0 font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-primary"
              >
                Authority
              </p>
              <h3 id="agent-authority-h" class="sr-only">Agent authority</h3>
              <dl
                class="mt-3 grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-border bg-border sm:grid-cols-3"
              >
                <div class="bg-background p-3">
                  <dt>Role</dt>
                  <dd>
                    <a :href="dashboardHref('team', { role: agent.role })">{{
                      agent.role || 'unassigned'
                    }}</a>
                  </dd>
                </div>
                <div class="bg-background p-3">
                  <dt>Grant</dt>
                  <dd class="font-mono">{{ agent.grant || 'unknown' }}</dd>
                </div>
                <div class="bg-background p-3">
                  <dt>State</dt>
                  <dd>{{ availability }}</dd>
                </div>
              </dl>
            </section>

            <section aria-labelledby="agent-lineage-h" class="mt-7 border-t border-border pt-6">
              <p
                class="m-0 font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-primary"
              >
                Lineage
              </p>
              <h3 id="agent-lineage-h" class="mt-1 text-base font-semibold">Parent and children</h3>
              <div class="mt-3 grid gap-3 sm:grid-cols-2">
                <div class="rounded-md border border-border bg-background p-3">
                  <p class="agent-label">Parent</p>
                  <button
                    v-if="agent.parent"
                    type="button"
                    class="agent-link"
                    @click="emit('navigateAgent', agent.parent)"
                  >
                    {{ agent.parent }}
                  </button>
                  <p v-else class="mt-2 text-xs text-muted-foreground">root identity</p>
                </div>
                <div class="rounded-md border border-border bg-background p-3">
                  <p class="agent-label">Children · {{ agent.children.length }}</p>
                  <div v-if="agent.children.length" class="mt-2 flex flex-wrap gap-2">
                    <button
                      v-for="child in agent.children"
                      :key="child"
                      type="button"
                      class="agent-link"
                      @click="emit('navigateAgent', child)"
                    >
                      {{ child }}
                    </button>
                  </div>
                  <p v-else class="mt-2 text-xs text-muted-foreground">none recorded</p>
                </div>
              </div>
            </section>

            <section aria-labelledby="agent-tasks-h" class="mt-7 border-t border-border pt-6">
              <p
                class="m-0 font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-primary"
              >
                Owned work
              </p>
              <h3 id="agent-tasks-h" class="mt-1 text-base font-semibold">
                {{ agent.tasks.length }} current task{{ agent.tasks.length === 1 ? '' : 's' }}
              </h3>
              <p v-if="agent.tasks.length === 0" class="mt-3 text-sm text-muted-foreground">
                No task currently names this agent as owner.
              </p>
              <ul v-else class="mt-3 space-y-2">
                <li
                  v-for="task in agent.tasks"
                  :key="task.id"
                  class="rounded-md border border-border bg-background p-3"
                >
                  <a
                    :href="
                      dashboardHref('work', {
                        project: task.project,
                        task: routableTaskRef(task.id, task.seq),
                      })
                    "
                    class="text-sm font-semibold text-primary hover:underline"
                    >{{ String(task.seq).padStart(3, '0') }} · {{ task.title }}</a
                  >
                  <p class="mt-1 text-[11px] text-muted-foreground">
                    {{ task.status }} · {{ task.priority || 'normal priority' }} ·
                    {{ task.estimated ? `Te ${task.points}` : 'unestimated' }}
                  </p>
                </li>
              </ul>
            </section>

            <section aria-labelledby="agent-runs-h" class="mt-7 border-t border-border pt-6">
              <p
                class="m-0 font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-primary"
              >
                Run ledger
              </p>
              <h3 id="agent-runs-h" class="mt-1 text-base font-semibold">
                {{ agent.runs.length }} historical run{{ agent.runs.length === 1 ? '' : 's' }}
              </h3>
              <p v-if="agent.runs.length === 0" class="mt-3 text-sm text-muted-foreground">
                No durable run record is attributable to this agent.
              </p>
              <ol v-else class="mt-3 space-y-2">
                <li
                  v-for="run in agent.runs"
                  :key="run.run_id"
                  class="rounded-md border border-border bg-background p-3"
                >
                  <div class="flex flex-wrap items-start justify-between gap-2">
                    <a
                      :href="run.transcript_url"
                      target="_blank"
                      rel="noopener"
                      class="break-all font-mono text-xs font-semibold text-primary hover:underline"
                      >{{ run.run_id }}</a
                    >
                    <span
                      class="rounded-full border border-border px-2 py-0.5 text-[10px] font-semibold uppercase"
                      :class="run.live ? 'text-success' : 'text-muted-foreground'"
                      >{{ run.live ? 'live' : 'dead' }}</span
                    >
                  </div>
                  <p class="mt-2 text-xs text-muted-foreground">
                    {{ run.role || 'unassigned role' }} · {{ run.runtime || 'unknown runtime' }} ·
                    PID {{ run.pid || '—' }}
                  </p>
                  <p class="mt-1 font-mono text-[10px] text-muted-foreground">
                    started {{ run.started || 'unknown' }} · task {{ run.task || '—' }}
                  </p>
                  <div class="mt-3 flex gap-3 text-xs">
                    <a :href="run.transcript_url" target="_blank" rel="noopener">Transcript</a
                    ><a :href="run.diff_url" target="_blank" rel="noopener">Diff</a
                    ><a
                      v-if="run.task"
                      :href="dashboardHref('work', { task: routableTaskRef(run.task) })"
                      >Task</a
                    >
                  </div>
                </li>
              </ol>
            </section>
          </template>
        </div>

        <footer
          class="border-t border-border px-5 py-3 font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground sm:px-7"
        >
          projection only · no kill, retry, grant, or ownership authority
        </footer>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style scoped>
dt,
.agent-label {
  margin: 0;
  color: hsl(var(--muted-foreground));
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.6rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}
dd {
  margin: 0.3rem 0 0;
  font-size: 0.8rem;
  font-weight: 650;
}
.agent-link {
  min-height: 2rem;
  color: hsl(var(--primary));
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.72rem;
  text-align: left;
}
a {
  color: hsl(var(--primary));
}
@media (max-width: 390px) {
  .agent-inspector {
    max-width: 100%;
  }
}
</style>
