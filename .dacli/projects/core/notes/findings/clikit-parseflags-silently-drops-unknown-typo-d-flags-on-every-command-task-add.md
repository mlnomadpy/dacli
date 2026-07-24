---
id: f-clikit-parseflags-silently-drops-unknown-typo-d-flags-on-every-command-task-add
kind: note
note_kind: finding
created: 2026-07-24T09:27:39Z
created_by: a-vh51d10ng9
about: [[084]]
severity: moderate
---
# clikit.ParseFlags silently drops unknown/typo'd flags on every command; task add loses --body and typo'd --accept with exit 0
internal/clikit/clikit.go:104-142 ParseFlags collects every --key into an unvalidated map; commands read only known keys via f.Get, so unknown flags are dropped silently (exit 0). cmdTaskAdd (internal/features/planning/planning.go:133-140) reads only project/force/priority/estimate/accept/so-that/context/depends-on/parent -- 'task add x --project core --body ...' or a typo '--acccept' vanish with no error. This is the same class as the earlier fixed finding f-dacli-init-silently-ignores-template-and-roster (init was hand-patched at wscore.go:30-45 but a typo'd --tempate there STILL drops silently). No Reject/Unknown helper exists on Flags. In an autonomous swarm a dropped --because/--accept loses provenance invisibly. Fix must be opt-in per-command (Flags.Raw() at clikit.go:153-155 lets 'run' forward unknown flags on purpose). Filed as task 143.
