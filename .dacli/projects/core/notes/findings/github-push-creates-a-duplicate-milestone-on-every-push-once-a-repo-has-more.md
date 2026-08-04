---
id: f-github-push-creates-a-duplicate-milestone-on-every-push-once-a-repo-has-more
kind: note
note_kind: finding
created: 2026-08-04T14:04:27Z
created_by: a-root
severity: moderate
origin: internal/features/ghmirror/ghmirror.go:1036
---
# github push creates a duplicate milestone on every push once a repo has more than 30 milestones
Found by a-go-auditor-fqc92q auditing today's landed code (task 264), and hand-verified before filing.

milestoneExists lists via `gh api repos/<repo>/milestones?state=all --jq .[].title`. `gh api` does not auto-paginate and GitHub's milestones endpoint defaults to per_page=30, so only the first page is ever seen. It is the lone uncapped list read in the file — fetchAllIssues uses --limit 1000 and project.go has projectItemListLimit. The inconsistency with its own neighbours is the tell.

Failure, concretely: a linked repo with more than 30 milestones whose project milestone sorts onto page 2.
1. milestoneExists returns false although the milestone exists, so
2. ensureMilestone POSTs a create. GitHub does NOT enforce milestone-title uniqueness (unlike labels), so the POST SUCCEEDS and creates a duplicate.
3. The re-list at the end of ensureMilestone still cannot see it — still page 1 — so it returns false. haveMilestone stays false, task issues are silently never grouped, and a fresh duplicate milestone accumulates on every single push.

The doc comment above ensureMilestone is wrong on both counts past 30 milestones: duplicate creates do not 422, and the re-list is not complete.

Latent — it needs 30+ milestones, and dacli maps one per project — but silent and compounding once crossed.

Worth noting where this came from: the code shipped in PRs 322 and 327, both of which merged WITHOUT CI ever running (task 263). The auditor went looking there for exactly that reason, and found something.
