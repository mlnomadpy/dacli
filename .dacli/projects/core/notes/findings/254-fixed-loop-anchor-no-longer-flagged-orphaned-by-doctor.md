---
id: f-254-fixed-loop-anchor-no-longer-flagged-orphaned-by-doctor
kind: note
note_kind: finding
created: 2026-08-04T11:47:20Z
created_by: a-maintainer-as4r9s
about: "[[254]]"
severity: moderate
---
# 254 fixed: loop anchor no longer flagged orphaned by doctor
Root cause: the standing continuous-improvement anchor (e.g. task 244) is owned by 'loop' (ensureImproveTask, orchestration.go:1223). doctor's orphan check (insight.go) flagged any open/active task whose owner is non-root with no live run — and 'loop' is never a live process, so the anchor was reported orphaned on every run. Fix: added '&& !t.IsLoopAnchor()' to the orphan condition (insight.go:~989), reusing the shared predicate from decision 112. Regression test TestDoctorSkipsLoopAnchor (insight_test.go) fails before the change (flags '001-continuous-improvement...(owner loop)') and passes after — verified via git stash. go build/vet clean, gofmt clean, go test ./... green except the pre-existing DACLI_AGENT-leak catalog test (green with 'go test -exec env -u DACLI_AGENT'). Branch: dacli/254-doctor-reports-the-loop-anchor-as-orphaned-on-every-run, commit 2b9bde5. Acceptance 'the standing review anchor is not reported as an orphaned task' is met; owner runs 'dacli accept 254'.
