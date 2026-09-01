import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DependencyGraph from '../DependencyGraph.vue'
import { emptyGraph } from '@/types'
import type { Graph, GraphNode } from '@/types'

function node(over: Partial<GraphNode> = {}): GraphNode {
  return {
    id: 'x',
    seq: 1,
    slug: 'a-task',
    title: 'A task',
    status: 'open',
    points: 0,
    estimated: false,
    critical: false,
    slack: -1,
    early_start: 0,
    ...over,
  }
}

function graph(over: Partial<Graph> = {}): Graph {
  return { ...emptyGraph(), ...over }
}

// A linear A→B→C chain, entirely on the critical path, plus a standalone done
// task off the path — the same shape the Go handler test builds.
function chainGraph(): Graph {
  return graph({
    project: 'core',
    scheduled: true,
    duration: 6,
    critical_path: ['a', 'b', 'c'],
    nodes: [
      node({
        id: 'a',
        seq: 1,
        slug: 'design',
        status: 'open',
        estimated: true,
        points: 2,
        critical: true,
        slack: 0,
      }),
      node({
        id: 'b',
        seq: 2,
        slug: 'build',
        status: 'active',
        estimated: true,
        points: 2,
        critical: true,
        slack: 0,
      }),
      node({
        id: 'c',
        seq: 3,
        slug: 'ship',
        status: 'open',
        estimated: true,
        points: 2,
        critical: true,
        slack: 0,
      }),
      node({ id: 'd', seq: 4, slug: 'old-work', status: 'done', estimated: true, points: 2 }),
    ],
    edges: [
      { from: 'a', to: 'b', type: 'FS' },
      { from: 'b', to: 'c', type: 'FS' },
    ],
  })
}

