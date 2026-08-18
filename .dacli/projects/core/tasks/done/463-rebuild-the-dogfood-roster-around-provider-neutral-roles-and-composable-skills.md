---
id: t-01M0AEG5694R7SDMSREJ8KPF4K
kind: task
created: 2026-08-18T12:45:49Z
created_by: a-root
owner: a-root
priority: must
depends_on: [468]
github:
  issue: 689
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Rebuild the dogfood roster around provider-neutral roles and composable skills
## Context
Adopted from GitHub issue #689.

## Problem

The dogfood roster no longer represents a coherent, provider-neutral team:

- only 2 of 25 roles select any workspace skill;
- 17 roles are pinned to Opus even when their task capacity is 2–8 points;
- `prompt-auditor`, `role-architect`, and `visionary` have no version;
- the two Codex loop roles and both bootstrap loop roles omit `role_kind`, so automatic phase routing cannot reason about them;
- `codex-process-architect` is a permanent role whose prompt is tied to completed task 375;
- overlapping `fixer`, `maintainer`, `codex-maintainer`, and bootstrap-maintainer roles encode vendor/history in the job definition rather than capabilities;
- only `using-dacli` exists as a reusable skill, and compiling it for `codex-rw` reports an approximately 1,968-token inline tax every turn.

`dacli doctor` reports no anti-patterns and `preflight` reports no mismatch for the main roles, so the current validation surface cannot detect this roster drift.

## Design

Model roles by responsibility and blast radius, not provider name. Keep runtime/model as replaceable execution policy. Extract compact, composable skills for shared operating knowledge (evidence and verification, Go architecture/persistence, GitHub delivery, runtime/process safety, frontend quality, product research) and assign only the relevant set to each role. Keep the mandatory inline body small; put detailed lookup material in resources or role-local method text so progressive disclosure is preserved on runtimes without native skill delivery.

Retire task-specific and bootstrap-only roles once no live agent references them. Give every durable role a kind, version, cost/capacity profile, scope/out-of-scope boundary, escalation path, and provider fallback where an independently probed runtime exists. Use cheap models for bounded low-blast-radius work and reserve frontier models for ambiguous, security-sensitive, concurrency, persistence, or architecture work.



## Evidence

Run on 2026-08-18:

```text
dacli role list                 # 25 roles; 17 model:opus; 2 roles with skills
dacli skill compile using-dacli --runtime codex-rw --dry-run
# total per-turn tax on codex-rw: ~1968 tokens — progressive disclosure is gone on this target
dacli doctor                    # no anti-patterns detected
```

## Acceptance
- [x] Every durable role has `version`, `role_kind`, grant, runtime, provider-neutral cost/capacity fields, scope or explicit workspace-wide scope, out-of-scope guidance, escalation, and at least one relevant skill.
- [x] Task-specific and bootstrap-only roles are removed or converted into reusable capability roles; removal refuses while a live agent still holds a role.
- [x] The implementer/reviewer/designer/researcher/planner families have distinct, checkable methods and do not duplicate a provider-named copy of the same responsibility.
- [x] Model selection uses at least two cost tiers and documents when blast radius raises the tier; Opus/frontier models are not the unconditional default for 2–8 point work.
- [x] Reviewer independence is preserved through an explicit different-runtime fallback/panel policy where locally available and probed.
- [x] Reusable role-family skills cover evidence/mutation testing, Go system design and persistence, runtime/process safety, GitHub-first delivery, frontend quality, and research/design as applicable.
- [x] Skill compilation reports the per-turn tax for every role/runtime pair and keeps the mandatory inline total within a documented budget.
- [x] `dacli preflight` succeeds for every active role; strict read-only roles use only a runtime whose behavioral read-only probe succeeds, or are explicitly marked cooperative and excluded from unattended loops.
- [x] A roster invariant test or `doctor` check detects missing versions/kinds/skills, task-specific role prompts, unsupported runtime/model combinations, and excessive default-model concentration.
- [x] `docs/ROSTER.md`, role/skill docs, and the installed dacli operator skill are regenerated or refreshed from the same committed definitions.
## Log
- 2026-08-18T12:48:13Z claimed by a-codex-maintainer-as8sk8
- 2026-08-18T14:05:36Z accepted by a-root
- 2026-08-18T14:05:36Z verified by `env GOCACHE=/tmp/dacli-463-accept-gocache GOTMPDIR=/tmp go test ./...` (exit 0) in branch main at 7256a42 — proves that tree builds, not that the work is in trunk
- 2026-08-18T14:05:36Z deliverable: dacli/463-rebuild-the-dogfood-roster-around-provider-neutral-roles-and-composable-skills is merged into main
- 2026-08-18T14:05:36Z completed by a-root
## Verification Evidence
{"command":"env GOCACHE=/tmp/dacli-463-accept-gocache GOTMPDIR=/tmp go test ./...","exit_code":0,"duration_ms":81874,"artifact_hash":"sha256:cac1afed09e1e8bf256bb2e132d37b174eb4c67c04703003003ccc83f1fd42a8","verifier":"a-root","branch":"main","commit_sha":"7256a4203a5567687fec14d95afaf3bca17d0d40"}
