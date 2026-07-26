---
id: f-143-complete-on-branch-dacli-143-commit-ff2fabe-all-3-acceptance-criteria-met
kind: note
note_kind: finding
created: 2026-07-26T20:47:02Z
created_by: a-x7vfysmvvg
about: [[143]]
severity: moderate
---
# 143: complete on branch dacli/143 — commit ff2fabe, all 3 acceptance criteria met
Committed ff2fabe by a-x7vfysmvvg (fixer). Staged 5 files: internal/clikit/{clikit.go,clikit_test.go}, internal/features/planning/{planning.go,planning_test.go}, internal/features/knowledge/knowledge.go. ACCEPTANCE: (1) clikit.go adds Flags.Reject(known ...string) error (clikit.go, after Raw()) -- iterates f.vals, returns Usagef (exit 2) naming every unknown flag, sorted for a deterministic message; TestFlagsRejectUnknownFlag and TestFlagsRejectKnownSetPasses in clikit_test.go prove an unknown flag is rejected (exit 2, message names --acccept) and a correct known-set passes. (2) Adopted in cmdTaskAdd (planning.go:106-109, allowlist project/force/priority/estimate/accept/so-that/context/depends-on/parent), cmdTaskCheck (planning.go, allowlist n/all), and cmdNoteAdd (knowledge.go, allowlist project/about/body/rejected/because/severity/scope/origin/against). cmdRun and its Raw()-forwarding path in execution.go were NOT touched. (3) TestTaskAddRejectsTypoedFlag in planning_test.go reproduces the exact regression: 'task add <t> --project p --acccept y' exits 2, creates 0 tasks, and the same command with correctly-spelled --accept succeeds. go build ./... clean; go test ./... all green; gofmt -l . clean. NOTE: task front-matter owner is still a-vh51d10ng9 (my claim landed as an event, not yet synced), so box-checking is refused to me (agentid.CanMutate: not the owner) -- owner should sync the claim event and then check/done + merge --task 143.
