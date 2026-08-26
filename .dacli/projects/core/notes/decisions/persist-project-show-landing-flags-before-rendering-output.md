---
id: d-persist-project-show-landing-flags-before-rendering-output
kind: note
note_kind: decision
created: 2026-08-26T14:23:36Z
created_by: a-fixer-eqe3tq
about: "[[t-01M0F8DMCN93FCDE59FSEDTJB3]]"
---
# Persist project show landing flags before rendering output
## Chose
Persist project show landing flags before rendering output
## Rejected
Continue treating landing flags as display-only overrides
## Because
The documented invocation configures durable project policy, and a display-only override silently leaves later integration on the legacy policy.
