---
id: t-01M146B9YAADZ9X23NQ9EFE55N
kind: task
created: 2026-08-28T12:43:36Z
created_by: a-root
owner: a-root
github:
  issue: 860
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Route monorepo verification by changed paths and working directory
## Context
Adopted from GitHub issue #860.

## Parent

Part of #855.

## Observed symptom

Repository-profile inference emits generic root commands by detected language. On dacli itself, the resolved loop profile includes root `npm test` and `npm run build`, while the actual `package.json` is under `internal/features/dashboard/ui/`. A valid project policy can therefore be unusable or run unrelated gates.

## Objective

Add declarative, path-aware verification routing for multi-language monorepos and execute only the gates required by the claimed/diffed change surface.

## Proposed contract

Each verification rule should define stable structured fields such as:

- path include/exclude matchers;
- working directory;
- argv, without nested shell-string quoting;
- environment policy;
- gate kind and whether it can fan out;
- dependency/contract groups whose changes trigger multiple language gates.

Explicit repository configuration must take precedence over inference.



## Non-goals

- Inventing project-specific commands without evidence.
- Making every detected language mandatory for every task.
- Replacing CI workflow configuration.

## Manual workaround today

Operators hand-edit resolved verification commands or run each subsystem's gates outside dacli.

## Acceptance
- [ ] A fixture with Go root, nested web package, Python service, docs, and shared contract maps each changed path set to the expected cwd/argv gates.
- [ ] A shared-contract change triggers every declared dependent language gate; a docs-only change does not run unrelated builds.
- [ ] The dacli repository profile resolves dashboard commands with cwd `internal/features/dashboard/ui`, never the repository root.
- [ ] Commands are represented as structured argv plus cwd and execute paths containing spaces without shell-quoting ambiguity.
- [ ] Explicit configured rules remain byte-for-byte stable across inspect/configure/start; inference fills only missing policy.
- [ ] Empty, conflicting, or unmatched required rules fail with actionable diagnostics rather than silently falling back to an unrelated root command.
- [ ] Regression tests fail when cwd/path selection is removed, and the repository-wide quality gates pass.
## Log
