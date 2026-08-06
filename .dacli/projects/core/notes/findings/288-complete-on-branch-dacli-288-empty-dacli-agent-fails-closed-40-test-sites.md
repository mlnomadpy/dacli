---
id: f-288-complete-on-branch-dacli-288-empty-dacli-agent-fails-closed-40-test-sites
kind: note
note_kind: finding
created: 2026-08-06T08:20:30Z
created_by: a-fixer-aa9apn
about: "[[288]]"
severity: major
---
# 288: complete on branch dacli/288 — empty DACLI_AGENT fails closed, ~40 test sites migrated blank-to-unset
Commit 71480d5 (a-fixer-aa9apn). Both acceptance criteria met.

(1) internal/agentid/agentid.go: Resolve now uses os.LookupEnv(EnvVar) instead of os.Getenv, splitting into resolveToken(w, tok, present) — present=false (unset) resolves to root as before; present=true and tok=="" (set but empty) now returns new ErrEmptyToken instead of silently falling through to root. clikit.ExitCode maps ErrEmptyToken to exit 3 (a policy refusal, not a transient error) so a supervisor does not retry a broken environment.

(2) Testability without relying on the environment the test runs under: resolveToken is the pure core, called directly with explicit (tok, present) tuples in TestResolveTokenDistinguishesEmptyFromUnset — no env mutation needed to prove the distinction. TestResolveFailsClosedOnEmptyToken exercises the real env path end to end.

Consequence handled: making present-but-empty fail closed meant every existing t.Setenv(DACLI_AGENT, "") — the established convention for forcing tests to act as root regardless of the ambient env — now triggers the new refusal instead of resolving to root. Migrated all ~40 such sites (24 files across internal/cli and internal/features/*) from blank-to-unset, using os.LookupEnv+os.Unsetenv (t.Setenv cannot itself unset). internal/cli sites unset directly (TestMain already establishes the unset baseline for that package); other packages got a small per-package unsetAgentEnv(t) helper (or inline for single-occurrence files) that restores whatever the process started with.

PROOF: go build ./... clean, go vet ./... clean, gofmt -l . clean. go test ./... green across all 40 packages (both with -exec 'env -u DACLI_AGENT' and as plain go test, which exercises whatever ambient DACLI_AGENT this dogfood session actually carries). Before the fix, running the full suite failed in internal/features/{teamops,vcs,wscore} etc. with 'DACLI_AGENT is set but empty (lost agent identity); refusing to fall back to root' — the exact regression these ~40 sites needed migrating to avoid, confirming the fix's fail-closed behavior actually fires end-to-end, not just in the two new agentid tests.

Owner: dacli accept 288 then integrate/merge --task 288 — task check/done are gated to the owner (a-root), reported here instead per convention.
