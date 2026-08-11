---
id: f-readme-md-and-docs-index-md-s-96-merged-prs-claim-is-stale-and-now-also
kind: note
note_kind: finding
created: 2026-08-11T09:47:01Z
created_by: a-fixer-1p9jwx
about: "[[354]]"
severity: minor
---
# README.md and docs/index.md's '96 merged PRs' claim is stale and now also inaccurate in kind
README.md:11 and docs/index.md:19 both say 'this tool built and hardened itself, across 96 merged PRs' — a snapshot number frozen from an earlier count (docs/SELFHOSTING.md's old intro cited '96 pull requests (#39-#293)' as of 2026-08-04, which I corrected in this task since it also claimed 'every one authored by a dacli agent', which is no longer true: PR #440/issue #437, commit a007429, carries zero Dacli-Agent trailers and five Co-Authored-By: Claude Opus 5 lines — an interactive session, not a spawned agent). main is now at 616 commits with 290 carrying a Dacli-Agent trailer (git log main --grep='^Dacli-Agent:' -E --oneline | wc -l). Out of scope for task 354 (whose acceptance criteria are about docs/SELFHOSTING.md specifically), but the two landing-page docs should either drop the fixed number for a reproducible-count pointer to SELFHOSTING.md, or be refreshed and kept refreshed.
