---
id: t-01KZX7PXQBEVM1M0N2BKWYD4RK
kind: task
created: 2026-08-13T09:37:03Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
depends_on: "[403:FS, 404:FS]"
github:
  issue: 550
  repo: mlnomadpy/dacli
---
# Implement explicit provider limit policies and fallback chains
## So that
quota exhaustion pauses or changes runtime predictably instead of burning repeated runs
## Context
Use typed failure classification, Retry Policy, Circuit Breaker, and an explicit Chain of Responsibility. Fallback is opt-in per role; dacli must never silently switch vendor, model, grant, or data boundary.
## Acceptance
- [x] Runtime adapters map rate limit, quota exhausted, authentication, unavailable, and permanent input failures to typed outcomes
- [x] Retry policy honors provider reset metadata and applies bounded exponential backoff with jitter when reset metadata is absent
- [x] A per-runtime circuit breaker prevents repeated spawns during a recorded cooldown and survives loop restart
- [x] Fallback chains are explicit ordered role policy and preserve or strengthen grant and capability requirements
- [x] A fallback or pause prints and records source runtime, destination runtime, reason, and cooldown
- [x] Tests prove permanent and policy failures do not trigger fallback
## Log
- 2026-08-13T10:13:51Z claimed by a-codex-maintainer-76ksyq
- 2026-08-13T10:46:12Z accepted by a-root (applied 1 proposal(s))
- 2026-08-13T10:46:12Z closed WITHOUT verification — no --verify command was given
- 2026-08-13T10:46:12Z completed by a-root
- 2026-08-13T10:46:19Z deliverable: dacli/406-implement-explicit-provider-limit-policies-and-fallback-chains exists but is NOT in main — closed anyway
- 2026-08-13T10:46:34Z reopened by a-root: Provider-policy core is verified, but runtime outcome classification and spawn/supervise integration were removed after the path-claim refusal; acceptance criteria 1, 5, and end-to-end fallback behavior are not yet fully landed. (cleared 6 acceptance box(es) — the close claimed work that was not verified)
- 2026-08-13T10:47:42Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/561 (event 01KZXBNPFQBF7AY9Z8CGBFMYQN)
- 2026-08-13T11:37:56Z accepted by a-root
- 2026-08-13T11:37:56Z verified by `cd /Users/tahabsn/Documents/GitHub/dacli/.dacli/worktrees/core-406-implement-explicit-provider-limit-policies-and-fallback-chains && gofmt -l . && GOCACHE=/private/tmp/dacli-accept-406 go vet ./... && GOCACHE=/private/tmp/dacli-accept-406 go test ./internal/features/execution ./internal/providerpolicy ./internal/store ./internal/team` (exit 0) in branch main at 3ff307e — proves that tree builds, not that the work is in trunk
- 2026-08-13T11:37:56Z completed by a-root
- 2026-08-13T11:39:31Z a-verifier-m44jfv: verify-verdict: no-verdict — cc (a-verifier-m44jfv) on claim: internal/features/execution,internal/providerpolicy,internal/store,internal/team — panelist reported nothing — counts as unconfirmed (event 01KZXCSPEKG9ZGHRSQTJ62R70N)
- 2026-08-13T11:39:31Z a-verifier-yt78vq: verify-verdict: no-verdict — codex-ro (a-verifier-yt78vq) on claim: internal/features/execution,internal/providerpolicy,internal/store,internal/team — panelist reported nothing — counts as unconfirmed (event 01KZXCW82RJM25RK3MXEAR4X7Z)
- 2026-08-13T11:39:31Z a-verifier-7w0rza: verify-verdict: no-verdict — cc (a-verifier-7w0rza) on claim: runtime-execution-now-enforces-persisted-provider-cooldowns — panelist reported nothing — counts as unconfirmed (event 01KZXCWT7ZPVA9X5EYFNAZEGZP)
- 2026-08-13T11:39:31Z a-verifier-wg8fr4: verify-verdict: no-verdict — codex-ro (a-verifier-wg8fr4) on claim: runtime-execution-now-enforces-persisted-provider-cooldowns — panelist reported nothing — counts as unconfirmed (event 01KZXCZC9STK9J39MZ2QJ2W414)
- 2026-08-13T11:39:31Z a-verifier-tbhkzb: verify-verdict: no-verdict — cc (a-verifier-tbhkzb) on claim: f-runtime-execution-now-enforces-persisted-provider-cooldowns — panelist reported nothing — counts as unconfirmed (event 01KZXD064TKW98JH00GAQ3ETSK)
- 2026-08-13T11:39:31Z a-verifier-w5ddgm: verify-verdict: no-verdict — codex-ro (a-verifier-w5ddgm) on claim: f-runtime-execution-now-enforces-persisted-provider-cooldowns — panelist reported nothing — counts as unconfirmed (event 01KZXD1TY6370AQAHZ1H3E2GWH)
- 2026-08-13T11:39:31Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/561 at merge commit b4db035c51de0b57adffc601b155e36620ad0c43 into main; local cleanup complete (event 01KZXEMWR8KNKR8M2JADV3675E)
