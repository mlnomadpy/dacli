<script setup lang="ts">
import { computed } from 'vue'
import type { Agent, Burn, GraphNode, Project, Role } from '@/types'

interface AttentionItem {
  label: string
  detail: string
  href: string
  tone: 'danger' | 'warning'
}

const props = defineProps<{
  projects: Project[]
  agents: Agent[]
  roles: Role[]
  burn: Burn
  pendingEvents: number
}>()

const unhealthyStates = new Set<Agent['state']>(['blocked', 'stalled', 'silent'])

const openTasks = computed(() =>
  props.projects.reduce((total, project) => total + (project.counts.open ?? 0), 0),
)
const activeTasks = computed(() =>
  props.projects.reduce((total, project) => total + (project.counts.active ?? 0), 0),
)
const blockedTasks = computed(() =>
  props.projects.reduce((total, project) => total + (project.counts.blocked ?? 0), 0),
)
const unhealthyAgents = computed(() =>
  props.agents.filter((agent) => unhealthyStates.has(agent.state)),
)
const cappedRoles = computed(() => props.roles.filter((role) => role.wip_exceeded))

const criticalFocus = computed<{ node: GraphNode; project: Project } | null>(() => {
  for (const project of props.projects) {
    for (const id of project.graph.critical_path) {
      const node = project.graph.nodes.find((candidate) => candidate.id === id)
      if (node && node.status !== 'done') return { node, project }
    }
  }
  return null
})

const criticalChain = computed(() => {
  if (!criticalFocus.value) return []
  const { project } = criticalFocus.value
  return project.graph.critical_path
    .map((id) => project.graph.nodes.find((node) => node.id === id))
    .filter((node): node is GraphNode => Boolean(node))
})

const graphNote = computed(
  () =>
    props.projects.find((project) => !project.graph.scheduled && project.graph.note)?.graph.note ??
    '',
)

const attention = computed<AttentionItem[]>(() => {
  const items: AttentionItem[] = []
  if (props.burn.alert) {
    items.push({
      label: `Burn is ${props.burn.ratio.toFixed(1)}× the calibrated ceiling`,
      detail: 'Review the latest run intensity before funding another wave.',
      href: '#agents',
      tone: 'danger',
    })
  }
  if (blockedTasks.value > 0) {
    items.push({
      label: `${blockedTasks.value} blocked task${blockedTasks.value === 1 ? '' : 's'}`,
      detail: 'Resolve recorded blockers before widening delivery.',
      href: '#delivery',
      tone: 'danger',
    })
  }
  if (unhealthyAgents.value.length > 0) {
    items.push({
      label: `${unhealthyAgents.value.length} unhealthy agent${unhealthyAgents.value.length === 1 ? '' : 's'}`,
      detail: 'Inspect stalled, silent, or blocked run evidence.',
      href: '#agents',
      tone: 'warning',
    })
  }
  if (props.pendingEvents > 0) {
    items.push({
      label: `${props.pendingEvents} pending event${props.pendingEvents === 1 ? '' : 's'}`,
      detail: 'Reconcile the append-only journal into owner-visible state.',
      href: '#pulse',
      tone: 'warning',
    })
  }
  if (cappedRoles.value.length > 0) {
    items.push({
      label: `${cappedRoles.value.length} role${cappedRoles.value.length === 1 ? '' : 's'} at ${cappedRoles.value.length === 1 ? 'its' : 'their'} WIP cap`,
      detail: 'Wait for capacity or change policy with explicit authority.',
      href: '#team',
      tone: 'warning',
    })
  }
  return items
})

const attentionTitle = computed(() => {
  const count = attention.value.length
  return count === 0
    ? 'No recorded blockers'
    : `${count} signal${count === 1 ? '' : 's'} need attention`
})

function toneClass(tone: AttentionItem['tone']): string {
  return tone === 'danger' ? 'bg-destructive' : 'bg-warning'
}
</script>

