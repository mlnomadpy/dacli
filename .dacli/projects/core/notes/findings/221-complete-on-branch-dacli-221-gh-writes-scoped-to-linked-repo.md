---
id: f-221-complete-on-branch-dacli-221-gh-writes-scoped-to-linked-repo
kind: note
note_kind: finding
created: 2026-08-04T11:55:11Z
created_by: a-maintainer-nyj8xr
about: "[[221]]"
severity: moderate
---
# 221 complete on branch dacli/221-...: gh writes scoped to linked repo
Committed 19852f1 by a-maintainer-nyj8xr on branch dacli/221-gh-calls-never-pass-repo-so-writes-can-target-the-wrong-repository. Acceptance ('gh writes target the stored repo explicitly') SATISFIED: added ghRepo(w,repo,args...) choke point in internal/features/ghmirror/ghmirror.go that APPENDS --repo <github_repo> after the subcommand verb (gh's --repo is a per-command flag, invalid at root; same placement as selfreport.go:116/catalog.go:351) and delegates to the stubbable gh var. Threaded the linked repo through every issue/label call: issue create/edit/close/comment/view, issue list (fetchAllIssues/listIssues/markerIndex), label create (ensureLabel/precreateLabels), applyStatusLabel/applyTaskLabels/applyFindingLabels, mirrorFindings, mirrorDecisions, mirrorFindingIssues. disclosureGate/repoView now judge the LINKED repo's visibility (repo param), mirroring catalog's dacli-167 fix; empty repo falls back to cwd so github doctor/link discovery still works. auth status stays bare; the gh project surface was already scoped via --owner/--url. Proof: go build ./... + go vet ./... clean, gofmt -l clean, go test ./... all 41 pkgs green (env -u DACLI_AGENT; the one catalog failure without it is the known DACLI_AGENT test-leak, f-016, not this change). New reposcope_test.go pins --repo onto the helper surface; VERIFIED it fails-before by temporarily removing ghRepo's append (repro: 'ghRepo dropped --repo octo/linked') and passes-after restoring. PR-first off: owner runs dacli accept 221 then merge/integrate --task 221.
