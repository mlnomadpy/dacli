---
id: d-warn-not-refuse-on-os-executable-vs-runtime-allowlist-mismatch
kind: note
note_kind: decision
created: 2026-08-04T14:43:30Z
created_by: a-maintainer-c76h39
about: "[[267]]"
---
# warn (not refuse) on os.Executable vs runtime-allowlist mismatch
## Chose
warn (not refuse) on os.Executable vs runtime-allowlist mismatch
## Rejected
refuse with exit 3 like dacli 250's write-tool gate
## Because
Claude Code merges --allowedTools from settings.json too, so the runtime file's Bash rule is necessary-but-not-sufficient: a mismatch is a strong signal, not a certainty. Refusing on a stale-but-overridden path would block a spawn that would actually work; a loud stderr warning naming both paths surfaces it without the false refusal.
