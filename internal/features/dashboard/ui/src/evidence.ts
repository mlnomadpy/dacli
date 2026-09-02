import type { DashboardState, OutcomeMeasure, OutcomeMetric } from '@/types'
import fixture from '@/__tests__/fixtures/dashboard-state.json'

const snapshot = fixture as DashboardState
const generated = snapshot.generated
const evidenceMode = new URLSearchParams(window.location.search).get('state') ?? 'live'
let outcomeReads = 0
const exactEvidence = {
  tasks: ['t-901', 't-904', 't-907'],
  runs: ['run-901', 'run-904'],
  truncated: false,
}

function measure(
  key: string,
  label: string,
  value: number | null,
  unit: string,
  state: OutcomeMeasure['state'] = 'complete',
): OutcomeMeasure {
  return {
    key,
    label,
    value,
    unit,
    sample_size: value === null ? 0 : 18,
    eligible: 21,
    coverage: value === null ? 0 : 0.86,
    state,
    provenance: 'exact durable task, run, verification, review, and landing records',
    caveat: state === 'partial' ? 'Three historical runs have no provider usage report.' : '',
    evidence: exactEvidence,
  }
}

function metric(
  key: string,
  label: string,
  value: number | null,
  previous: number | null,
  unit: string,
  state: OutcomeMeasure['state'] = 'complete',
): OutcomeMetric {
  return {
    key,
    label,
    current: measure(key, label, value, unit, state),
    previous: measure(key, label, previous, unit, state),
    change: value === null || previous === null ? null : value - previous,
    trend:
      value === null || previous === null ? 'not-comparable' : value >= previous ? 'up' : 'down',
  }
}

const days = [
  ['2026-08-24', 1, 2, 9200],
  ['2026-08-25', 2, 3, 12800],
  ['2026-08-26', 1, 3, 11400],
  ['2026-08-27', 3, 4, 18100],
  ['2026-08-28', 2, 3, 13600],
  ['2026-08-29', 4, 5, 20200],
  ['2026-08-30', 3, 4, 16900],
  ['2026-08-31', 2, 3, 12100],
  ['2026-09-01', 4, 5, 19800],
  ['2026-09-02', 3, 4, 15400],
] as const

function attentionResponse() {
  return {
    schema: 'operator-attention/v1',
    generated,
    alerts: [
      {
        id: 'github_state_unknown/dacli/t-907',
        code: 'github_state_unknown',
        severity: 'critical',
        affected: { project: 'dacli', task: 't-907', pr: '970' },
        first_observed: '2026-08-30T09:10:00Z',
        last_observed: generated,
        freshness: 'stale',
        retryable: true,
        summary: 'Required GitHub check state is unknown; absence is not treated as healthy.',
        next_action: 'Re-observe the exact PR head after GitHub recovers.',
        link: '#/delivery?project=dacli&task=t-907',
        evidence: [
          {
            kind: 'github-check',
            id: 'check-970',
            url: '#/delivery?project=dacli&task=t-907',
            observed_at: generated,
            confidence: 'low',
          },
        ],
        occurrences: 3,
        duration_seconds: 12000,
        critical_path: true,
        confidence: 'low',
        rank: 1,
        rank_reason: 'severity=critical; critical_path=true; age=3h20m; confidence=low',
      },
      {
        id: 'owner_handoff/dacli/t-905/run-905',
        code: 'owner_handoff',
        severity: 'high',
        affected: { project: 'dacli', task: 't-905', run: 'run-905' },
        first_observed: '2026-08-30T11:50:00Z',
        last_observed: generated,
        freshness: 'fresh',
        retryable: false,
        summary: 'A worker recorded an exact root-only publication handoff.',
        next_action: 'Inspect hashes and consume the root handoff after re-observation.',
        link: '#/agents?project=dacli&agent=a-capture',
        evidence: [
          {
            kind: 'run',
            id: 'run-905',
            url: '#/agents?project=dacli&agent=a-capture',
            observed_at: generated,
            confidence: 'high',
          },
        ],
        occurrences: 1,
        duration_seconds: 2400,
        critical_path: false,
        confidence: 'high',
        rank: 2,
        rank_reason: 'severity=high; owner_action=true; age=40m; confidence=high',
      },
    ],
    ranking_rule: 'severity, critical path, age, confidence, stable identity',
  }
}

