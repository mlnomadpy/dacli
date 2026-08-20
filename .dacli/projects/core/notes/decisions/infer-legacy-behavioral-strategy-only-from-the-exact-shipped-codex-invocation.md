---
id: d-infer-legacy-behavioral-strategy-only-from-the-exact-shipped-codex-invocation
kind: note
note_kind: decision
created: 2026-08-20T09:43:21Z
created_by: a-maintainer-1ckxmn
about: "[[t-01M0F8JAH5CNJ327M31B1821BF]]"
---
# Infer legacy behavioral strategy only from the exact shipped Codex invocation contract
## Chose
Infer legacy behavioral strategy only from the exact shipped Codex invocation contract
## Rejected
Infer from runtime name, usage_format, or any exec --json prefix
## Because
Exact binary basename, prompt mode, global args, invocation args, and read-only sandbox preserve mature workspace role references without granting provider-specific execution to ambiguous custom adapters.
