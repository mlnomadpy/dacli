---
id: t-01KZYQ5EHSHB04Z1ASZFJNNADN
kind: task
created: 2026-08-13T23:26:22Z
created_by: a-root
owner: a-root
github:
  issue: 629
  repo: mlnomadpy/dacli
---
# [agent-report] commit claim matcher rejects files inside the displayed claim
## Context
Adopted from GitHub issue #629.

Task 001's dacli commit refusal displays claim [docs/supabase/**, .dacli/projects/supabase/**] but rejects four files whose paths all begin docs/supabase/: firebase-inventory.json, firebase-to-supabase-contract.md, firebase-to-supabase-verification.json, verify-firebase-supabase.mjs. Files were staged by dacli and no paths outside that claim are staged.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
## Log
