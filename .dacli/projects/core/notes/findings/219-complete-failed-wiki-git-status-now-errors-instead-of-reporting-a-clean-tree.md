---
id: f-219-complete-failed-wiki-git-status-now-errors-instead-of-reporting-a-clean-tree
kind: note
note_kind: finding
created: 2026-08-04T11:40:24Z
created_by: a-maintainer-f89wdf
about: "[[219]]"
severity: major
---
# 219 complete: failed wiki git status now errors instead of reporting a clean tree
Branch dacli/219-catalog-wiki-publish-reports-success-when-git-status-fails, commit 8a4295e. Root cause: internal/features/catalog/catalog.go:303 ran 'out, _ := git(w, tmp, "status", "--porcelain")' discarding the error; git prints nothing to stdout on failure, so strings.TrimSpace(out)=="" read a FAILED status as a clean tree and printed 'wiki already up to date' + returned nil, skipping commit/push — a never-pushed wiki falsely reported up to date. Fix: added wikiClean(out,err) helper (catalog.go, next to git helper) that returns an error when err!=nil, else clean=(trimmed out==""); call site now 'clean, err := wikiClean(git(...)); if err != nil { return err }; if clean {...}'. Test TestWikiCleanFailedStatusIsError in catalog_test.go pins the contract (err->error, empty+no-err->clean, non-empty->dirty). REPRO VERIFIED: a throwaway oldWikiClean('',err) returns clean=true/err=nil (false success) while new wikiClean('',err) returns clean=false/err!=nil — same input, opposite outcome. Proof: go build ./... clean; go vet ./... clean; gofmt -l internal/ clean; go test -exec 'env -u DACLI_AGENT' ./internal/features/catalog/ all green incl new test. NOTE: plain 'go test ./...' shows catalog's TestCatalogRefusesRatherThanWritingAnEmptyRoster fail with 'agent token not recognized' — the known DACLI_AGENT leak (this session runs as a dacli agent; catalog package has no TestMain clearing it, unlike internal/cli), NOT a regression; green with the var stripped. Acceptance box 1 ('a failed git status is an error not an empty tree') satisfied — owner to check + accept.
