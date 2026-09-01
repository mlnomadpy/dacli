<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { ActivityResponse, Phase, Project } from '@/types'
import type { DashboardSelection } from '@/composables/useDashboardRoute'
import { dashboardHref } from '@/composables/useDashboardRoute'
import Badge from '@/components/ui/badge/Badge.vue'
import ErrorPanel from '@/components/ErrorPanel.vue'
import SkeletonBlock from '@/components/SkeletonBlock.vue'

const props = defineProps<{
  activity: ActivityResponse | null
  phase: Phase
  hasSnapshot: boolean
  error: string | null
  selection: DashboardSelection
  projects: Project[]
}>()
const emit = defineEmits<{
  change: [selection: DashboardSelection]
  retry: []
}>()

const taskInput = ref(props.selection.task ?? '')
const actorInput = ref(props.selection.actor ?? '')
watch(
  () => [props.selection.task, props.selection.actor] as const,
  ([task, actor]) => {
    taskInput.value = task ?? ''
    actorInput.value = actor ?? ''
  },
)

const kinds = [
  ['finding', 'Review findings'],
  ['help', 'Owner asks'],
  ['review', 'Review verdicts'],
  ['block', 'Blocked work'],
  ['propose-status', 'Status proposals'],
  ['dependency', 'Dependency proposals'],
  ['dismissal', 'Reconciliation'],
  ['run', 'Runs'],
  ['commit', 'Commits'],
  ['exit', 'Run exits'],
] as const

const resultLabel = computed(() => {
  const count = props.activity?.events.length ?? 0
  if (!props.activity) return 'No activity observation yet'
  const suffix = props.activity.truncated ? ' on this page; more recorded' : ' observed'
  return `${count} event${count === 1 ? '' : 's'}${suffix}`
})

function update(patch: Partial<DashboardSelection>, clearCursor = true): void {
  const next = { ...props.selection, ...patch }
  if (clearCursor) delete next.cursor
  for (const [key, value] of Object.entries(next)) {
    if (!value || value === 'all') delete next[key as keyof DashboardSelection]
  }
  emit('change', next)
}

function applyIdentityFilters(): void {
  update({ task: taskInput.value.trim() || undefined, actor: actorInput.value.trim() || undefined })
}

function eventTone(category: string): string {
  switch (category) {
    case 'refusal':
      return 'border-l-destructive'
    case 'finding':
    case 'review':
      return 'border-l-warning'
    case 'handoff':
    case 'reconciliation':
      return 'border-l-primary'
    case 'delivery':
      return 'border-l-success'
    default:
      return 'border-l-border'
  }
}

function formatTime(at: string): string {
  if (!at) return 'unknown time'
  const date = new Date(at)
  return Number.isNaN(date.valueOf()) ? 'unknown time' : date.toLocaleString()
}
</script>

