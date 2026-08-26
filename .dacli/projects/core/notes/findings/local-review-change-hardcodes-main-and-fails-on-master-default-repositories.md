---
id: f-local-review-change-hardcodes-main-and-fails-on-master-default-repositories
kind: note
note_kind: finding
created: 2026-08-26T13:22:55Z
created_by: a-adversarial-reviewer-f9a8a9
about: "[[t-01KZXRP56538HNHDP3P4FJHGGP]]"
severity: major
---
# Local review change hardcodes main and fails on master-default repositories
Queued task 481 already owns this local-review change, so no distinct task was filed after checking core open and active work. In commit a18e8e1068d240c18ce3d9ba0d8f02ed4b2fc6a8, internal/prompts/tpl/review_workflow.md:11 instructs git diff main...<task-branch>. Trigger: a valid repository whose default/trunk branch is master and which has the canonical task branch but no main ref. Observed wrong outcome: the instructed command exits 128 with fatal ambiguous argument, so the reviewer cannot read the actual diff; internal/store/landing.go:188-201 already resolves origin/HEAD then main/master. Request changes to task 481: render the resolved trunk ref into the local review prompt and cover a master-only repository. Reproduction: initialize a repo with git init -b master, commit base, commit a dacli/* branch, then git diff main...dacli/*; exit 128. Focused current-main execution tests passed with GOCACHE=/private/tmp/dacli-go-cache, but reported no matching tests because task 481 is queued and not present on main.
