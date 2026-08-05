---
id: d-288-fail-closed-on-a-present-but-empty-dacli-agent-os-lookupenv-and-migrate-the
kind: note
note_kind: decision
created: 2026-08-04T20:49:30Z
created_by: a-maintainer-s36xgr
about: "[[288]]"
---
# 288: fail closed on a present-but-empty DACLI_AGENT (os.LookupEnv), and migrate the test convention from blank to unset -- no new spawn-marker env var
## Chose
288: fail closed on a present-but-empty DACLI_AGENT (os.LookupEnv), and migrate the test convention from blank to unset -- no new spawn-marker env var
## Rejected
A DACLI_SPAWNED marker env var set by execRuntime, keeping empty==root so the ~40 t.Setenv(DACLI_AGENT,"") sites stay green untouched
## Because
The marker keeps CI green now but leaks into go test under dogfood: a spawned agent's env carries DACLI_SPAWNED=1, every slice test clears only DACLI_AGENT, so resolveToken sees spawned+empty and every root-expecting test fails -- a deferred landmine, worse than an immediate red. And to fix the LITERAL titled bug (empty DACLI_AGENT in a spawned context must fail closed) present-empty must fail closed, which the dogfood test env uses via t.Setenv(EnvVar,""). Either design forces the ~40-site convention change; present-but-empty needs it NOW for CI-green, the marker needs it for dogfood-honesty. Present-but-empty is the smaller code surface (agentid+clikit, no execution.go, no new protocol var), aligns with the documented 'unset means the root agent' (cli.go:177), and the migration (blank->unset) mirrors what cli/main_test.go already does (os.Unsetenv) -- making the tests hermetic rather than relying on empty==root, the very accident that hid this bug.
