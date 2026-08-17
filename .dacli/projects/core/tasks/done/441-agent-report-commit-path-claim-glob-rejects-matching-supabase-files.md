---
id: t-01KZYQ5E9PFVWRVMSWPB39E38K
kind: task
created: 2026-08-13T23:26:22Z
created_by: a-root
owner: a-root
github:
  issue: 641
  repo: mlnomadpy/dacli
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# [agent-report] commit path-claim glob rejects matching Supabase files
## Context
Adopted from GitHub issue #641.

From isolated task branch dacli/002-create-the-supabase-schema-migrations-and-rls-security-tests, task claim printed as [supabase/**, package.json, package-lock.json, .github/workflows/**]. Running dacli commit with staged supabase/config.toml, supabase/migrations/*.sql, supabase/tests/database/rls.test.sql and other supabase paths refused them as outside the claim even though they match supabase/**. The same refusal correctly excluded scripts/verify-supabase-types.mjs and src/types/database.types.ts, which were subsequently moved/removed to honor scope.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] `internal/procmon` regression tests prove `supabase/**` overlaps `supabase/config.toml` and `docs/supabase/**` overlaps deep descendants.
- [x] The matcher remains symmetric for recursive tree claims and rejects sibling-prefix paths such as `supabase-old/file.sql`.
- [x] Exact file claims and existing literal directory claims retain their current behavior.
- [x] Embedded or unsupported wildcard forms are not silently broadened; their policy is documented and tested.
- [x] `internal/features/vcs` regression coverage proves `dacli commit` accepts staged descendants covered by a trailing-`/**` claim and still refuses paths outside the claim.
- [x] The new regression test demonstrably fails against the pre-fix matcher.
- [x] `go test ./internal/procmon ./internal/features/vcs` passes.
## Log
- 2026-08-16T18:34:28Z claimed by a-maintainer-w9qqkt
- 2026-08-17T15:29:33Z accepted by a-root (applied 1 proposal(s))
- 2026-08-17T15:29:33Z verified by `GOCACHE=/tmp/dacli-441-final-retry go test ./...` (exit 0) in branch main at 98ba962 — proves that tree builds, not that the work is in trunk
- 2026-08-17T15:29:33Z deliverable: dacli/441-agent-report-commit-path-claim-glob-rejects-matching-supabase-files is merged into main
- 2026-08-17T15:29:33Z completed by a-root
- 2026-08-17T15:40:19Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/680 (event 01M05Y0E3CF1311K4PD981BSTA)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-441-accept go test ./...","exit_code":0,"duration_ms":69432,"artifact_hash":"sha256:745e764604b03c79309b23df3a66de63c7cf42b46741405f70f186f12f127e34","verifier":"a-root","branch":"main","commit_sha":"98ba962ef40ed07fc769d5165fa5948ccc1c01e0"}
{"command":"GOCACHE=/tmp/dacli-441-final-retry go test ./...","exit_code":0,"duration_ms":70284,"artifact_hash":"sha256:4e802a49f0438c7a1662fa02c85c738a677dfae4b70880ac5fb26da59dba61d7","verifier":"a-root","branch":"main","commit_sha":"98ba962ef40ed07fc769d5165fa5948ccc1c01e0"}
