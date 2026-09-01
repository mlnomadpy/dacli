<script setup lang="ts">
import { computed } from 'vue'
import { dashboardHref } from '@/composables/useDashboardRoute'
import type { OverviewResponse, Project } from '@/types'

interface AttentionItem {
  label: string
  detail: string
  href: string
  tone: 'danger' | 'warning'
}

const props = defineProps<{
  overview: OverviewResponse
  projects: Project[]
}>()

const focusProject = computed(
  () =>
    props.projects.find((project) => (project.counts.active ?? 0) > 0) ??
    props.projects.find((project) => (project.counts.open ?? 0) > 0) ??
    props.projects[0] ??
    null,
)

const attention = computed<AttentionItem[]>(() => {
  const items: AttentionItem[] = []
  const blocked = props.overview.counts.blocked ?? 0
  if (blocked > 0) {
    items.push({
      label: `${blocked} blocked task${blocked === 1 ? '' : 's'}`,
      detail: 'Open Work to inspect the affected project portfolio.',
      href: dashboardHref('work', { project: focusProject.value?.slug }),
      tone: 'danger',
    })
  }
  if (props.overview.pending_events > 0) {
    items.push({
      label: `${props.overview.pending_events} pending event${props.overview.pending_events === 1 ? '' : 's'}`,
      detail: 'Review the durable event inbox before assuming owner-visible state is current.',
      href: dashboardHref('activity', { event_state: 'pending' }),
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
          Workspace now
        </p>
        <h3 id="operator-pulse-h" class="mt-1 mb-0 text-lg font-semibold tracking-[-0.02em]">
          Operator pulse
        </h3>
      </div>
      <span
        class="rounded-sm border border-border bg-background px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground"
      >
        lightweight overview
      </span>
    </header>

    <div class="grid lg:grid-cols-[1.05fr_1.35fr_0.8fr]">
      <article class="min-w-0 border-b border-border p-5 lg:border-r lg:border-b-0">
        <p class="m-0 text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
          Next work area
        </p>
        <template v-if="focusProject">
          <p class="mt-4 mb-1 font-mono text-xs text-primary">
            {{ focusProject.slug }} / {{ focusProject.stage || 'unphased' }}
          </p>
          <h4 class="m-0 text-xl leading-snug font-semibold tracking-[-0.025em]">
            {{ focusProject.title }}
          </h4>
          <p class="mt-3 mb-0 text-xs leading-relaxed text-muted-foreground">
            {{ focusProject.counts.active ?? 0 }} active · {{ focusProject.counts.open ?? 0 }} open
            · {{ focusProject.counts.blocked ?? 0 }} blocked
          </p>
          <a
            :href="dashboardHref('work', { project: focusProject.slug })"
            class="mt-5 inline-flex min-h-10 items-center rounded-md border border-border bg-background px-3 text-xs font-semibold text-foreground no-underline transition-colors hover:bg-secondary"
          >
            Inspect work →
          </a>
        </template>
        <p v-else class="mt-4 mb-0 text-sm text-muted-foreground">
          No project is recorded in this workspace.
        </p>
      </article>

      <article class="min-w-0 border-b border-border p-5 lg:border-r lg:border-b-0">
        <div class="flex items-center justify-between gap-3">
          <div>
            <p
              class="m-0 text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground"
            >
              Needs attention
            </p>
            <h4 class="mt-1 mb-0 text-base font-semibold">{{ attentionTitle }}</h4>
          </div>
          <span
            v-if="attention.length > 0"
            class="grid size-8 place-items-center rounded-full border border-border bg-background font-mono text-xs text-foreground"
            >{{ attention.length }}</span
          >
        </div>

        <p v-if="attention.length === 0" class="mt-5 mb-0 text-sm text-muted-foreground">
          The lightweight global signals are calm. Open a focused area for its own evidence.
        </p>
        <ul v-else class="mt-4 mb-0 list-none space-y-2 p-0">
          <li v-for="item in attention" :key="item.label">
            <a
              :href="item.href"
              class="group grid grid-cols-[3px_minmax(0,1fr)_auto] items-center gap-3 rounded-md border border-transparent bg-background/70 pr-3 transition-colors hover:border-border hover:bg-secondary"
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
              <span class="font-mono text-xs text-muted-foreground" aria-hidden="true">→</span>
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
            <dd class="mt-0.5 mb-0 text-xl font-semibold tabular-nums">
              {{ overview.live_agents }}
            </dd>
          </div>
          <div class="border-l border-primary pl-3">
            <dt class="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
              Tasks active
            </dt>
            <dd class="mt-0.5 mb-0 text-xl font-semibold tabular-nums">
              {{ overview.counts.active ?? 0 }}
            </dd>
          </div>
          <div class="border-l border-muted-foreground pl-3">
            <dt class="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
              Tasks open
            </dt>
            <dd class="mt-0.5 mb-0 text-xl font-semibold tabular-nums">
              {{ overview.counts.open ?? 0 }}
            </dd>
          </div>
          <div class="border-l border-success pl-3">
            <dt class="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">Projects</dt>
            <dd class="mt-0.5 mb-0 text-xl font-semibold tabular-nums">
              {{ overview.project_count }}
            </dd>
          </div>
        </dl>
      </article>
    </div>
  </section>
</template>
