---
id: t-01M0F8JAH5CNJ327M31B1821BF
kind: task
created: 2026-08-20T09:38:20Z
created_by: a-root
owner: a-root
github:
  issue: 763
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Make behavioral preflight capability explicit for persisted Codex adapters
## Context
Adopted from GitHub issue #763.

## Reproduction

After PR #759 / issue #746 landed, this mature workspace ran:

```text
dacli preflight --role maintainer
launch declared/unsupported · adapter declares no behavioral launch strategy
preflight maintainer on codex-rw (rw): no mismatches
```

The role uses the Codex desktop binary and Codex JSONL `exec` invocation, but its persisted runtime predates the shipped preset and has no `usage_format`. `hasBehavioralPreflight` currently enables the launch handshake only when `UsageFormat == "codex-jsonl"`. Therefore an existing Codex adapter silently bypasses the pre-spawn behavioral gate even though its transport is supported.

## Design direction

Represent behavioral-preflight strategy as an explicit adapter capability/version, separate from usage parsing. New presets should persist it, runtime loading should migrate or safely infer known legacy Codex records from an exact invocation contract, and doctor should distinguish unsupported adapters from legacy records requiring migration. Do not use runtime name alone as authority and do not couple launch safety to token-usage parsing.



## Relationship

Follow-up to #746 and PR #759. This defect was found by running the shipped preflight against dacli's own persisted `codex-rw` adapter before loop cycle 109.

## Acceptance
- [ ] Runtime records and presets declare a versioned behavioral-preflight strategy independently of `usage_format`, runtime name, and provider-output usage parsing.
- [ ] Existing persisted Codex `exec --json` adapters that match the supported invocation contract migrate or resolve to the Codex strategy without destructive remove/recreate and without changing role references.
- [ ] Ambiguous/custom adapters remain unsupported with an actionable migration command; runtime name alone never enables provider-specific execution.
- [ ] `runtime doctor`, standalone `preflight`, and `spawn` resolve the same effective strategy for the same persisted adapter.
- [ ] A mature-workspace fixture without `usage_format` reproduces `declared/unsupported` before the fix and performs the bounded Codex handshake after migration/resolution.
- [ ] Cache fingerprints include the effective strategy/version so evidence from an earlier adapter contract cannot authorize a changed one.
- [ ] Human and JSON output expose declared, inferred/migrated, and probed provenance without credentials, prompt content, or environment values.
- [ ] Mutation proof shows restoring the `usage_format` equality gate makes the mature-workspace regression fail.
- [ ] `gofmt -l .`, `go vet ./...`, pinned `golangci-lint run`, and `go test ./...` pass.
## Log
- 2026-08-20T09:38:44Z claimed by a-maintainer-1ckxmn
- 2026-08-20T09:49:49Z a-verifier-74g40a: verify-verdict: no-verdict — codex-ro (a-verifier-74g40a) on claim: Task 492 implementation satisfies all acceptance criteria — panelist reported nothing — counts as unconfirmed (event 01M0F95TYF0ANEM44TYRAK0M21)
- 2026-08-20T09:49:49Z a-verifier-qx6x8x: verify-verdict: no-verdict — cc (a-verifier-qx6x8x) on claim: Task 492 implementation satisfies all acceptance criteria — panelist reported nothing — counts as unconfirmed (event 01M0F95VH3W6J9C8SKBP4ZRKZR)
- 2026-08-20T09:49:49Z a-verifier-n0hxqh: verify-verdict: no-verdict — cc-rw (a-verifier-n0hxqh) on claim: Commit 0881d1a decouples behavioral preflight from usage parsing and satisfies task 492 acceptance — panelist reported nothing — counts as unconfirmed (event 01M0F96H9R2YCQQG7QMN40EKSA)
