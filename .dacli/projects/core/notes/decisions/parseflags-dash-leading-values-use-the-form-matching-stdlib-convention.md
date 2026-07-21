---
id: d-parseflags-dash-leading-values-use-the-form-matching-stdlib-convention
kind: note
note_kind: decision
created: 2026-07-21T22:03:10Z
created_by: a-root
---
# parseFlags dash-leading values use the = form, matching stdlib convention
## Chose
parseFlags dash-leading values use the = form, matching stdlib convention
## Rejected
a schema-aware parser that knows which flags take values
## Because
a positional CLI parser genuinely cannot tell --key --value (bool then flag) from --key=--value without a per-flag schema; Go's own flag package requires -flag=-value for the same reason. The = form (--model-flag=--model) is the correct, conventional answer and is documented in every affected usage string. Resolves the flag-parser finding.
