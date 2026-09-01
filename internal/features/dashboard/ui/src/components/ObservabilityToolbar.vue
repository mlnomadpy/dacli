<script setup lang="ts">
import { computed } from 'vue'
import type { Agent, Project, Role } from '@/types'
import type { DashboardRouteName } from '@/stores/dashboard'
import { DASHBOARD_TIME_RANGES, type DashboardSelection } from '@/composables/useDashboardRoute'
import {
  inactiveFilters,
  supportsFilter,
  type DashboardFilterKey,
} from '@/composables/useObservabilityFilters'
import { ago } from '@/composables/useRelativeTime'

const props = defineProps<{
  route: DashboardRouteName
  selection: DashboardSelection
  projects: Project[]
  roles: Role[]
  agents: Agent[]
  generated: string | null
  resultLabel: string
}>()

const emit = defineEmits<{ change: [selection: DashboardSelection] }>()

const runtimeOptions = computed(() =>
  [
    ...new Set([
      ...props.roles.map((role) => role.runtime),
      ...props.agents.map((a) => a.runtime),
      props.selection.runtime ?? '',
    ]),
  ]
    .filter(Boolean)
    .sort(),
)
const modelOptions = computed(() =>
  [...new Set([...props.roles.map((role) => role.model), props.selection.model ?? ''])]
    .filter(Boolean)
    .sort(),
)
const roleOptions = computed(() =>
  [
    ...new Set([
      ...props.roles.map((role) => role.name),
      ...props.agents.map((a) => a.role),
      props.selection.filter_role ?? '',
    ]),
  ]
    .filter(Boolean)
    .sort(),
)
const stateOptions = computed(() =>
  [...new Set([...props.agents.map((agent) => agent.state), props.selection.state ?? ''])]
    .filter(Boolean)
    .sort(),
)
const unavailable = computed(() => inactiveFilters(props.route, props.selection))
const activeCount = computed(
  () =>
    Object.entries(props.selection).filter(
      ([key, value]) => !['task', 'agent', 'role', 'live'].includes(key) && Boolean(value),
    ).length,
)
const isPaused = computed(() => props.selection.live === 'paused')

function supports(key: DashboardFilterKey): boolean {
  return supportsFilter(props.route, key)
}

function update(key: keyof DashboardSelection, value: string): void {
  const next = { ...props.selection }
  if (value) Object.assign(next, { [key]: value })
  else delete next[key]
  emit('change', next)
}

function clearFilters(): void {
  const next: DashboardSelection = {}
  for (const key of ['task', 'agent', 'role'] as const) {
    if (props.selection[key]) next[key] = props.selection[key]
  }
  if (props.selection.live) next.live = props.selection.live
  emit('change', next)
}

function toggleLive(): void {
  update('live', isPaused.value ? '' : 'paused')
}
</script>

<template>
  <section class="observation-strip" aria-labelledby="observation-scope-h">
    <div class="observation-summary">
      <div>
        <p id="observation-scope-h">Observation scope</p>
        <span aria-live="polite">{{ resultLabel }}</span>
      </div>
      <div class="observation-clock" :class="{ paused: isPaused }">
        <span class="signal" aria-hidden="true" />
        <span>
          {{ isPaused ? 'paused snapshot' : 'live observations' }}
          <small :title="generated ?? undefined">{{ ago(generated) }}</small>
        </span>
        <button type="button" :aria-pressed="isPaused" @click="toggleLive">
          {{ isPaused ? 'Resume' : 'Pause' }}
        </button>
      </div>
    </div>

    <div class="observation-controls">
      <label>
        <span>Project</span>
        <select
          :value="selection.project ?? ''"
          :disabled="!supports('project')"
          @change="update('project', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">All projects</option>
          <option
            v-if="
              selection.project && !projects.some((project) => project.slug === selection.project)
            "
            :value="selection.project"
          >
            {{ selection.project }} (not observed)
          </option>
          <option v-for="project in projects" :key="project.slug" :value="project.slug">
            {{ project.title || project.slug }}
          </option>
        </select>
      </label>

      <label class="search-control">
        <span>Search</span>
        <input
          type="search"
          :value="selection.q ?? ''"
          :disabled="!supports('q')"
          maxlength="128"
          placeholder="Run, task, role…"
          @change="update('q', ($event.target as HTMLInputElement).value.trim())"
        />
      </label>

      <label>
        <span>Window</span>
        <select
          :value="selection.range ?? ''"
          :disabled="!supports('range')"
          @change="update('range', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">Current snapshot</option>
          <option v-for="range in DASHBOARD_TIME_RANGES" :key="range" :value="range">
            Last {{ range }}
          </option>
        </select>
      </label>

      <label>
        <span>Role</span>
        <select
          :value="selection.filter_role ?? ''"
          :disabled="!supports('filter_role')"
          @change="update('filter_role', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">Any role</option>
          <option v-for="role in roleOptions" :key="role" :value="role">{{ role }}</option>
        </select>
      </label>

      <label>
        <span>Runtime</span>
        <select
          :value="selection.runtime ?? ''"
          :disabled="!supports('runtime')"
          @change="update('runtime', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">Any runtime</option>
          <option v-for="runtime in runtimeOptions" :key="runtime" :value="runtime">
            {{ runtime }}
          </option>
        </select>
      </label>

      <label>
        <span>Model</span>
        <select
          :value="selection.model ?? ''"
          :disabled="!supports('model')"
          @change="update('model', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">Any model</option>
          <option v-for="model in modelOptions" :key="model" :value="model">{{ model }}</option>
        </select>
      </label>

      <label>
        <span>State</span>
        <select
          :value="selection.state ?? ''"
          :disabled="!supports('state')"
          @change="update('state', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">Any state</option>
          <option v-for="state in stateOptions" :key="state" :value="state">{{ state }}</option>
        </select>
      </label>

      <button
        class="clear-filters"
        type="button"
        :disabled="activeCount === 0"
        @click="clearFilters"
      >
        Clear {{ activeCount || '' }}
      </button>
    </div>

    <p v-if="unavailable.length" class="inactive-note" role="status">
      Preserved for another route, not applied here: {{ unavailable.join(', ') }}.
    </p>
  </section>
