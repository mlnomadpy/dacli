---
id: f-141-dacli-already-ships-a-code-backed-human-approve-gate-reviewer-proposes-box
kind: note
note_kind: finding
created: 2026-07-24T09:34:46Z
created_by: a-f5x84xzytt
about: [[141]]
severity: moderate
---
# 141: dacli already ships a code-backed human approve-gate — reviewer proposes box-checks, owner-only accept applies them
The reviewer agent (ro grant, .dacli/roles/reviewer.md) cannot close its own task: dacli propose() records a box-check PROPOSAL as an event (internal/features/acceptance/acceptance.go:91-104) and only the owner's dacli accept applies it (acceptance.go:106-126); RUNTIMES.md:456 states box-checking/task-closing are owner-only. So H6 (approve-gate a PR from the UI) is not a new gate — it is a UI affordance for a gate that already exists in the CLI. The research verdict: the gate adds value on weak/contested/irreversible outcomes and is pure friction (a bottleneck, per INTERVIEW_GUIDE §5) on clean 3-0 panels over reversible changes, where --auto/integrator autonomous landing (GITHUB.md §9.4) already degrades the human to veto. Argues for a conditional gate, not a uniform one.
