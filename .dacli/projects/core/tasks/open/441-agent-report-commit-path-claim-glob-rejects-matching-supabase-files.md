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
- [ ] `internal/procmon` regression tests prove `supabase/**` overlaps `supabase/config.toml` and `docs/supabase/**` overlaps deep descendants.
- [ ] The matcher remains symmetric for recursive tree claims and rejects sibling-prefix paths such as `supabase-old/file.sql`.
- [ ] Exact file claims and existing literal directory claims retain their current behavior.
- [ ] Embedded or unsupported wildcard forms are not silently broadened; their policy is documented and tested.
- [ ] `internal/features/vcs` regression coverage proves `dacli commit` accepts staged descendants covered by a trailing-`/**` claim and still refuses paths outside the claim.
- [ ] The new regression test demonstrably fails against the pre-fix matcher.
- [ ] `go test ./internal/procmon ./internal/features/vcs` passes.
## Log
