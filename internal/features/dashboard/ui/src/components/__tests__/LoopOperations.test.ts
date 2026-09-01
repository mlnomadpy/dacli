import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import LoopOperations from '@/components/LoopOperations.vue'
import type { LoopOperationResponse } from '@/types'

const operation: LoopOperationResponse = {
  schema: 'loop-operation/v1',
  generated: '2026-09-01T20:00:00Z',
  project: 'core',
  state: {
    value: 'halted-policy',
    freshness: 'fresh',
    source: 'cycle-preflight',
    cycle: 4,
    generation: 4,
    phase: 'review-pending',
    checkpoint: 'complete-cycle-preflight',
    retryable: false,
    halt_class: 'permanent_refusal',
    reason: 'reviewer capacity is unavailable',
    next_action: 'assign an independent reviewer before resuming',
  },
  wave: { requested_width: 2, allocated_width: 1, live_width: 1 },
  budget: {
    mode: 'advisory',
    cycle: { unit: 'output_tokens', limit: 20_000, spent: 4_000, reserved: 8_000, remaining: null },
    rolling: {
      unit: 'output_tokens',
      limit: 80_000,
      spent: 12_000,
      reserved: 8_000,
      remaining: null,
    },
    runs: [],
    review_reservation: 2_000,
    recovery_reserve: 1_000,
    unallocated: null,
    unknown_usage_runs: ['01UNKNOWN'],
    accounting_boundary: 'provider-reported output tokens; not billing',
  },
  tasks: [
    {
      task: 't-01TASK942',
      phase: 'review-pending',
      generation: 4,
      role: 'codex-reviewer',
      runtime: 'codex',
      model: 'gpt',
      grant: 'ro',
      claim_count: 2,
      capacity_fit: false,
      task_points: 8,
      role_limit: 5,
      verification_cwd: 'internal/features/dashboard',
      verification_argv: 'go test ./internal/features/dashboard',
    },
  ],
  active_runs: [],
  routing: [
    {
      task: 't-01TASK942',
      selected: { role: 'codex-reviewer', runtime: 'codex', model: 'gpt' },
      candidates: [
        {
          role: 'codex-reviewer',
          runtime: 'codex',
          model: 'gpt',
          eligible: true,
          score: {
            cost_tier: 1,
            tokens_per_completed: 1000,
            token_samples: 12,
            first_pass_success: 0.8,
            success_samples: 12,
            latency_seconds: 30,
            domain_fit: 2,
            total: 10,
          },
        },
      ],
      source: 'cheapest capable; bounded by recorded harness policy',
      freshness: 'fresh',
    },
  ],
  preflight: [
    {
      phase: 'reviewer-runtime',
      role: 'codex-reviewer',
      verdict: 'refuse',
      classification: 'permanent_refusal',
      remediation: 'restore reviewer capacity',
    },
  ],
  harness: { mode: 'single', allowed: ['codex'], source: 'operating-profile' },
  warnings: [],
}

describe('LoopOperations', () => {
  it('renders policy refusal, advisory budgets, provider boundary, and evidence links honestly', () => {
    const wrapper = mount(LoopOperations, { props: { operation } })
    expect(wrapper.text()).toContain('halted policy')
    expect(wrapper.text()).toContain('policy answer — do not retry unchanged')
    expect(wrapper.text()).toContain('not enforceable')
    expect(wrapper.text()).toContain('provider-reported output tokens; not billing')
    expect(wrapper.text()).toContain('single · codex')
    expect(wrapper.text()).not.toContain('claude')
    expect(wrapper.text()).toContain('capacity refused')
    expect(wrapper.text()).toContain('internal/features/dashboard · go test')
    expect(wrapper.get('a').attributes('href')).toContain('task=t-01TASK942')
    expect(wrapper.find('[role="progressbar"]').exists()).toBe(false)
  })

  it('never styles a corrupt record as healthy', () => {
    const corrupt = structuredClone(operation)
    corrupt.state.value = 'corrupt'
    corrupt.state.freshness = 'corrupt'
    corrupt.warnings = ['token reservation ledger: identity mismatch']
    const wrapper = mount(LoopOperations, { props: { operation: corrupt } })
    expect(wrapper.get('[data-tone="danger"]').text()).toBe('corrupt')
    expect(wrapper.get('[role="alert"]').text()).toContain('token reservation ledger')
  })
})
