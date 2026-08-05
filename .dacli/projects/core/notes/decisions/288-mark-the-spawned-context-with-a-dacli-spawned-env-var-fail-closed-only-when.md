---
id: d-288-mark-the-spawned-context-with-a-dacli-spawned-env-var-fail-closed-only-when
kind: note
note_kind: decision
created: 2026-08-04T20:42:16Z
created_by: a-maintainer-s36xgr
about: "[[288]]"
---
# 288: mark the spawned context with a DACLI_SPAWNED env var; fail closed only when a token is missing AND we were spawned
## Chose
288: mark the spawned context with a DACLI_SPAWNED env var; fail closed only when a token is missing AND we were spawned
## Rejected
Make agentid.Resolve fail closed on any present-but-empty DACLI_AGENT (os.LookupEnv present==true, value=='')
## Because
Present-but-empty is the finding's literal target, but ~40 test sites across ~20 feature slices use t.Setenv(DACLI_AGENT, "") as the canonical 'act as root' (task 262). Making present-empty fail closed reddens ~180 tests and forces a cross-slice migration of every slice's test harness -- violating the slice-boundary rule for a scoped task, and over-refusing a human/CI that legitimately exports DACLI_AGENT=. A spawn is the ONE place dacli sets the child env (execution.go execRuntime), so marking it (DACLI_SPAWNED=1) lets resolveToken fail closed precisely for 'lost token INSIDE a spawned context' (the acceptance wording) while empty-without-marker still resolves to root -- keeping the whole suite green and the fix scoped to agentid + clikit + one line in execution.
