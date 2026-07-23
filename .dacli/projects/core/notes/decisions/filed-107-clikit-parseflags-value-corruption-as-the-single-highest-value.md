---
id: d-filed-107-clikit-parseflags-value-corruption-as-the-single-highest-value
kind: note
note_kind: decision
created: 2026-07-23T13:43:36Z
created_by: a-y4c733ze1d
about: [[084]]
---
# Filed 107 (clikit ParseFlags '-'-value corruption) as the single highest-value evidence-based change
## Chose
Filed 107 (clikit ParseFlags '-'-value corruption) as the single highest-value evidence-based change
## Rejected
the leaked .dacli-tmp-1516999556 in notes/findings (benign: loaders filter on .md so it is never mis-read), and various audit findings already covered by done tasks
## Because
the flag-parser bug is still live at clikit.go:104 (code comment 82-84 admits it, usage strings at execution.go:83 warn users around it), is grounded in a real failure (run 01KY2K8N4C sent garbage argv to claude), and directly manifests risk #1 (vendor CLI flag drift breaks adapters silently) across the runtime-adapter config surface --flag/--arg/--sandbox-ro-arg/--model-flag; no existing task addresses it
