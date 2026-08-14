---
id: t-01KZYREKZGV6Z0CG3GM5J4G5BG
kind: task
created: 2026-08-13T23:48:51Z
created_by: a-root
owner: a-root
github:
  issue: 652
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Preserve GitHub issue acceptance checklists as dacli task acceptance criteria on pull
## Context
Adopted from GitHub issue #652.

Implementation and claim boundary: `internal/features/ghmirror` owns GitHub issue-body parsing, inbound adoption, idempotency, and its regression tests. Reuse the canonical task acceptance representation exposed by `internal/store` without changing store semantics or unrelated outbound mirroring.

## Symptom

`dacli github pull <project>` adopts a GitHub issue whose body contains an `## Acceptance criteria` checklist, but places the entire issue body under the task Context and leaves the task's canonical `## Acceptance` section empty.

Observed on issue #650 → task 440:

- The ten `- [ ]` criteria appear under `## Context` / `## Acceptance criteria`.
- The canonical `## Acceptance` heading contains zero boxes.
- `dacli accept 440` therefore refuses because the task has no acceptance criteria unless the owner uses the explicitly unverified escape hatch.

## Suspected cause

`issueContext` copies the issue body verbatim into `TaskOpts.Context`, while the pull adoption path never extracts a recognized acceptance heading/checklist into `TaskOpts.Accept`.

## Manual workaround

The owner must manually copy the existing checklist into the canonical `## Acceptance` section before claiming/checking/accepting the task.

## Acceptance criteria

- [ ] Pull recognizes a documented GitHub issue acceptance heading and imports its checklist into the task's canonical Acceptance section.
- [ ] Imported checked/unchecked state has an explicit policy and regression coverage.
- [ ] The original issue body remains available as context without creating two independently editable acceptance sources.
- [ ] Issues without a recognized acceptance checklist retain current behavior and do not gain invented criteria.
- [ ] Re-pull/idempotent adoption does not duplicate acceptance boxes.
- [ ] A regression test proves the adopted task can be checked and accepted without `--allow-unverified`.

## Acceptance
- [x] Pull recognizes a documented GitHub issue acceptance heading and imports its checklist into the task's canonical Acceptance section.
- [x] Imported checked/unchecked state has an explicit policy and regression coverage.
- [x] The original issue body remains available as context without creating two independently editable acceptance sources.
- [x] Issues without a recognized acceptance checklist retain current behavior and do not gain invented criteria.
- [x] Re-pull/idempotent adoption does not duplicate acceptance boxes.
- [x] A regression test proves the adopted task can be checked and accepted without `--allow-unverified`.
## Log
- 2026-08-14T01:22:52Z claimed by a-maintainer-q52j1d
- 2026-08-14T08:59:18Z accepted by a-root
- 2026-08-14T08:59:18Z verified by `GOCACHE=/tmp/dacli-accept-446 go test ./internal/features/ghmirror` (exit 0) in branch main at 0c1f80a — proves that tree builds, not that the work is in trunk
- 2026-08-14T08:59:18Z deliverable: dacli/446-preserve-github-issue-acceptance-checklists-as-dacli-task-acceptance-criteria is merged into main
- 2026-08-14T08:59:18Z completed by a-root
- 2026-08-14T09:11:42Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/666 (event 01KZZQHAKE9F12Z14KKHT084CP)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-accept-446 go test ./internal/features/ghmirror","exit_code":0,"duration_ms":7153,"artifact_hash":"sha256:39fecab17b64ff95b3289d958bb90a3ec3c9907940b9fdf252f3dca09d4d5656","verifier":"a-root","branch":"main","commit_sha":"0c1f80acacb6f131acfa7505228a7c0e653cc764"}
