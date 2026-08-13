---
id: d-cover-the-existing-proc-txt-outcome-as-the-single-terminal-lifecycle-state
kind: note
note_kind: decision
created: 2026-08-13T21:03:10Z
created_by: a-codex-maintainer-8r5s5s
about: "[[436]]"
github:
  issue: 625
  repo: mlnomadpy/dacli
---
# Cover the existing proc.txt Outcome as the single terminal lifecycle state
## Chose
Cover the existing proc.txt Outcome as the single terminal lifecycle state
## Rejected
Add another recovered-state flag or infer terminality again from outcome.md
## Because
agents already atomically releases claims and stamps proc.txt via CompleteRecord; wait must consume that same durable state even while outcome.md remains the detached-running placeholder
