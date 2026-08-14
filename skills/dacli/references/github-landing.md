# GitHub collaboration and landing

## Contents

- Source-of-truth model
- Linking and diagnostics
- Issue synchronization
- Pull requests and reviews
- Integration and shipping
- Releases and GitHub Apps

## Source-of-truth model

Use GitHub as the shared human collaboration surface and dacli as the detailed
execution/evidence record. Keep them synchronized instead of allowing two
independent backlogs:

- GitHub issues expose work to humans and external contributors.
- dacli tasks carry scheduling, dependencies, acceptance, ownership, briefs,
  events, and verification.
- PRs and checks govern landing.
- Marker-backed synchronization preserves identity and idempotency.

When the operator declares GitHub the main collaboration source, create or
confirm the issue first, then adopt it with `github pull`. Still record agent
execution and acceptance through dacli.

## Linking and diagnostics

```bash
dacli github link <project>
dacli github doctor
dacli project show <project>
```

Public repositories require explicit disclosure consent. Do not disclose
workspace notes, transcripts, findings, or private paths merely because a repo
is linked.

The shipped integration uses the authenticated `gh` CLI. A GitHub App is a
future installation-scoped control-plane option, not required for local or
single-maintainer operation. Prefer an App when centralized installations,
webhooks, organization policy, or service-mode credentials become requirements.

## Issue synchronization

Preview every outbound or broad inbound mutation:

```bash
dacli github pull <project> --dry-run
dacli github push <project> --dry-run
dacli github sync <project> --dry-run
```

Then apply the reviewed direction:

```bash
dacli github pull <project>
dacli github push <project>
dacli github sync <project>
dacli github project <project> --dry-run
```

`pull` adopts human-authored issues as tasks. `push` mirrors tasks and selected
records outward. `sync` pulls then pushes. Review dry-run counts carefully:
task-windowed operations may still include project-level decisions/findings,
depending on flags and policy.

Check for semantic duplicates before creating an issue. An issue can be closed
but cannot be unpublished. Include:

- Observed symptom or user outcome.
- Reproduction/evidence.
- Suspected cause, labeled as a hypothesis.
- Manual workaround.
- Checkable acceptance criteria.
- Explicit non-goals and safety constraints.

## Pull requests and reviews

Let `spawn --worktree` create the canonical `dacli/NNN-slug` task branch when
possible. Branch naming is part of task-to-branch resolution.

```bash
dacli push <ref>
dacli pr --task <ref> --with-verdicts
dacli pr --task <ref> --with-verdicts --auto
dacli pr status --task <ref>
```

Use `--auto` only when repository branch protection and required checks make
GitHub auto-merge trustworthy. Otherwise leave the PR open for review.

Select durable PR landing (`landing.mode: pr`, with `landing.base`) when GitHub
checks, required reviews, and PR discussion are the collaboration boundary.
CLI flags explicitly override project configuration; the legacy default is
local. Effective PR mode fails closed on remote errors. Diagnose with `dacli
github doctor` and `dacli pr status --task <ref>`, then rerun to reuse the
canonical task branch and recorded PR.

Review against the task brief and acceptance criteria. File defects both in
dacli findings and the PR review when humans need to see them. Use
`--approve`/`--request-changes` for real review states, not ambiguous comments.

## Integration and shipping

Use dacli's landing commands because raw `gh pr merge` omits task events,
verdicts, acceptance state, and record commits:

```bash
dacli integrate --tasks <refs> --into main --pr --no-merge
dacli integrate --tasks <refs> --into main --pr
dacli integrate --tasks <refs> --into main --pr --auto
dacli ship --into main --pr --dry-run
```

Run integration from the target branch checkout. Merge overlapping PRs one at
a time and re-check mergeability after each landing. `integrate` requires named
tasks to be done unless an explicit owner override is justified.

`ship` closes a wave by accepting, integrating only the tasks this run closes,
committing the dacli record, and optionally pushing. Use `--tasks` to constrain
the real integration window. The current dry run checks an explicit window
before simulating acceptance and therefore refuses an active task that the real
pipeline can accept and integrate; issue #651 tracks that mismatch. Until it is
fixed, preview the project wave without `--tasks`, inspect proposed tasks, and
then use the explicit task window on the real command.

Wait for required CI before merge. A green local tree does not substitute for
the repository's required checks, and a green CI run does not prove an unmerged
task branch reached trunk. Confirm both.

Close the GitHub issue only after the deliverable is landed and synchronized.

## Releases and GitHub Apps

Preview release engineering:

```bash
dacli github release <project> <tag> --dry-run
```

Creating or pushing a version tag is publication. Do it only with explicit
authority, even when release configuration and snapshot verification are in
scope.

Do not require a GitHub App merely to use dacli. The current `gh` adapter is
appropriate for local-first and maintainer-driven workflows. Consider an App
as a separate control-plane adapter when dacli needs webhook-driven operation,
installation-scoped permissions, organization-wide deployment, multi-user
identity, or hosted automation. Docker changes the execution envelope; it does
not itself provide GitHub identity or project isolation.
