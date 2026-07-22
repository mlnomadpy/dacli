---
id: f-g4-bidirectional-github-loop-complete-github-pull-adopts-human-issues-as-tasks
kind: note
note_kind: finding
created: 2026-07-22T17:35:37Z
created_by: a-sdpxn53045
about: [[057]]
severity: moderate
---
# G4 bidirectional GitHub loop complete: github pull adopts human issues as tasks + findings backlink to issue comments
Branch dacli/057-g4-bidirectional-... in internal/features/ghmirror/ghmirror.go. INBOUND: 'github pull <project>' (cmdPull) lists issues via the strongly-consistent list endpoint and adopts each human-authored one (shouldImport skips issues carrying a dacli marker AND issues already mapped to a local task) as a task via store.CreateTask, seeding title + Context (issue backlink+body) and writing the github: issue/repo block back so a re-pull/push treats it as linked. Idempotency is number-mapping (mappedIssues), not a body marker, since pull never edits the remote. FINDINGS->COMMENTS: mirrorFindings() in cmdPush posts each finding note about the task (findingAboutTask) as an issue comment, idempotent via a per-finding marker (findingMarker, distinct prefix from task/decision) checked against existing comments (issueComments/commentsHaveMarker) so a re-push never duplicates. github sync = pull then push. Disclosure gate factored into disclosureGate(), shared by push + finding comments (the risk-rank-2 leak surface); pull is inbound so not gated (decision recorded). NOTE: updated internal/cli/supervise_test.go (OUT of STRICT scope, --force) because it asserted github sync was a planned stub — now stale; repointed at 'shortcut promote' (still planned). No live gh in tests: marker/idempotency/skip logic unit-tested on fixtures. go build ./... clean; go test ./internal/... green; vet+fmt clean.
