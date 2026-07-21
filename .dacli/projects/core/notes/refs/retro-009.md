---
id: r-retro-009
kind: note
note_kind: ref
created: 2026-07-21T21:33:16Z
created_by: a-root
about: [[t-01KY38DMYQA60MVQGMMMK9VKM4]]
scope: workspace
---
# Retro: 009
## Went well
- a real opus agent spawned through dacli's reviewer role found a major bug my own green tests missed — the model-tiered review ability paid for itself on first real use

## Didn't go well
- the reviewer could not self-file: Bash was gated in its spawned session despite the scoped sandbox, so I filed its findings by hand

## Improve next time
- investigate why Claude Code gated Bash(dacli:*) in headless -p mode; the review channel is only free if the agent can report itself

