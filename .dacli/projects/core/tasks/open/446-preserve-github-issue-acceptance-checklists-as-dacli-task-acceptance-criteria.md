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
- [ ] Pull recognizes a documented GitHub issue acceptance heading and imports its checklist into the task's canonical Acceptance section.
- [ ] Imported checked/unchecked state has an explicit policy and regression coverage.
- [ ] The original issue body remains available as context without creating two independently editable acceptance sources.
- [ ] Issues without a recognized acceptance checklist retain current behavior and do not gain invented criteria.
- [ ] Re-pull/idempotent adoption does not duplicate acceptance boxes.
- [ ] A regression test proves the adopted task can be checked and accepted without `--allow-unverified`.
## Log
