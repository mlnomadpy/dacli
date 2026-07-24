<script setup lang="ts">
import { computed } from 'vue'
import type { Graph, GraphNode, Status } from '@/types'
import { statusColor } from '@/composables/useStatusTheme'

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

/** One cubic-bezier per edge, from the source's right edge to the target's left
 * edge. An edge whose both endpoints are on the critical path is itself
 * critical and painted to match. */
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
        critical: from.node.critical && to.node.critical,
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
  <section aria-labelledby="dag-h">
    <div class="section-head">
      <h2 id="dag-h">Dependency graph</h2>
      <span v-if="graph.scheduled" class="cp-chip" title="critical-path length ÷ project duration">
        ★ {{ criticalCount }} on path · {{ graph.duration.toFixed(1) }} Te
      </span>
    </div>

    <p v-if="!hasNodes" class="empty-note">no tasks to graph yet — add a task with a dependency</p>

    <template v-else>
      <!-- The critical path could not be computed (an unestimated open task or a
           cycle): the DAG still draws, but we say so rather than implying a path. -->
      <p v-if="!graph.scheduled && graph.note" class="degrade-note" role="note">
        {{ graph.note }}
      </p>

      <div class="canvas">
        <svg
          :viewBox="`0 0 ${layout.width} ${layout.height}`"
          :width="layout.width"
          :height="layout.height"
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
      <ul class="legend" aria-hidden="true">
        <li><span class="swatch" :style="{ background: statusColor('open') }" />open</li>
        <li><span class="swatch" :style="{ background: statusColor('active') }" />active</li>
        <li><span class="swatch" :style="{ background: statusColor('blocked') }" />blocked</li>
        <li><span class="swatch" :style="{ background: statusColor('done') }" />done</li>
        <li class="cp-key"><span aria-hidden="true">★</span> critical path</li>
      </ul>
      <p class="caption">
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
section {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 14px 16px;
  background: var(--panel);
}
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.cp-chip {
  font-size: 12px;
  color: var(--accent);
  border: 1px solid var(--accent);
  border-radius: 999px;
  padding: 2px 8px;
  font-weight: 600;
}
.empty-note,
.degrade-note {
  margin: 10px 0 0;
  color: var(--muted);
  font-size: 12px;
}
.degrade-note {
  color: var(--blocked);
}
/* Horizontal scroll for a wide graph rather than squashing nodes — readability
 * at 40 tasks beats fitting everything on one screen. */
.canvas {
  margin-top: 12px;
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg);
}
svg {
  display: block;
  max-width: none;
}
.edge {
  stroke: var(--border);
  stroke-width: 1.5;
}
.edge.critical {
  stroke: var(--accent);
  stroke-width: 2.5;
}
.node {
  fill: var(--sc);
  fill-opacity: 0.16;
  stroke: var(--sc);
  stroke-width: 1.5;
}
.node.critical {
  stroke: var(--accent);
  stroke-width: 2.5;
  fill-opacity: 0.24;
}
.n-title {
  fill: var(--text);
  font-size: 12px;
  font-weight: 600;
}
.n-title .star {
  fill: var(--accent);
}
.n-sub {
  fill: var(--muted);
  font-size: 10.5px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.legend {
  list-style: none;
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  margin: 12px 0 0;
  padding: 0;
  font-size: 11px;
  color: var(--muted);
}
.legend li {
  display: flex;
  align-items: center;
  gap: 5px;
}
.swatch {
  width: 10px;
  height: 10px;
  border-radius: 2px;
  display: inline-block;
}
.cp-key {
  color: var(--accent);
  font-weight: 600;
}
.caption {
  margin: 8px 0 0;
  color: var(--muted);
  font-size: 11px;
}
</style>
