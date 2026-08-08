---
id: f-048-complete-on-branch-dacli-048-commit-4f26d6c-all-acceptance-met
kind: note
note_kind: finding
created: 2026-07-22T16:26:39Z
created_by: a-n2q5ysnx5y
about: [[048]]
severity: moderate
---
# 048 complete on branch dacli/048 — commit 4f26d6c, all acceptance met
Committed 4f26d6c by a-n2q5ysnx5y. Staged ONLY the 4 store files (git add + dacli commit --no-add): internal/store/{store.go,calibration.go,taint.go,lessons.go}. ACCEPTANCE, all satisfied:
(1) SLUGIFY: store.go Slugify captures orig at entry and, when the [a-z0-9] slug collapses to '' (punctuation-only '???' or non-ASCII/CJK/Arabic i18n titles), falls back to a stable deterministic token 'u<fnv32a(orig hex)>' — so CreateNote no longer writes a hidden '.md' with a bare 'f-'/'d-' id and CreateTask no longer writes 'NNN-.md'; distinct titles get distinct slugs, ascii unchanged ('Hello World'->'hello-world'). Verified by throwaway test (removed).
(2) GRADEFINDING determinism: store.go GradeFinding now prefers an exact id match (unique -> targets the intended note regardless of os.ReadDir order); only if no id matches does it fall back to the level-1 title, and it grades ALL same-titled twins (deterministic, order-independent) rather than stamping whichever the FS listed first. LOGSPAN: calibration.go logSpan now measures the FINAL claim->completion cycle (running claim paired with each completion) instead of first-claim/last-completion, so a completed->reopened->re-claimed->re-completed task no longer inflates the wall-clock actual across the idle gap. Verified by throwaway test (removed).
(3) TAINT perf: taint.go builds ONE store.BuildTaskIndex up front and canonRef resolves through it (idx.Find, O(1)/hit) instead of FindTask-per-hit (O(hits x tasks)). TASKBAND: calibration.go TaskBand walks run dirs NEWEST-first (ULID order) via new readRunBand helper and stops at the first run naming the task — no longer builds the whole runBands map per task; newest-first preserves runBands' 'last completing run wins'. runBands refactored onto the same helper.
(4) TAINT metric over-report: lessons.go now exposes lessonKinds{Decision,Finding,Ref}+SurfacesAsLesson(kind); WorkspaceLessons and taint.go both read it. taint.go marks TreeWide only when scope==workspace AND SurfacesAsLesson(kind), so a scope:workspace METRIC (which WorkspaceLessons never surfaces) is no longer falsely reported as reaching every project's brief. The two files now agree.
(5) go build ./... clean; go test ./internal/... all 39 pkgs green incl. internal/store, internal/cli, TestFeatureSlicesAreIsolated; gofmt clean; go vet clean.
Owner: verify and close via dacli task check/done + dacli merge --task 048.
