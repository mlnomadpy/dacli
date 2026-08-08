---
id: f-074-complete-on-branch-dacli-074-fix-analytics-all-4-acceptance-met-build-test
kind: note
note_kind: finding
created: 2026-07-22T19:27:04Z
created_by: a-5gg79k0bcq
about: [[074]]
severity: moderate
---
# 074 complete on branch dacli/074-fix-analytics — all 4 acceptance met, build+test green
Commit c071180 by a-5gg79k0bcq on branch dacli/074-fix-analytics-estimate-token-unit-calibrate-one-authoritative-line-doctor-brief. (1) cmdEstimate (insight.go:386-425) now PREFERS token-per-point: collects the band's HasTokens() TokenRatio() samples and, when n>=10, prints 'tok/point ... THIS is the estimate' with wall-clock demoted to the fallback (n>=10 wall-clock only when no token history) — same preference calibrate prints. (2a) cmdCalibrate computes tokenByAgent FIRST and a tokenAuthoritative set; the wall-clock agent-band loop (insight.go:737-749) demotes any token-authoritative band to '(fallback — tokens/point below IS the estimate)', so exactly ONE '← AUTHORITATIVE / IS the estimate' line prints per band instead of two contradictory ones. (2b) cmdDoctor's finding query gained Pending:true (insight.go:895) so an applied (synced) EventFinding is not counted as BOTH event and note — same !e.Applied dedup contrib uses. (3) brief.go 'What siblings found' now scopes pending finding EVENTS to the project (inProject set from ListTasks(w,p.Slug)) matching the per-project finding NOTES, removing the sync-boundary visibility flip (issue #21); documented in the section comment and decision note. New internal/brief/brief_test.go proves same-project pending events show and cross-project ones do not. go build ./... clean; go test ./internal/... all green. Binary smoke of calibrate/doctor blocked by headless sandbox (exec needs approval), verified via build+unit tests. Owner: verify and close via dacli task check 074 --n 1..4 / task done, then dacli merge --task 074.
