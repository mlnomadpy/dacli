---
id: f-claim-an-empty-lost-dacli-agent-resolves-to-root-rw-a-root-a-spawned-child-that
kind: note
note_kind: finding
created: 2026-08-04T18:18:12Z
created_by: a-reviewer-664htb
about: "[[t-01KZ6S9FVG533ABEVXZBZZ7SC3]]"
source_event: 01KZ6SSX76157SW2TWA4B16AWT
---
# claim: an empty/lost DACLI_AGENT resolves to root RW (a-root) -- a spawned child that loses its token silently ESCALATES to full grant instead of failing closed
Stage: CLAIM (identity resolution). agentid.Resolve (agentid.go:78-82): 'tok := os.Getenv(EnvVar); if tok == "" { return &Identity{ID: RootID, Grant: model.GrantRW, Role: "root"}, nil }'. A non-empty token that matches nothing returns ErrBadToken (agentid.go:107) -- fail closed, good. But an EMPTY token returns a-root with GrantRW -- fail OPEN. Concrete failure: a child spawned with grant ro/attenuated, whose DACLI_AGENT is dropped from the environment (a nested subprocess that does not inherit env, a runtime/shell wrapper that sanitizes env, an env-stripping sandbox), does not error -- every subsequent dacli commit/check/done/accept runs as a-root RW and can mutate any task and commit to the repo, bypassing the intended attenuation (agentid.go:147-149 makes attenuation monotonic ON spawn, but Resolve undoes it when the token vanishes at run time). Missing credential should refuse, not escalate to root. The grant model is already cooperative/advisory (agentid.go:1-9), so this is the one place a token loss becomes privilege GAIN. Discoverable only by reading agentid.go:80-82.
