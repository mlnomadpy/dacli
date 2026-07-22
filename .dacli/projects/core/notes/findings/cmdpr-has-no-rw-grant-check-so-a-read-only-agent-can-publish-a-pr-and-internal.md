---
id: f-cmdpr-has-no-rw-grant-check-so-a-read-only-agent-can-publish-a-pr-and-internal
kind: note
note_kind: finding
created: 2026-07-22T18:23:33Z
created_by: a-7sy0x8b84g
about: [[t-01KY5GP5RHAPGVT35FDFS4Z0PB]]
source_event: 01KY5H0YJTAP17YB4W4K3410BM
github:
  issue: 24
  repo: mlnomadpy/dacli
---
# cmdPR has no rw-grant check, so a read-only agent can publish a PR (and internal findings/verdicts) to GitHub
internal/features/vcs/lifecycle.go:130-175 cmdPR — unlike cmdPush (:110), cmdMerge (:323) and cmdIntegrate (:394), which all refuse a non-rw grant, cmdPR performs NO id.Grant != model.GrantRW check before running 'gh pr create' (an outward-facing GitHub write) and, with --with-verdicts, 'gh pr review --comment' posting taskFindings + verdictReview. This is exactly the brief's rank-2 risk ('public-repo mirror leaks internal findings'): a read-only agent can open a PR whose body carries the task's finding notes (prBody->taskFindings, :190/:230) and the verify panel's verdicts to a public repo. Exploitation needs the branch already on origin (a ro agent can't cmdPush it), so it is not always reachable — but the missing gate is inconsistent with every other outward vcs command and should refuse ro like they do.
