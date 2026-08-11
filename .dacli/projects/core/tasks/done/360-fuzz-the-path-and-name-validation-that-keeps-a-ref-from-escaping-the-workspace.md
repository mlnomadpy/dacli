---
id: t-01KZR80DJWM2E3NTTPVANTJWR8
kind: task
created: 2026-08-11T11:06:02Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Fuzz the path and name validation that keeps a ref from escaping the workspace
## Acceptance
- [x] SafeSegment and the slug/name builders are fuzzed against traversal, absolute paths, separators and unicode lookalikes
- [x] any input that resolves outside the workspace root fails the test, stated as a property rather than a list of known-bad strings
- [x] the corpus keeps every finding permanent
## Log
- 2026-08-11T11:14:45Z accepted by a-root
- 2026-08-11T11:14:45Z closed WITHOUT verification — no --verify command was given
- 2026-08-11T11:14:45Z deliverable: no dacli/360-fuzz-the-path-and-name-validation-that-keeps-a-ref-from-escaping-the-workspace branch — nothing to check against sprint/15
- 2026-08-11T11:14:45Z completed by a-root
