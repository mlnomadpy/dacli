---
id: t-01KY5JSH6ZVA5N8NP29D2Y7BRB
kind: task
created: 2026-07-22T18:53:14Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# FIX-gh: ghmirror perf + attribution + selfreport gate (WAVE 2 — after G6)
## Acceptance
- [x] searchByMarker does not do a full gh issue list per task/decision inside the push loop (batch or cache); cmdPush does not rewrite every task file on every push when the mapping is unchanged
- [x] findingAboutTask matches the task precisely (not a loose zero-padded-seq substring that cross-matches); disclosure consent is scoped to the consented repo, not a bare boolean
- [x] dacli report honors a disclosure gate and does not attach workspace name + raw transcript tail to a public upstream ungated
- [x] committed by an agent; build + test green
## Log
- 2026-07-22T19:31:57Z claimed by a-a3xyv593bf
- 2026-07-22T19:41:58Z accepted by a-root
- 2026-07-22T19:41:58Z completed by a-root
