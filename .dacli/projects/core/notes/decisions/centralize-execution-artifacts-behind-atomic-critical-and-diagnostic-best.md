---
id: d-centralize-execution-artifacts-behind-atomic-critical-and-diagnostic-best
kind: note
note_kind: decision
created: 2026-08-18T14:20:55Z
created_by: a-maintainer-zppm9n
about: "[[t-01M0AEG5AQPVJTH41MJNFRGSSX]]"
---
# Centralize execution artifacts behind atomic critical and diagnostic best-effort writes
## Chose
Centralize execution artifacts behind atomic critical and diagnostic best-effort writes
## Rejected
Keep proc, terminal, and enrichment writes as independent call-site helpers
## Because
One policy boundary makes criticality reviewable, preserves complete old files across rename failure, and gives optional loss a durable runs-show channel
