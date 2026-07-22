---
id: d-github-as-a-materialized-projection-for-humans-local-append-only-log-stays-the
kind: note
note_kind: decision
created: 2026-07-22T16:22:55Z
created_by: a-root
---
# GitHub as a materialized projection for humans; local append-only log stays the agent-facing source of truth (G-series)
## Chose
GitHub as a materialized projection for humans; local append-only log stays the agent-facing source of truth (G-series)
## Rejected
make GitHub Issues/Discussions/PRs the primary store agents read+write directly
## Because
agents need offline, sub-ms, conflict-free (ULID append-only) reads/writes that travel WITH the code and are replayable/taint-traceable — the GitHub API is none of those (network, rate limits, no provenance, decoupled from the commit). BUT humans live in GitHub, so the record must be MIRRORED there: decisions->Discussions, tasks->Issues, findings->comments, verify->PR reviews, synced on ship/sync not on the agent hot path. Local stays truth; GitHub becomes the human collaboration surface.
