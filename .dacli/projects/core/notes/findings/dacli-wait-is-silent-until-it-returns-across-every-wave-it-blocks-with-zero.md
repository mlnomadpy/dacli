---
id: f-dacli-wait-is-silent-until-it-returns-across-every-wave-it-blocks-with-zero
kind: note
note_kind: finding
created: 2026-07-22T15:04:04Z
created_by: a-root
severity: minor
---
# dacli wait is silent until it returns: across every wave it blocks with zero incremental output, so progress is invisible (and it repeatedly hit the caller's timeout). wait should stream 'agent X done (N/M)' as each finishes
