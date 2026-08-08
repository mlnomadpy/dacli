---
id: f-077-complete-on-branch-dacli-077-fix-gh-ghmirror-perf-attribution-selfreport
kind: note
note_kind: finding
created: 2026-07-22T19:41:14Z
created_by: a-a3xyv593bf
about: [[077]]
severity: moderate
---
# 077 complete on branch dacli/077-fix-gh-ghmirror-perf-attribution-selfreport-gate-wave-2-after-g6 (commit 344e906) — all 4 acceptance met
Committed 344e906 by a-a3xyv593bf (maintainer) via git add + dacli commit --no-add, staging ONLY 4 files: internal/features/ghmirror/{ghmirror.go,ghmirror_test.go}, internal/features/selfreport/selfreport.go, internal/cli/selfreport_test.go. ACCEPTANCE: (1) PERF — searchByMarker replaced by markerIndex fetched ONCE per push (newMarkerIndex/idx.find, ghmirror.go), shared across the task loop, mirrorDecisions, and mirrorFindingIssues, so adoption no longer costs a full 'gh issue list' per note; cmdPush/mirrorDecisions/mirrorFindingIssues now write the github: block back only when it changed (mappedBlockChanged + githubBlock), so an idempotent re-push rewrites no task/note file. (2) ATTRIBUTION — findingAboutTask matches EXACTLY via aboutRefs (unwraps [[ref]] wikilinks) + taskMatchesRef (mirrors store.matchesRef: ULID/slug/bare-seq/NNN/NNN-slug), no loose zero-padded substring (TestFindingAboutTaskPrecise proves 10007/0070/070/70/008 no longer cross-match task 007); disclosure consent is repo-scoped — github_public_confirmed now stores the repo nameWithOwner and disclosureGate uses consentCoversRepo to compare it to the LIVE repo, so consent for one public repo does not authorize another, and a legacy 'true' fails closed. (3) SELFREPORT GATE — dacli report withholds the workspace name + raw transcript tail by default (public upstream), attaching them only with --disclose; dry-run stays network-free. (4) go build ./... clean; go test -exec 'env -u DACLI_AGENT' ./internal/... all green incl. ghmirror + cli. Owner: verify and close via dacli task check/done + dacli merge --task 077.
