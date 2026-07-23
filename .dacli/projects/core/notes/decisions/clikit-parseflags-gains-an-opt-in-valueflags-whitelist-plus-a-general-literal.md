---
id: d-clikit-parseflags-gains-an-opt-in-valueflags-whitelist-plus-a-general-literal
kind: note
note_kind: decision
created: 2026-07-23T15:13:06Z
created_by: a-3s7f6bmn06
about: [[107]]
---
# clikit.ParseFlags gains an opt-in valueFlags whitelist plus a general -- literal terminator, instead of a global schema
## Chose
clikit.ParseFlags gains an opt-in valueFlags whitelist plus a general -- literal terminator, instead of a global schema
## Rejected
making ParseFlags error whenever any --key is immediately followed by another --token (the fully general reading of the bug report)
## Because
that would break every existing bare-boolean-flag-adjacent-to-flag invocation across the app (e.g. spawn --cooperative --review, ship --dry-run --force) since the parser genuinely cannot tell two adjacent booleans from a value-flag whose value starts with -- without knowing which flags are boolean, and the prior decision d-parseflags-dash-leading-values-use-the-form-matching-stdlib-convention already rejected a full per-command schema. Instead: ParseFlags(args, valueFlags...) lets a call site name its OWN flags that are never boolean (opt-in, not global) -- runtime add now passes flag/arg/sandbox-ro-arg/model-flag, so their space-form value is always captured verbatim (missing value is now a clear Usagef error naming the = and -- mechanisms) -- and a new literal -- terminator (--key -- value) lets ANY flag on ANY command force a dash-leading literal value without whitelisting. The documented = form is unchanged. This closes the run 01KY2K8N4C corruption at its actual call site without touching the ambiguity every other bare-boolean flag in the app depends on.
