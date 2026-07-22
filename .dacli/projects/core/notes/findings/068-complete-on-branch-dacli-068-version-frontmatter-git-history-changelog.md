---
id: f-068-complete-on-branch-dacli-068-version-frontmatter-git-history-changelog
kind: note
note_kind: finding
created: 2026-07-22T20:15:36Z
created_by: a-64p4b0yykq
about: [[068]]
severity: moderate
---
# 068 complete on branch dacli/068 — version frontmatter + git-history changelog, build+test green
Commit 39e2cbc by a-64p4b0yykq. Staged ONLY 5 scoped files (git add + dacli commit --no-add): internal/store/{version.go,version_test.go}, internal/store/roles.go, internal/features/teamops/teamops.go, internal/features/skillforge/skillforge.go. ACCEPTANCE: (1) roles+skills carry version: frontmatter (roles.go CreateRole sets version=v1; skillforge cmdAdd sets version=store.DefaultVersion). role show (NEW cmd, teamops.go:cmdRoleShow) and skill show (skillforge.go:cmdShow) print 'version: vN' + a changelog derived from git history — store.FileChangelog runs 'git log --follow --format=%h|%an|%ar|%s' on the manifest file (who/when/what). (2) a change bumps or prompts: 'dacli role bump'/'dacli skill bump' increment vN->v(N+1) in frontmatter (store.NextVersion, store.BumpFileVersion, rw-gated); show prints '⚠ changed in N commit(s) since vN was set' via store.VersionIsStale (git -S on the version: line + commit-count since). Versions are human-legible incrementing v1/v2. (3) committed by agent; go build ./... clean; go test ./internal/... all green incl. new store TestNextVersion/TestFileVersionAndBump/TestFileChangelogAndStaleness (real temp git repo). NOTE: binary smoke blocked by headless sandbox (arbitrary-path exec needs approval); verified via build+unit tests. Box-check is owner-only (a-root) — owner: dacli accept 068 then integrate/merge --task 068.
