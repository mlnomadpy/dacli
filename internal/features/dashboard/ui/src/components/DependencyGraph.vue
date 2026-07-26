<script setup lang="ts">
import { computed } from 'vue'
import type { Graph, GraphNode, Status } from '@/types'
import { statusColor } from '@/composables/useStatusTheme'
import { Badge } from '@/components/ui/badge'

// The task dependency DAG (task 145). Nodes are tasks colored by status, edges
// are depends_on relations flowing left→right by dependency depth, and the CPM
// critical path — the chain internal/spm/criticalpath.go computes — is
// highlighted so the operator stops reconstructing it by hand. Data-only prop
// like the other chart leaves; the store owns the poll, the parent hands down
// the selected project's `graph`.
const props = defineProps<{ graph: Graph }>()

// Layout constants. Sized so a 10–40 task graph stays readable: wide-enough
// nodes for a seq + slug, generous column gap for the edges to breathe.
const NODE_W = 158
const NODE_H = 44
const COL_GAP = 66
const ROW_GAP = 16
const PAD = 16

const hasNodes = computed(() => props.graph.nodes.length > 0)

/** longest-path layer per node = the longest chain of dependencies ending at
 * it, so a node always sits to the right of every task it depends on. A cycle
 * (should never happen in a real DAG, but the server may span messy data)
 * is broken by a visiting-guard that treats the back-edge as depth 0. */
function layerOf(nodes: GraphNode[], edges: Graph['edges']): Map<string, number> {
  const ids = new Set(nodes.map((n) => n.id))
  const preds = new Map<string, string[]>()
  for (const e of edges) {
    if (!ids.has(e.from) || !ids.has(e.to)) continue
    const arr = preds.get(e.to)
    if (arr) arr.push(e.from)
    else preds.set(e.to, [e.from])
  }
  const depth = new Map<string, number>()
  const visiting = new Set<string>()
  function calc(id: string): number {
    const cached = depth.get(id)
    if (cached !== undefined) return cached
    if (visiting.has(id)) return 0 // cycle guard
    visiting.add(id)
    let d = 0
    for (const p of preds.get(id) ?? []) d = Math.max(d, calc(p) + 1)
    visiting.delete(id)
    depth.set(id, d)
    return d
  }
  for (const n of nodes) calc(n.id)
  return depth
}

interface Placed {
  node: GraphNode
  x: number
  y: number
}

/** Place every node on a (layer × row) grid: layer → column x, position within
 * the layer (ordered by seq for stability) → row y. Also reports the canvas
 * size so the SVG viewBox fits exactly. */
const layout = computed(() => {
  const nodes = props.graph.nodes
  const depth = layerOf(nodes, props.graph.edges)
  const byLayer = new Map<number, GraphNode[]>()
  let maxLayer = 0
  for (const n of nodes) {
    const l = depth.get(n.id) ?? 0
    maxLayer = Math.max(maxLayer, l)
    const arr = byLayer.get(l)
    if (arr) arr.push(n)
    else byLayer.set(l, [n])
  }
  const byId = new Map<string, Placed>()
  let maxRows = 0
  for (let l = 0; l <= maxLayer; l++) {
    const col = (byLayer.get(l) ?? []).slice().sort((a, b) => a.seq - b.seq)
    maxRows = Math.max(maxRows, col.length)
    col.forEach((node, row) => {
      byId.set(node.id, {
        node,
        x: PAD + l * (NODE_W + COL_GAP),
        y: PAD + row * (NODE_H + ROW_GAP),
      })
    })
  }
  const width = PAD * 2 + (maxLayer + 1) * NODE_W + maxLayer * COL_GAP
  const rows = Math.max(1, maxRows)
  const height = PAD * 2 + rows * NODE_H + (rows - 1) * ROW_GAP
  return { placed: [...byId.values()], byId, width, height }
})

/** The consecutive (from→to) pairs of the ordered critical path. An edge is
 * critical iff it IS one of these adjacencies — NOT merely because both its
 * endpoints happen to be critical. A redundant edge between two on-path but
 * non-adjacent tasks (a direct A→B when the path runs A→C→B) has positive slack
 * and must render as an ordinary edge, so we key on the path's adjacency, not
 * on the node flags. */
const criticalEdges = computed(() => {
  const pairs = new Set<string>()
  const cp = props.graph.critical_path
  for (let i = 0; i + 1 < cp.length; i++) pairs.add(`${cp[i]}->${cp[i + 1]}`)
  return pairs
})

/** One cubic-bezier per edge, from the source's right edge to the target's left
 * edge. An edge is painted critical only when it is a consecutive pair on the
 * ordered critical path (see `criticalEdges`). */
const edgePaths = computed(() =>
  props.graph.edges.flatMap((e) => {
    const from = layout.value.byId.get(e.from)
    const to = layout.value.byId.get(e.to)
    if (!from || !to) return []
    const x1 = from.x + NODE_W
    const y1 = from.y + NODE_H / 2
    const x2 = to.x
    const y2 = to.y + NODE_H / 2
    const mx = (x1 + x2) / 2
    return [
      {
        key: `${e.from}->${e.to}`,
        d: `M${x1},${y1} C${mx},${y1} ${mx},${y2} ${x2},${y2}`,
        critical: criticalEdges.value.has(`${e.from}->${e.to}`),
        type: e.type,
      },
    ]
  }),
)

/** Pass the node's status color to the scoped CSS as a custom property so a
 * single rule can tint fill + stroke without a per-status class explosion. */
function nodeStyle(status: Status): Record<string, string> {
  return { '--sc': statusColor(status) }
}