function outcomeResponse() {
  return {
    schema: 'outcome-analytics/v1',
    generated,
    project: 'dacli',
    current_window: { start: '2026-08-04T12:30:00Z', end: generated, days: 30 },
    previous_window: {
      start: '2026-07-05T12:30:00Z',
      end: '2026-08-04T12:30:00Z',
      days: 30,
    },
    metrics: [
      metric('throughput', 'Throughput', 25, 18, 'tasks'),
      metric('execution_time', 'Execution time', 2.7, 3.4, 'hours'),
      metric('current_tree_acceptance', 'Accepted on current tree', 86, 79, 'percent'),
      metric('first_pass_review', 'First-pass review', 72, 68, 'percent'),
      metric('retry_rate', 'Retry rate', 14, 19, 'percent'),
      metric('cost', 'Provider-reported cost', 11.84, 9.72, 'USD', 'partial'),
    ],
    breakdowns: [
      {
        dimension: 'task_size',
        key: 'medium',
        size_band: 'medium',
        current: measure('throughput', 'Throughput', 12, 'tasks'),
        previous: measure('throughput', 'Throughput', 8, 'tasks'),
        comparable: true,
        evidence: exactEvidence,
      },
      {
        dimension: 'runtime',
        key: 'codex',
        size_band: 'mixed',
        current: measure('current_tree_acceptance', 'Accepted on current tree', 86, 'percent'),
        previous: measure('current_tree_acceptance', 'Accepted on current tree', 79, 'percent'),
        comparable: true,
        evidence: exactEvidence,
      },
    ],
    series: days.map(([day, completed, runs, tokens]) => ({
      day,
      completed,
      runs,
      tokens,
      evidence: exactEvidence,
    })),
    performance: {
      tasks_scanned: 553,
      runs_scanned: 742,
      series_points: days.length,
      build_ms: 18,
      evidence_cap: 100,
      cache: 'fresh-index',
      cache_entries: 2,
    },
    notes: [
      'Descriptive adjacent-window evidence; not a causal model ranking.',
      'Provider-reported usage is not a billing statement. Missing cost remains unknown.',
    ],
  }
}

function fixtureResponse(url: string): unknown {
  if (url === '/api/overview') {
    return {
      generated,
      project_count: snapshot.projects.length,
      task_count: snapshot.projects.reduce((total, project) => total + project.total, 0),
      counts: { open: 26, active: 9, blocked: 6, done: 519 },
      pending_events: snapshot.pending_events,
      live_agents: snapshot.agents.length,
    }
  }
  if (url === '/api/projects') return { generated, projects: snapshot.projects }
  if (url === '/api/attention') return attentionResponse()
  if (url === '/api/agents') return { generated, agents: snapshot.agents }
  if (url === '/api/burn') return { generated, ...snapshot.burn }
  if (url === '/api/roles') return { generated, roles: snapshot.roles }
  if (url === '/api/loop-operation?project=dacli') {
    return {
      schema: 'loop-operation/v1',
      generated,
      project: 'dacli',
      state: {
        value: 'running',
        freshness: 'fresh',
        source: 'loop-phase-journal/v1',
        cycle: 7,
        generation: 7,
        phase: 'reviewed',
        next_action: 'observe required checks on the exact PR head',
      },
      wave: { requested_width: 3, allocated_width: 2, live_width: 2 },
      budget: {
        mode: 'enforceable',
        cycle: {
          unit: 'output_tokens',
          limit: 30000,
          spent: 9400,
          reserved: 12000,
          remaining: 8600,
        },
        rolling: {
          unit: 'output_tokens',
          limit: 100000,
          spent: 38200,
          reserved: 12000,
          remaining: 49800,
        },
        runs: [],
        review_reservation: 3000,
        recovery_reserve: 1500,
        unallocated: 8600,
        unknown_usage_runs: [],
        accounting_boundary: 'provider-reported output tokens; not billing',
      },
      tasks: [],
      active_runs: [],
      routing: [],
      preflight: [],
      harness: { mode: 'single', allowed: ['codex'], source: 'operator profile' },
      warnings: [],
    }
  }
  if (url.startsWith('/api/outcomes?project=dacli')) return outcomeResponse()
  return null
}

window.fetch = (async (input: RequestInfo | URL) => {
  const url = String(input)
  if (evidenceMode === 'cold-error') {
    return new Response('representative upstream unavailable', { status: 503 })
  }
  if (url.startsWith('/api/outcomes?') && evidenceMode === 'stale-outcomes') {
    outcomeReads += 1
    if (outcomeReads > 1) {
      return new Response('representative analytics refresh unavailable', { status: 503 })
    }
  }
  const body = fixtureResponse(url)
  if (body === null) return new Response('not found', { status: 404 })
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}) as typeof fetch

if (!window.location.hash) window.location.hash = '#/overview?project=dacli'

void import('./main')
