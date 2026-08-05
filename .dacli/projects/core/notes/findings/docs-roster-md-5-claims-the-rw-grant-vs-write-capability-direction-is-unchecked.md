---
id: f-docs-roster-md-5-claims-the-rw-grant-vs-write-capability-direction-is-unchecked
kind: note
note_kind: finding
created: 2026-08-05T13:56:52Z
created_by: a-fixer-p5ee58
about: "[[272]]"
severity: minor
---
# docs/ROSTER.md:5 claims the rw grant-vs-write-capability direction is unchecked, but dacli 250 already checks it
docs/ROSTER.md:5 reads: 'The rw direction is not checked: an rw role spawns on any runtime, including one whose allowlist has no Edit or Write, and the mismatch only surfaces as an agent that burns its run unable to change a file.' That was accurate before dacli 250 landed, but sandboxFor (internal/features/execution/execution.go) has refused exactly this case since then — grant != ro && !store.RuntimeWritable(rt) && !cooperative returns clikit.Refusedf('runtime %s grants no write tool...'), exit 3 — and dacli 272 (this task) now surfaces the same check as a warning ahead of that refusal too. Found while implementing 272's preflight and cross-checking every doc claim about the grant/runtime coupling for consistency; left unfixed because it is a pre-existing drift from an earlier task (250), not something 272 touched or introduced, and editing docs/ROSTER.md is outside 272's acceptance criteria.
