---
id: r-165-deferred-deployment-decision-not-code
kind: note
note_kind: ref
created: 2026-07-27T23:03:55Z
created_by: a-root
about: "[[165]]"
---
# 165 deferred: deployment decision, not code
Children must invoke dacli, so the fix is to install the binary to a non-writable path and allowlist THAT, rather than the gitignored writable ./dacli build output. Changing runtime config unilaterally could break the loop, so left for the operator.
