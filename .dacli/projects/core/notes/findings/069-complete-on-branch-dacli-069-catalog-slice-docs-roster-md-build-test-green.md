---
id: f-069-complete-on-branch-dacli-069-catalog-slice-docs-roster-md-build-test-green
kind: note
note_kind: finding
created: 2026-07-22T20:53:51Z
created_by: a-0bsqxp2kpx
about: [[069]]
severity: moderate
---
# 069 complete on branch dacli/069-...; catalog slice + docs/ROSTER.md, build+test green
Commit 9992df1 by a-0bsqxp2kpx (maintainer), staged ONLY 4 files via git add + dacli commit --no-add: internal/features/catalog/{catalog.go,catalog_test.go} (new slice), internal/cli/cli.go (register), docs/ROSTER.md (sample). Acceptance: (1) 'dacli catalog' generates a browsable markdown catalog of every role (name, version, grant, kind, model, skills, purpose, last-changed via store.FileChangelog) and every skill (name, version, purpose, est-tokens) grouped into GitHub-scannable tables — renderCatalog (catalog.go:184) is pure/deterministic and cell()-escapes pipes+newlines; reads via store.LoadRoles + skills.LoadSkills + store.FileVersion/FileChangelog (entity layer, no feature import). (2) one-way: banner states 'do not edit this page … edit its file under .dacli/ (via PR)'; source stays in .dacli. (3) --publish-wiki mirrors to <owner/repo>.wiki.git honoring the SAME disclosure gate as github push (disclosureGate+consentCoversRepo reimplemented locally since arch_test forbids importing ghmirror; PUBLIC repo needs recorded per-project consent), best-effort — a wiki clone/push failure degrades to a stderr warning and exit 0, never failing the docs write; no live network in tests. go build ./... clean; go test ./internal/... all green incl. TestFeatureSlicesAreIsolated + new catalog tests. Owner: verify + close via dacli accept 069, then integrate --tasks 069.