</template>

<style scoped>
.observation-strip {
  margin-top: 0.75rem;
  border: 1px solid var(--border);
  border-radius: 0.55rem;
  background: color-mix(in srgb, var(--card) 86%, transparent);
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--primary) 72%, transparent);
}
.observation-summary {
  display: flex;
  min-height: 42px;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.55rem 0.75rem;
  border-bottom: 1px solid var(--border);
}
.observation-summary p,
.observation-controls label > span {
  margin: 0;
  color: var(--primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.58rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
.observation-summary > div:first-child > span {
  display: block;
  margin-top: 0.12rem;
  color: var(--muted-foreground);
  font-size: 0.72rem;
}
.observation-clock {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  color: var(--foreground);
  font-size: 0.7rem;
}
.observation-clock small {
  display: block;
  color: var(--muted-foreground);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.58rem;
}
.signal {
  width: 0.45rem;
  height: 0.45rem;
  border-radius: 999px;
  background: var(--success);
  animation: pulse 2s infinite;
}
.observation-clock.paused .signal {
  background: var(--warning);
  animation: none;
}
.observation-clock button,
.clear-filters {
  min-height: 30px;
  border: 1px solid var(--border);
  border-radius: 0.3rem;
  background: var(--background);
  padding-inline: 0.65rem;
  color: var(--foreground);
  font-size: 0.67rem;
  font-weight: 650;
  cursor: pointer;
}
.observation-controls {
  display: grid;
  grid-template-columns:
    minmax(120px, 1.2fr) minmax(180px, 2fr) repeat(5, minmax(90px, 1fr))
    auto;
  gap: 0.55rem;
  padding: 0.7rem 0.75rem;
}
.observation-controls label {
  display: grid;
  min-width: 0;
  gap: 0.22rem;
}
.observation-controls input,
.observation-controls select {
  width: 100%;
  min-height: 32px;
  min-width: 0;
  border: 1px solid var(--border);
  border-radius: 0.3rem;
  background: var(--background);
  padding: 0 0.5rem;
  color: var(--foreground);
  font-size: 0.72rem;
}
.observation-controls input:disabled,
.observation-controls select:disabled,
.clear-filters:disabled {
  cursor: not-allowed;
  opacity: 0.38;
}
.clear-filters {
  align-self: end;
}
.inactive-note {
  margin: 0;
  border-top: 1px solid var(--border);
  padding: 0.45rem 0.75rem;
  color: var(--warning);
  font-size: 0.66rem;
}
@media (max-width: 1050px) {
  .observation-controls {
    grid-template-columns: repeat(4, minmax(110px, 1fr));
  }
  .search-control {
    grid-column: span 2;
  }
}
@media (max-width: 640px) {
  .observation-summary {
    align-items: flex-start;
  }
  .observation-controls {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .search-control {
    grid-column: 1 / -1;
  }
  .clear-filters {
    min-height: 44px;
  }
}
</style>
