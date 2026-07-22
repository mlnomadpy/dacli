---
id: t-01KY4ZWW4GQWW38EHMJMVAPNA9
kind: task
created: 2026-07-22T13:23:01Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 4, probable: 6, pessimistic: 9}
---
# D3: trust as a first-class brief property — trust-floor label + spawn-time taint gate
## Context
D3 makes TRUST visible at the point of consumption. As trees scale, the bottleneck shifts from "does the child have context" (solved) to "can I trust its report". Three concrete moves; the trust-floor label is the priority.

Anchors:
- **Persist verify grades onto the finding.** `internal/features/execution/verify.go` `cmdVerify` (~:34) derives a confirmed/refuted verdict per claim but writes it only to the run outcome (:140). Make it also stamp the graded finding NOTE with a `trust:` frontmatter value (`confirmed` | `refuted`). Finding notes are written via `store.CreateNote`; add a small `store.GradeFinding(w, project, findingID, trust)` (or set the `trust:` front key on the note doc and re-save) and call it from verify for the finding it judged (`latestFinding`/`--claim` identifies it). A finding with no `trust:` key is **ungraded**.
- **Trust-floor label on the brief.** `internal/brief/brief.go` §8 "What siblings found" (~:205-233) surfaces finding notes + pending finding events. Label each surfaced finding with its trust grade, and add a **trust-floor** line to the brief = the WORST grade among surfaced findings, ordered refuted < ungraded < confirmed. An ungraded finding lowers the floor to "unverified" — that is the point: an unverified claim is visible as such BEFORE a child acts on it.
- **Spawn-time taint gate.** `internal/features/execution/execution.go` `cmdSpawn` — D2 already DISPLAYS taint status under `--advise`. D3 makes it a GATE: if the task sits in a taint blast radius (reuse the same taint check), REFUSE the spawn (clikit.Refusedf, exit 3) unless `--force` or `--cooperative` is passed. A refused spawn on a poisoned task is the whole point of taint-at-spawn.

## Scope (STRICT) — touch ONLY:
- `internal/store/store.go` (the GradeFinding helper / trust key)
- `internal/brief/brief.go`
- `internal/features/execution/verify.go`
- `internal/features/execution/execution.go` (the spawn taint gate)

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the files above plus this task's file. `go build ./...` + `go test ./internal/...` green — the brief has tests; the trust-floor line must not break them (extend if needed). `dacli note add finding` with the summary, then `dacli commit`. Box-checking is owner-only — file a completion finding, do not retry a refused check.

## Acceptance
- [x] a brief carries a trust-floor label derived from verify grades of the findings it surfaces
- [x] verify grades a finding BEFORE it enters siblings briefs, not after
- [x] taint becomes a spawn-time gate (refuse/warn on a tainted task), not only an audit query
- [x] committed on branch by an agent; build + test green
## Log
- 2026-07-22T13:54:50Z completed by a-root
