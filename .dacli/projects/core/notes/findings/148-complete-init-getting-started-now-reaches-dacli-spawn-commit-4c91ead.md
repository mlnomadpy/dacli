---
id: f-148-complete-init-getting-started-now-reaches-dacli-spawn-commit-4c91ead
kind: note
note_kind: finding
created: 2026-07-26T15:54:37Z
created_by: a-4t2ys9fqpn
about: [[148]]
severity: minor
---
# 148 complete: init getting-started now reaches dacli spawn, commit 4c91ead
Committed 4c91ead by a-4t2ys9fqpn (fixer). Staged only the 2 intended files: internal/features/wscore/wscore.go, internal/cli/overview_test.go. ACCEPTANCE: (1) printGettingStarted (wscore.go:87-97) previously dead-ended at whoami/project add/task add/next/overview with no path to a first agent run (per finding f-init-getting-started-never-surfaces-dashboard-or-spawn). Now inserts two concrete, copy-pasteable steps between 'next' and 'overview': 'dacli runtime add claude-code --preset claude-code' (connect the coding-agent CLI) and 'dacli spawn --task <ref> --runtime claude-code --grant ro' (launch the first agent), matching the claude-code preset already shipped in execution.go:57-72. (2) Human-facing only: printGettingStarted is called from cmdInit only when !ctx.JSON (wscore.go:75-77, unchanged); TestInitJSONSkipsGettingStarted (internal/cli/overview_test.go) still asserts --json omits the section — no regression. (3) Test coverage: extended internal/cli/overview_test.go TestInitPrintsGettingStarted to assert 'dacli runtime add' and 'dacli spawn --task' both appear, and that runtime add precedes spawn in output order (an adopter following the list top-to-bottom needs a runtime configured before spawn can find one). go build ./... clean; go test ./... all green. NOTE: dacli adopt (internal/features/onboard/onboard.go:151) has a separate, unrelated dead-end ('next: dacli context <task>' / 'dacli lint', no spawn mention) — out of scope, task 148's acceptance names wscore.go specifically; left as a lead for a future finding if the owner wants it covered too. Owner: verify and close via dacli task check/done + dacli merge --task 148.
