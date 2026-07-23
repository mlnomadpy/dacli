---
id: d-ad-hoc-tracking-is-dacli-run-cmd-dacli-executes-and-records-not-a-passive-log
kind: note
note_kind: decision
created: 2026-07-23T10:08:42Z
created_by: a-e3jt52gb4k
about: [[090]]
---
# ad-hoc tracking is dacli run --cmd (dacli executes and records), not a passive log of commands run elsewhere
## Chose
ad-hoc tracking is dacli run --cmd (dacli executes and records), not a passive log of commands run elsewhere
## Rejected
a --record/--log-only verb that appends an EventRun for a command the agent already ran via its own shell tool, without dacli executing it
## Because
the shortcut engine's safety story (docs/SHORTCUTS.md) is about dacli controlling what it executes; a log-only path would let an agent claim a run it never made, and it duplicates run's existing exec+eventlog.Append pattern for nothing. About is a sha256 content-hash of the command (adhocKey), not the raw text, because About is a wikilink target in the event format and raw command text can carry anything
