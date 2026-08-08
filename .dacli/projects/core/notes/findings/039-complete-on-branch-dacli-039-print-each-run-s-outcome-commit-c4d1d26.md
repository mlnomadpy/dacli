---
id: f-039-complete-on-branch-dacli-039-print-each-run-s-outcome-commit-c4d1d26
kind: note
note_kind: finding
created: 2026-07-22T15:31:58Z
created_by: a-87qa0eamp1
about: [[039]]
severity: moderate
---
# 039 complete on branch dacli/039-...-print-each-run-s-outcome; commit c4d1d26
Commit c4d1d26 by a-87qa0eamp1 on branch dacli/039-dacli-wait-streams-completions-print-each-run-s-outcome-as-it-finishes-not. Staged ONLY internal/features/execution/execution.go (git add + dacli commit). cmdWait UX polish per brief: (1) startup line 'waiting on N run(s): <sorted short ids>'; (2) per-completion line now streams the moment each child is detected gone with '(N of M)' progress via finalizeRun (existing streaming kept exactly); (3) light heartbeat 'still waiting on K run(s) (up <elapsed>)' every ~30s via a nextBeat timer, NOT every poll. Both acceptance criteria met: per-completion streaming + build/test green. go build ./... clean; go test ./internal/... all green. Box-checking refused for non-owner (a-root); done recorded as a propose event — owner closes via dacli accept 039 or task done, then dacli merge --task 039.
