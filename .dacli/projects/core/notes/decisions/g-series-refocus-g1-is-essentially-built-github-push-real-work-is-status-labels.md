---
id: d-g-series-refocus-g1-is-essentially-built-github-push-real-work-is-status-labels
kind: note
note_kind: decision
created: 2026-07-22T16:43:12Z
created_by: a-root
---
# G-series refocus: G1 is essentially built (github push); real work is status-labels + G2 Discussions + G3 PR-reviews + inbound
## Chose
G-series refocus: G1 is essentially built (github push); real work is status-labels + G2 Discussions + G3 PR-reviews + inbound
## Rejected
build G1 tasks->issues from scratch
## Because
github push already does idempotent, gated, close-on-done tasks->issues with backlink; duplicating it would waste effort and risk regressions. Narrow G1 to status-labels; prioritize decisions->Discussions and verify->PR-reviews and inbound (the planned stubs) as the actual 'full GitHub' gaps