<template>
  <section
    aria-labelledby="operator-pulse-h"
    class="operator-pulse overflow-hidden rounded-xl border border-border bg-card/90 shadow-[0_28px_90px_-56px_rgba(105,210,231,0.55)]"
  >
    <header
      class="flex flex-wrap items-center justify-between gap-3 border-b border-border px-5 py-3.5"
    >
      <div>
        <p class="m-0 font-mono text-[10px] font-medium uppercase tracking-[0.16em] text-primary">
          Decision surface
        </p>
        <h2 id="operator-pulse-h" class="mt-1 mb-0 text-lg font-semibold tracking-[-0.02em]">
          Operator pulse
        </h2>
      </div>
      <span
        class="rounded-sm border border-border bg-background px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground"
      >
        read-only projection
      </span>
    </header>

    <div class="grid lg:grid-cols-[1.05fr_1.35fr_0.8fr]">
      <article class="min-w-0 border-b border-border p-5 lg:border-r lg:border-b-0">
        <p class="m-0 text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
          Next on the critical path
        </p>
        <template v-if="criticalFocus">
          <p class="mt-4 mb-1 font-mono text-xs text-primary">
            {{ criticalFocus.project.slug }} / zero slack
          </p>
          <h3 class="m-0 text-xl leading-snug font-semibold tracking-[-0.025em]">
            #{{ criticalFocus.node.seq }} · {{ criticalFocus.node.title }}
          </h3>
          <p class="mt-3 mb-0 text-xs leading-relaxed text-muted-foreground">
            Te {{ criticalFocus.node.points.toFixed(1) }} ·
            {{ criticalFocus.project.graph.duration.toFixed(1) }} Te project duration
          </p>
          <div v-if="criticalChain.length > 1" class="mt-6 border-t border-border pt-4">
            <p
              class="m-0 text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground"
            >
              Recorded path
            </p>
            <ol class="mt-2 mb-0 flex list-none flex-wrap items-center gap-1.5 p-0">
              <li
                v-for="(node, index) in criticalChain"
                :key="node.id"
                class="flex items-center gap-1.5 font-mono text-[11px]"
              >
                <span :class="node.status === 'done' ? 'text-muted-foreground' : 'text-foreground'"
                  >#{{ node.seq }}</span
                >
                <span
                  v-if="index < criticalChain.length - 1"
                  class="text-primary"
                  aria-hidden="true"
                  >→</span
                >
              </li>
            </ol>
          </div>
        </template>
        <template v-else>
          <h3 class="mt-4 mb-1 text-base font-semibold">Critical path unavailable</h3>
          <p class="m-0 text-xs leading-relaxed text-muted-foreground">
            {{ graphNote || 'No open scheduled task is currently on a recorded critical path.' }}
          </p>
        </template>
      </article>

      <article class="min-w-0 border-b border-border p-5 lg:border-r lg:border-b-0">
        <div class="flex items-center justify-between gap-3">
          <div>
            <p
              class="m-0 text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground"
            >
              Needs attention
            </p>
            <h3 class="mt-1 mb-0 text-base font-semibold">{{ attentionTitle }}</h3>
          </div>
          <span
            v-if="attention.length > 0"
            class="grid size-8 place-items-center rounded-full border border-border bg-background font-mono text-xs text-foreground"
            >{{ attention.length }}</span
          >
        </div>

        <p v-if="attention.length === 0" class="mt-5 mb-0 text-sm text-muted-foreground">
          Observed signals are within policy. Continue from the recorded critical path.
        </p>
        <ul v-else class="mt-4 mb-0 list-none space-y-2 p-0">
          <li v-for="item in attention" :key="item.label">
            <a
              :href="item.href"
              class="group grid grid-cols-[3px_minmax(0,1fr)_auto] items-center gap-3 rounded-md border border-transparent bg-background/70 pr-3 transition-colors hover:border-border hover:bg-secondary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
            >
              <span
                class="h-full min-h-12 rounded-l-md"
                :class="toneClass(item.tone)"
                aria-hidden="true"
              />
              <span class="min-w-0 py-2">
                <strong class="block text-xs font-semibold text-foreground">{{
                  item.label
                }}</strong>
                <small class="mt-0.5 block text-[11px] leading-snug text-muted-foreground">{{
                  item.detail
                }}</small>
              </span>
              <span
                class="font-mono text-xs text-muted-foreground transition-transform group-hover:translate-x-0.5"
                aria-hidden="true"
                >→</span
              >
            </a>
          </li>
        </ul>
      </article>

      <article class="min-w-0 p-5">
        <p class="m-0 text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
          System pulse
        </p>
        <dl class="mt-4 mb-0 grid grid-cols-2 gap-x-5 gap-y-4 lg:grid-cols-1">
          <div class="border-l border-primary pl-3">
            <dt class="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
              Agents running
            </dt>
            <dd class="mt-0.5 mb-0 text-xl font-semibold tabular-nums">{{ agents.length }}</dd>
          </div>
          <div class="border-l border-primary pl-3">
            <dt class="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
              Tasks active
            </dt>
            <dd class="mt-0.5 mb-0 text-xl font-semibold tabular-nums">{{ activeTasks }}</dd>
          </div>
          <div class="border-l border-muted-foreground pl-3">
            <dt class="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
              Tasks open
            </dt>
            <dd class="mt-0.5 mb-0 text-xl font-semibold tabular-nums">{{ openTasks }}</dd>
          </div>
          <div class="border-l border-success pl-3">
            <dt class="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">Projects</dt>
            <dd class="mt-0.5 mb-0 text-xl font-semibold tabular-nums">{{ projects.length }}</dd>
          </div>
        </dl>
      </article>
    </div>
  </section>
</template>
