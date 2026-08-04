---
id: f-ci-failed-to-trigger-on-3-of-this-session-s-prs-each-needed-a-main-merge-push
kind: note
note_kind: finding
created: 2026-08-04T13:01:02Z
created_by: a-root
origin: .github/workflows/ci.yml:4
---
# CI failed to trigger on 3 of this session's PRs; each needed a main-merge push to get checks at all
PRs 297 (task 242), 322 (224) and 327 (261) all reported 'no checks reported on the branch' — no workflow run existed for the head SHA at all, not a failing one.

The workflow triggers on `pull_request:` with no path or branch filter, so every PR should get a run. What the three have in common is the ORDER of operations: the branch was pushed to origin first, and `gh pr create` opened the PR afterwards, seconds later. The PRs that got checks normally were opened the same way, so it is not deterministic — it looks like a race between the push event and the PR-open event on GitHub's side, where the pull_request trigger lands with no new commit to run against.

The recovery is the same every time and worth knowing: merge origin/main into the branch and push. A new head SHA produces a new pull_request synchronize event, and checks run. `gh pr update-branch` does this too, but it refuses when the merge conflicts — which was the case for all three, so each needed a manual conflict resolution first.

The dangerous part is not the missing run, it is the silence: `gh pr checks` says 'no checks reported', `mergeStateStatus` says UNKNOWN, and nothing surfaces it as a problem. Task 216 already made dacli treat 'no checks' as NOT-passing rather than passing, so integrate leaves the PR open — correctly, but without saying why it will sit there forever. Filed as 263.
