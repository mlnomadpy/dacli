---
id: f-053-complete-on-branch-dacli-053-commit-d6092ed-all-4-acceptance-met
kind: note
note_kind: finding
created: 2026-07-22T16:36:37Z
created_by: a-sfa41hsara
about: [[053]]
severity: moderate
---
# 053 complete on branch dacli/053 — commit d6092ed, all 4 acceptance met
Branch dacli/053-fix-slices-init-flags-ghmirror-idempotency-collab-per-question-attribution, commit d6092ed by a-sfa41hsara. Staged the 9 scoped files (--force noted 3 out-of-claim files load-bearing for the fix; see decision note). ACCEPTANCE: (1) init honors --template (validated via gates.Get; recorded as config default_template; workspace.DefaultTemplate parsed in open(); planning cmdProjectAdd falls back to it so a no-flag project add seeds the process) and --roster (validated; seeds real role files from wscore/rosters.go software|research|solo via store.CreateRole); unknown values now REFUSE (exit 2) instead of silently exiting 0. (2) collab answer event's About now points at the QUESTION id not the task (collab.go:143), and cmdThreads keys the answered map + lookup by q.ID (collab.go:190) so two questions on one task answered by different agents attribute correctly; selfreport gh calls go through ghOutput() = exec.CommandContext + 120s WithTimeout (selfreport.go), mirroring ghmirror, so a hung gh cannot block dacli/mcp serve. (3) ghmirror searchByMarker now reads issue bodies via the strongly-consistent 'gh issue list --json number,body' and matches the marker by exact Go substring — NOT the eventually-consistent, tokenized --search index — so crash-recovery adoption converges on the first retry with zero duplicates; package + fn docstrings updated; ghmirror_test.go fixture updated to the new contract (TestGithubPushIdempotent green). (4) governance package docstring narrowed to the one remaining stub (shortcut promote), naming stagegate/ghmirror/skillforge/verify.go as shipped. go build ./... clean; go vet clean; go test ./internal/... all green incl. internal/cli (TestFeatureSlicesAreIsolated, ghmirror tests) and internal/workspace. Owner: verify and close via dacli task check/done + dacli merge --task 053.
