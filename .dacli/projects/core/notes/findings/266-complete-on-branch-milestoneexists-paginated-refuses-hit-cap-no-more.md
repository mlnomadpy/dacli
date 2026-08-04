---
id: f-266-complete-on-branch-milestoneexists-paginated-refuses-hit-cap-no-more
kind: note
note_kind: finding
created: 2026-08-04T14:43:09Z
created_by: a-maintainer-eswwm8
about: "[[266]]"
severity: major
---
# 266 complete on branch: milestoneExists paginated + refuses hit-cap, no more duplicate milestones past page 1
Commit 26a47c1 (a-maintainer-eswwm8) on branch dacli/266-milestoneexists-reads-only-the-first-page-of-milestones-so-ensuremilestone. Root cause: ghmirror.go milestoneExists called 'gh api repos/O/R/milestones?state=all' which returns only the DEFAULT first page (30 items), so a repo past 30 milestones never found an existing one on a later page and ensureMilestone POSTed a duplicate every push (matches ghmirror milestone list unpaginated finding, task 224 lineage). Fix: (1) milestoneExists now requests ?per_page=100 (endpoint max) and returns (bool,error); a positive find is definitive at any length, but a list landing exactly on the cap (100) WITHOUT the title errors instead of reporting a false absent — mirroring fetchAllIssues/listIssues (dacli 205) cap+refuse convention. new const ghMilestoneListLimit=100 (ghmirror.go:1030). (2) ensureMilestone refuses (returns false, creates nothing) when the existence check errors, rather than POSTing a milestone that may already exist on an unread page. ACCEPTANCE: [1] list is capped like every other list read and a hit-cap is refused — done; [2] TestEnsureMilestoneFindsTargetPastFirstPage drives 60 milestones (target at #45, past the old 30 page) and asserts 0 POSTs; TestEnsureMilestoneRefusesAtListCap drives exactly 100 non-matching and asserts error + 0 POSTs. VERIFIED: reproduced the bug by reverting only the URL to the page-1-only form — TestEnsureMilestoneFindsTargetPastFirstPage fails ('did not request a page beyond the default 30'); restored, all green. go build ./... ok, go test ./... all packages ok, go vet ./... clean, gofmt -l internal/ empty. Owner: dacli accept 266 then integrate/merge --task 266.