/** Parse the x offset out of a `translate(x,y)` transform. */
function xOf(transform: string | undefined): number {
  const m = /translate\(([-\d.]+),/.exec(transform ?? '')
  return m ? Number(m[1]) : NaN
}

describe('DependencyGraph', () => {
  it('is empty-safe: an empty graph draws no SVG and says so', () => {
    const w = mount(DependencyGraph, { props: { graph: emptyGraph() } })
    expect(w.text()).toContain('no tasks to graph yet')
    expect(w.find('svg').exists()).toBe(false)
    expect(w.findAll('.node')).toHaveLength(0)
    expect(w.html()).not.toContain('NaN')
  })

  it('draws a node per task and an edge per dependency', () => {
    const w = mount(DependencyGraph, { props: { graph: chainGraph() } })
    expect(w.findAll('.node')).toHaveLength(4)
    expect(w.findAll('.edge')).toHaveLength(2)
    // Every node label carries its seq and slug.
    expect(w.text()).toContain('design')
    expect(w.text()).toContain('ship')
    // No NaN leaks into a path or transform.
    expect(w.html()).not.toContain('NaN')
  })

  it('makes every visible task node keyboard-focusable and inspectable', async () => {
    const w = mount(DependencyGraph, { props: { graph: chainGraph() } })
    const nodes = w.findAll('[role="button"]')
    expect(nodes).toHaveLength(4)
    expect(nodes.every((task) => task.attributes('tabindex') === '0')).toBe(true)
    expect(nodes[0].attributes('aria-label')).toContain('Inspect task')
    await nodes[0].trigger('keydown', { key: 'Enter' })
    expect(w.emitted('inspect')?.[0]?.[0]).toBe(chainGraph().nodes[0].id)
  })

  it('highlights the critical path on nodes AND edges', () => {
    const w = mount(DependencyGraph, { props: { graph: chainGraph() } })
    // Three of the four nodes are critical (the done task is not).
    expect(w.findAll('.node.critical')).toHaveLength(3)
    // Both edges run between two critical nodes, so both are critical.
    expect(w.findAll('.edge.critical')).toHaveLength(2)
    // The header chip reports the critical-path length and duration.
    expect(w.find('.cp-chip').exists()).toBe(true)
    expect(w.find('.cp-chip').text()).toContain('3 on path')
    expect(w.find('.cp-chip').text()).toContain('6.0 Te')
  })

  it('marks an edge critical by adjacency on the path, not by both endpoints being critical', () => {
    // A, C, B are all on the critical path A→C→B. There is the on-path chain
    // (A→C, C→B) plus a REDUNDANT direct edge A→B: both its endpoints are
    // critical, but the pair (A,B) is not consecutive on the path, so it has
    // positive slack and must render as an ordinary (non-critical) edge.
    const g = graph({
      scheduled: true,
      duration: 6,
      critical_path: ['a', 'c', 'b'],
      nodes: [
        node({ id: 'a', seq: 1, slug: 'a', critical: true, estimated: true, points: 2 }),
        node({ id: 'c', seq: 2, slug: 'c', critical: true, estimated: true, points: 2 }),
        node({ id: 'b', seq: 3, slug: 'b', critical: true, estimated: true, points: 2 }),
      ],
      edges: [
        { from: 'a', to: 'c', type: 'FS' },
        { from: 'c', to: 'b', type: 'FS' },
        { from: 'a', to: 'b', type: 'FS' }, // redundant, off the path
      ],
    })
    const w = mount(DependencyGraph, { props: { graph: g } })
    // Three edges drawn, but only the two on-path adjacencies are critical.
    expect(w.findAll('.edge')).toHaveLength(3)
    expect(w.findAll('.edge.critical')).toHaveLength(2)
  })

  it('preserves the SS dasharray on an edge that is also on the critical path', () => {
    const g = graph({
      scheduled: true,
      critical_path: ['a', 'b'],
      nodes: [
        node({ id: 'a', seq: 1, slug: 'a', critical: true, estimated: true, points: 2 }),
        node({ id: 'b', seq: 2, slug: 'b', critical: true, estimated: true, points: 2 }),
      ],
      edges: [{ from: 'a', to: 'b', type: 'SS' }],
    })
    const w = mount(DependencyGraph, { props: { graph: g } })
    const edge = w.find('.edge')
    // Both the critical highlight and the SS dash survive together.
    expect(edge.classes()).toContain('critical')
    expect(edge.attributes('stroke-dasharray')).toBe('4 3')
  })

  it('lays dependents to the right of what they depend on (layered DAG)', () => {
    const w = mount(DependencyGraph, { props: { graph: chainGraph() } })
    const xBySlug = new Map<string, number>()
    for (const g of w.findAll('.node-group')) {
      const slug = g.find('.n-title').text()
      xBySlug.set(slug, xOf(g.attributes('transform')))
    }
    // design (layer 0) < build (layer 1) < ship (layer 2).
    const dx = [...xBySlug.entries()].find(([s]) => s.includes('design'))![1]
    const bx = [...xBySlug.entries()].find(([s]) => s.includes('build'))![1]
    const sx = [...xBySlug.entries()].find(([s]) => s.includes('ship'))![1]
    expect(dx).toBeLessThan(bx)
    expect(bx).toBeLessThan(sx)
  })

  it('still draws the DAG but no path (and shows the note) when unscheduled', () => {
    const g = graph({
      scheduled: false,
      note: '1 open task(s) lack a PERT estimate — DAG shown without the critical path',
      nodes: [
        node({ id: 'a', seq: 1, slug: 'estimated', estimated: true, points: 2 }),
        node({ id: 'b', seq: 2, slug: 'unestimated', estimated: false }),
      ],
      edges: [{ from: 'a', to: 'b', type: 'FS' }],
    })
    const w = mount(DependencyGraph, { props: { graph: g } })
    // The DAG is still fully drawn.
    expect(w.findAll('.node')).toHaveLength(2)
    expect(w.findAll('.edge')).toHaveLength(1)
    // ...but nothing is marked critical and the reason is shown.
    expect(w.findAll('.node.critical')).toHaveLength(0)
    expect(w.find('.cp-chip').exists()).toBe(false)
    expect(w.find('.degrade-note').text()).toContain('lack a PERT estimate')
  })

  it('dashes a start-start edge, which is the one that permits overlap', () => {
    const g = graph({
      scheduled: true,
      nodes: [
        node({ id: 'a', seq: 1, slug: 'a', critical: false }),
        node({ id: 'b', seq: 2, slug: 'b', critical: false }),
      ],
      edges: [{ from: 'a', to: 'b', type: 'SS' }],
    })
    const w = mount(DependencyGraph, { props: { graph: g } })
    expect(w.find('.edge').attributes('stroke-dasharray')).toBe('4 3')
  })
})
