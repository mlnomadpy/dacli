<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import BrandMark from '@/components/BrandMark.vue'
import ConnectionStatus from '@/components/ConnectionStatus.vue'
import { useDashboardStore } from '@/stores/dashboard'
import { useStatusTheme } from '@/composables/useStatusTheme'
import { ago, duration } from '@/composables/useRelativeTime'

// `App` is the ONLY thing that touches the network, via the Pinia store's poll
// loop; data flows down as props/getters (DESIGN.md §5). This scaffold renders a
// minimal-but-real projection of the snapshot — Overview + Live swarm — enough
// to exercise the full toolchain end-to-end. The complete component tree
// (TaskBoard, BurndownChart, ProjectSwitcher) is the follow-up build task.
const store = useDashboardStore()
const { phase, error, projects, agents, pendingEvents, generated } = storeToRefs(store)
const { statuses, statusColor, count } = useStatusTheme()

onMounted(() => store.start())
onUnmounted(() => store.stop())

function donePct(done: number, remaining: number): number {
  const total = done + remaining
  return total > 0 ? (done / total) * 100 : 0
}
</script>

<template>
  <header>
    <h1><BrandMark /> dacli dashboard</h1>
    <p class="tagline">mission control — the live agent swarm</p>
    <ConnectionStatus
      :phase="phase"
      :generated="generated"
      :pending-events="pendingEvents"
      :error="error"
      @retry="store.retry()"
    />
  </header>

  <main :class="{ stale: phase === 'error' && store.hasSnapshot }">
    <section aria-labelledby="overview-h">
      <h2 id="overview-h">Overview</h2>
      <div v-if="phase === 'loading'" class="empty">loading…</div>
      <div v-else-if="projects.length === 0" class="empty">no projects yet</div>
      <div v-else class="grid">
        <article v-for="p in projects" :key="p.slug" class="card">
          <h3>{{ p.title || p.slug }}</h3>
          <div class="stage">{{ p.slug }} · stage: {{ p.stage || '—' }}</div>
          <div class="counts">
            <span v-for="s in statuses" :key="s">
              <i class="dot" :style="{ background: statusColor(s) }" />{{ s }}
              {{ count(p.counts, s) }}
            </span>
          </div>
          <div class="bar">
            <div
              class="done-seg"
              :style="{ width: donePct(p.burndown.done_points, p.burndown.remaining_points) + '%' }"
            />
            <div
              class="rem-seg"
              :style="{
                width: 100 - donePct(p.burndown.done_points, p.burndown.remaining_points) + '%',
              }"
            />
          </div>
          <div class="points">
            {{ p.burndown.done_points.toFixed(1) }} done ·
            {{ p.burndown.remaining_points.toFixed(1) }} remaining pts
            <template v-if="p.burndown.unestimated > 0">
              · {{ p.burndown.unestimated }} unestimated
            </template>
          </div>
        </article>
      </div>
    </section>

    <section aria-labelledby="swarm-h">
      <h2 id="swarm-h">Live agent swarm</h2>
      <div v-if="phase === 'loading'" class="empty">loading…</div>
      <div v-else-if="agents.length === 0" class="empty">no live agents</div>
      <table v-else>
        <thead>
          <tr>
            <th scope="col">run</th>
            <th scope="col">child</th>
            <th scope="col">task</th>
            <th scope="col">role</th>
            <th scope="col">runtime</th>
            <th scope="col">pid</th>
            <th scope="col">uptime</th>
            <th scope="col">last activity</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in agents" :key="a.run_id">
            <td class="mono">{{ a.run_id.slice(0, 10) }}</td>
            <td>{{ a.child || '—' }}</td>
            <td>{{ a.task || '—' }}</td>
            <td>{{ a.role || '—' }}</td>
            <td>{{ a.runtime || '—' }}</td>
            <td class="mono">{{ a.pid || '—' }}</td>
            <td>{{ duration(a.runtime_secs) }}</td>
            <td :title="a.last_activity">{{ ago(a.last_activity) }}</td>
          </tr>
        </tbody>
      </table>
    </section>
  </main>
</template>

<style scoped>
header {
  margin-bottom: 8px;
}
h1 {
  font-size: 18px;
  margin: 0 0 4px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.tagline {
  color: var(--muted);
  font-size: 12px;
  margin: 0 0 12px;
}
h2 {
  font-size: 13px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--muted);
  margin: 32px 0 12px;
}
h3 {
  margin: 0 0 8px;
  font-size: 14px;
}
main.stale {
  opacity: 0.6;
  pointer-events: none;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}
.card {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 14px 16px;
}
.stage {
  color: var(--muted);
  font-size: 12px;
  margin-bottom: 10px;
}
.counts {
  display: flex;
  gap: 10px;
  font-size: 12px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}
.counts span {
  display: flex;
  align-items: center;
  gap: 4px;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}
.bar {
  display: flex;
  height: 8px;
  border-radius: 4px;
  overflow: hidden;
  background: var(--border);
  margin-bottom: 6px;
}
.bar .done-seg {
  background: var(--done);
}
.bar .rem-seg {
  background: var(--active);
}
.points {
  color: var(--muted);
  font-size: 11px;
}
table {
  width: 100%;
  border-collapse: collapse;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}
th,
td {
  text-align: left;
  padding: 8px 12px;
  font-size: 12px;
  border-bottom: 1px solid var(--border);
}
th {
  color: var(--muted);
  font-weight: 600;
  text-transform: uppercase;
  font-size: 10px;
  letter-spacing: 0.05em;
}
tr:last-child td {
  border-bottom: none;
}
.empty {
  color: var(--muted);
  padding: 16px;
  font-size: 13px;
}
</style>