function truncate(s: string, max: number): string {
  return s.length > max ? s.slice(0, max - 1) + '…' : s
}

const criticalCount = computed(() => props.graph.critical_path.length)

/** A one-line summary for the SVG's aria-label — the whole point of the view,
 * spoken: how many tasks, and where the critical path is. */
const summary = computed(() => {
  const n = props.graph.nodes.length
  const e = props.graph.edges.length
  if (props.graph.scheduled) {
    return `dependency graph of ${n} task(s), ${e} edge(s); critical path is ${criticalCount.value} task(s), ${props.graph.duration.toFixed(1)} Te units`
  }
  return `dependency graph of ${n} task(s), ${e} edge(s); ${props.graph.note || 'no critical path'}`
})
</script>

<template>
  <section aria-labelledby="dag-h" class="rounded-lg border border-border bg-card px-4 py-3.5">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h2
        id="dag-h"
        class="m-0 text-[13px] font-semibold uppercase tracking-[0.06em] text-muted-foreground"
      >
        Dependency graph
      </h2>
      <Badge
        v-if="graph.scheduled"
        variant="outline"
        class="cp-chip border-primary font-semibold text-primary"
        title="critical-path length ÷ project duration"
      >
        ★ {{ criticalCount }} on path · {{ graph.duration.toFixed(1) }} Te
      </Badge>
    </div>

    <p v-if="!hasNodes" class="mt-2.5 text-xs text-muted-foreground">
      no tasks to graph yet — add a task with a dependency
    </p>

    <template v-else>
      <!-- The critical path could not be computed (an unestimated open task or a
           cycle): the DAG still draws, but we say so rather than implying a path. -->
      <p
        v-if="!graph.scheduled && graph.note"
        class="degrade-note mt-2.5 text-xs text-destructive"
        role="note"
      >
        {{ graph.note }}
      </p>

      <div class="mt-3 overflow-x-auto rounded-md border border-border bg-background">
        <svg
          :viewBox="`0 0 ${layout.width} ${layout.height}`"
          :width="layout.width"
          :height="layout.height"
          class="block max-w-none"
          role="img"
          :aria-label="summary"
        >
          <!-- Edges first, so nodes paint over the line ends. -->
          <g class="edges" fill="none">
            <path
              v-for="p in edgePaths"
              :key="p.key"
              :d="p.d"
              class="edge"
              :class="{ critical: p.critical }"
              :stroke-dasharray="p.type === 'SS' ? '4 3' : undefined"
            />
          </g>

          <g
            v-for="p in layout.placed"
            :key="p.node.id"
            class="node-group"
            :transform="`translate(${p.x},${p.y})`"
          >
            <rect
              class="node"
              :class="{ critical: p.node.critical }"
              :width="NODE_W"
              :height="NODE_H"
              rx="7"
              :style="nodeStyle(p.node.status)"
            />
            <text class="n-title" :x="10" :y="18">
              <tspan v-if="p.node.critical" class="star" aria-hidden="true">★</tspan>
              {{ p.node.seq }} · {{ truncate(p.node.slug, 16) }}
            </text>
            <text class="n-sub" :x="10" :y="34">
              {{ p.node.status }}
              <template v-if="p.node.points > 0">· Te {{ p.node.points.toFixed(1) }}</template>
            </text>
          </g>
        </svg>
      </div>

      <!-- Legend: what the colors and the ★ mean — the map's key. -->
      <ul
        class="mt-3 flex list-none flex-wrap gap-3.5 p-0 text-[11px] text-muted-foreground"
        aria-hidden="true"
      >
        <li class="flex items-center gap-1.5">
          <span class="size-2.5 rounded-sm" :style="{ background: statusColor('open') }" />open
        </li>
        <li class="flex items-center gap-1.5">
          <span class="size-2.5 rounded-sm" :style="{ background: statusColor('active') }" />active
        </li>
        <li class="flex items-center gap-1.5">
          <span
            class="size-2.5 rounded-sm"
            :style="{ background: statusColor('blocked') }"
          />blocked
        </li>
        <li class="flex items-center gap-1.5">
          <span class="size-2.5 rounded-sm" :style="{ background: statusColor('done') }" />done
        </li>
        <li class="flex items-center gap-1.5 font-semibold text-primary">
          <span aria-hidden="true">★</span> critical path
        </li>
      </ul>
      <p class="mt-2 text-[11px] text-muted-foreground">
        {{ graph.nodes.length }} task(s), {{ graph.edges.length }} dependency edge(s)<template
          v-if="graph.scheduled"
        >
          · ★ = spawn children here first, slack tasks can wait</template
        >
      </p>
    </template>
  </section>
</template>

<style scoped>
/* SVG internals keep a scoped stylesheet — fill/stroke on nodes vary per status
 * via the `--sc` custom property, which Tailwind utilities can't express. Colors
 * reference the shadcn-vue theme tokens (index.css), not the legacy vars. */
.edge {
  stroke: var(--border);
  stroke-width: 1.5;
}
.edge.critical {
  stroke: var(--primary);
  stroke-width: 2.5;
}
.node {
  fill: var(--sc);
  fill-opacity: 0.16;
  stroke: var(--sc);
  stroke-width: 1.5;
}
.node.critical {
  stroke: var(--primary);
  stroke-width: 2.5;
  fill-opacity: 0.24;
}
.n-title {
  fill: var(--foreground);
  font-size: 12px;
  font-weight: 600;
}
.n-title .star {
  fill: var(--primary);
}
.n-sub {
  fill: var(--muted-foreground);
  font-size: 10.5px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
</style>