<template>
  <section aria-labelledby="activity-feed-h" class="space-y-4">
    <div
      class="rounded-xl border border-border bg-card/90 p-4 shadow-[0_24px_70px_-58px_rgba(121,166,255,0.75)]"
    >
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p
            class="m-0 font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-primary"
          >
            Append-only evidence
          </p>
          <h2 id="activity-feed-h" class="mt-1 mb-0 text-lg font-semibold">
            Activity and refusals
          </h2>
          <p class="mt-1 mb-0 max-w-2xl text-xs leading-relaxed text-muted-foreground">
            Newest first. This feed observes durable records; it cannot sync, dismiss, approve, or
            reconcile them.
          </p>
        </div>
        <p
          class="m-0 rounded border border-border bg-background px-2.5 py-1 font-mono text-[11px] text-muted-foreground"
          aria-live="polite"
        >
          {{ resultLabel }}
        </p>
      </div>

      <form
        class="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-6"
        @submit.prevent="applyIdentityFilters"
      >
        <label
          class="grid gap-1 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground"
        >
          Project
          <select
            :value="selection.project ?? ''"
            class="h-9 rounded-md border border-input bg-background px-2 text-xs text-foreground"
            @change="update({ project: ($event.target as HTMLSelectElement).value || undefined })"
          >
            <option value="">All projects</option>
            <option v-for="project in projects" :key="project.slug" :value="project.slug">
              {{ project.title }}
            </option>
          </select>
        </label>
        <label
          class="grid gap-1 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground"
        >
          Task
          <input
            v-model="taskInput"
            class="h-9 min-w-0 rounded-md border border-input bg-background px-2 text-xs normal-case tracking-normal text-foreground"
            placeholder="Exact task ref"
          />
        </label>
        <label
          class="grid gap-1 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground"
        >
          Event kind
          <select
            :value="selection.kind ?? ''"
            class="h-9 rounded-md border border-input bg-background px-2 text-xs text-foreground"
            @change="update({ kind: ($event.target as HTMLSelectElement).value || undefined })"
          >
            <option value="">All kinds</option>
            <option v-for="kind in kinds" :key="kind[0]" :value="kind[0]">{{ kind[1] }}</option>
          </select>
        </label>
        <label
          class="grid gap-1 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground"
        >
          Actor
          <input
            v-model="actorInput"
            class="h-9 min-w-0 rounded-md border border-input bg-background px-2 text-xs normal-case tracking-normal text-foreground"
            placeholder="Exact agent id"
          />
        </label>
        <label
          class="grid gap-1 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground"
        >
          State
          <select
            :value="selection.event_state ?? 'all'"
            class="h-9 rounded-md border border-input bg-background px-2 text-xs text-foreground"
            @change="
              update({
                event_state: ($event.target as HTMLSelectElement)
                  .value as DashboardSelection['event_state'],
              })
            "
          >
            <option value="all">All states</option>
            <option value="pending">Pending owner action</option>
            <option value="applied">Applied / journaled</option>
          </select>
        </label>
        <label
          class="grid gap-1 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground"
        >
          Range
          <select
            :value="selection.range ?? '7d'"
            class="h-9 rounded-md border border-input bg-background px-2 text-xs text-foreground"
            @change="
              update({
                range: ($event.target as HTMLSelectElement).value as DashboardSelection['range'],
              })
            "
          >
            <option value="24h">Last 24 hours</option>
            <option value="7d">Last 7 days</option>
            <option value="30d">Last 30 days</option>
          </select>
        </label>
        <button
          type="submit"
          class="min-h-9 rounded-md bg-primary px-3 text-xs font-semibold text-primary-foreground sm:col-span-1 xl:col-span-5 xl:justify-self-end"
        >
          Apply task and actor
        </button>
        <button
          type="button"
          class="min-h-9 rounded-md border border-border px-3 text-xs font-semibold text-foreground"
          @click="emit('change', {})"
        >
          Clear filters
        </button>
      </form>
    </div>

    <div
      v-if="phase === 'error' && hasSnapshot"
      role="alert"
      class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-destructive/50 bg-destructive/10 px-4 py-3 text-xs text-foreground"
    >
      <span>Stale activity snapshot — {{ error ?? 'the latest journal observation failed' }}.</span>
      <button
        type="button"
        class="font-semibold text-primary hover:underline"
        @click="emit('retry')"
      >
        Retry observation
      </button>
    </div>

    <div
      v-if="activity?.partial"
      role="alert"
      class="rounded-lg border border-warning/50 bg-warning/10 px-4 py-3 text-xs text-foreground"
    >
      Partial journal observation: {{ activity.unreadable_records }} durable record{{
        activity.unreadable_records === 1 ? '' : 's'
      }}
      could not be read. Visible records remain usable, but this page is not complete.
    </div>

    <div v-if="phase === 'loading' && !hasSnapshot" role="status" class="space-y-2">
      <p class="m-0 text-xs text-muted-foreground">Loading durable activity…</p>
      <SkeletonBlock height="220px" />
    </div>
    <ErrorPanel
      v-else-if="phase === 'error' && !hasSnapshot"
      :message="`couldn't load activity — ${error ?? 'unknown error'}`"
      @retry="emit('retry')"
    />
    <div v-else-if="activity?.events.length" class="relative">
      <div
        aria-hidden="true"
        class="absolute top-4 bottom-4 left-[11px] w-px bg-border sm:left-[19px]"
      />
      <ol class="relative m-0 grid list-none gap-3 p-0" aria-label="Newest-first durable activity">
        <li v-for="event in activity.events" :key="event.id" class="relative pl-7 sm:pl-10">
          <span
            aria-hidden="true"
            class="absolute top-5 left-[7px] size-2.5 rounded-full border-2 border-background bg-primary sm:left-[15px]"
          />
          <article
            class="rounded-lg border border-border border-l-4 bg-card p-3.5"
            :class="eventTone(event.category)"
          >
            <div class="flex flex-wrap items-start justify-between gap-2">
              <div class="flex flex-wrap items-center gap-2">
                <strong class="text-sm text-foreground">{{ event.label }}</strong>
                <Badge variant="outline">{{ event.kind }}</Badge>
                <Badge :variant="event.applied ? 'secondary' : 'outline'">{{
                  event.applied ? 'applied / journaled' : 'pending owner action'
                }}</Badge>
              </div>
              <time
                class="font-mono text-[10px] text-muted-foreground"
                :datetime="event.at || undefined"
                :title="event.at || 'unknown timestamp'"
                >{{ formatTime(event.at) }}</time
              >
            </div>
            <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
              <span
                >Actor
                <a
                  v-if="event.related_agent"
                  class="font-mono text-primary underline-offset-2 hover:underline"
                  :href="dashboardHref('agents', { agent: event.related_agent })"
                  >{{ event.actor }}</a
                ><strong v-else class="font-mono text-foreground">{{
                  event.actor || 'unknown'
                }}</strong></span
              >
              <span v-if="event.related_task"
                >Task
                <a
                  class="font-mono text-primary underline-offset-2 hover:underline"
                  :href="dashboardHref('work', { task: event.related_task })"
                  >{{ event.related_task }}</a
                ></span
              >
              <span v-else-if="event.about"
                >About <strong class="font-mono text-foreground">{{ event.about }}</strong></span
              >
              <span v-if="event.against"
                >Against
                <a
                  class="font-mono text-primary underline-offset-2 hover:underline"
                  :href="dashboardHref('agents', { agent: event.against })"
                  >{{ event.against }}</a
                ></span
              >
              <span
                >Origin
                <strong class="font-mono text-foreground">{{
                  event.origin || 'agent'
                }}</strong></span
              >
            </div>
            <p
              v-if="event.body"
              class="mt-3 mb-0 max-h-40 overflow-auto whitespace-pre-wrap break-words rounded border border-border/70 bg-background/70 p-2.5 font-mono text-[11px] leading-relaxed text-foreground"
            >
              {{ event.body }}
            </p>
          </article>
        </li>
      </ol>
    </div>
    <div
      v-else-if="hasSnapshot"
      class="rounded-lg border border-dashed border-border bg-card/50 px-5 py-10 text-center"
    >
      <p class="m-0 text-sm font-semibold">No events match this observation.</p>
      <p class="mt-1 mb-0 text-xs text-muted-foreground">
        Change the project, identity, kind, state, or time range. No record was marked applied by
        this view.
      </p>
    </div>

    <div
      v-if="activity && (activity.next_cursor || selection.cursor)"
      class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border bg-card px-4 py-3 text-xs"
    >
      <span class="text-muted-foreground"
        >Stable cursor pagination · {{ activity.limit }} records per page</span
      >
      <div class="flex gap-2">
        <button
          v-if="selection.cursor"
          type="button"
          class="rounded border border-border px-3 py-2 font-semibold"
          @click="update({ cursor: undefined }, false)"
        >
          Return to newest
        </button>
        <button
          v-if="activity.next_cursor"
          type="button"
          class="rounded bg-primary px-3 py-2 font-semibold text-primary-foreground"
          @click="update({ cursor: activity.next_cursor }, false)"
        >
          Older events
        </button>
      </div>
    </div>
  </section>
</template>
