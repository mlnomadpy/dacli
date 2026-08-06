---
id: d-292-canonical-token-count-flags-all-end-in-tokens-max-tokens-run-cycle-ceiling
kind: note
note_kind: decision
created: 2026-08-04T20:48:20Z
created_by: a-maintainer-qh146m
about: "[[292]]"
---
# 292: canonical token-count flags all end in -tokens (--max-tokens run/cycle ceiling, --window-tokens window ceiling, --brief-tokens brief size); --window is the window duration; old --budget/--budget-window kept as documented aliases via new clikit Flags.Alias
## Chose
292: canonical token-count flags all end in -tokens (--max-tokens run/cycle ceiling, --window-tokens window ceiling, --brief-tokens brief size); --window is the window duration; old --budget/--budget-window kept as documented aliases via new clikit Flags.Alias
## Rejected
Merge all four flags into a single --max-tokens name
## Because
The four spellings the finding lists are NOT one concept: --max-tokens/--window-tokens are token-SPEND ceilings, --budget is a brief-SIZE cap (flows to brief.Assemble on spawn/verify/context alike), and --budget-window is a DURATION. Merging them would be wrong. Instead give each a predictable name (token counts end in -tokens; the run ceiling is --max-tokens; no more overloaded 'budget'), keep old spellings as aliases so no persisted loop arg or invocation breaks, and let Reject fail a wrong cross-command guess loudly.
