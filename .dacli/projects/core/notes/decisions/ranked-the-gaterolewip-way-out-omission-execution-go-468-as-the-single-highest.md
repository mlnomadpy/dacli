---
id: d-ranked-the-gaterolewip-way-out-omission-execution-go-468-as-the-single-highest
kind: note
note_kind: decision
created: 2026-08-04T16:27:41Z
created_by: a-go-auditor-z48ata
about: "[[277]]"
---
# Ranked the gateRoleWIP way-out omission (execution.go:468) as the single highest-value refusal-audit finding
## Chose
Ranked the gateRoleWIP way-out omission (execution.go:468) as the single highest-value refusal-audit finding
## Rejected
Filing the rw-grant class (~12 sites naming the requirement but not the recourse), or the acceptance.go --require-verify pair, as the primary
## Because
The rw-grant class is uniform and arguably by-design: an agent's grant is fixed at spawn, so 'the way out' (ask root / have an rw agent run it) is understood, and changing 12 sites is a style sweep not a defect fix. acceptance.go:178/341 already says 'no --verify command was given', which half-names the fix. The WIP case is the sharp one: two refusals fire on the IDENTICAL condition (ActiveInRole >= role.WIP) yet only the human-facing team-assign path (teamops.go:63) names the remedy ('dacli agent retire one, or raise wip'), while the agent-facing spawn/supervise gate (gateRoleWIP, execution.go:468 — the MORE-traveled path per its own doc comment) is a bare dead end. Inconsistency-between-neighbours is the strongest evidence class, the fix is a one-line copy of the sibling's suffix, and it is unit-verifiable by asserting both message strings carry the remedy.
