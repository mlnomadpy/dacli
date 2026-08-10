---
id: t-01KZPJRA6S3QNB3WDKQY8X9NWX
kind: task
created: 2026-08-10T19:35:22Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# accept's landing check ignores the sprint branch, so every accept during a sprint warns that work is not in trunk
## Acceptance
- [x] the landing check compares against the branch the work is being integrated into, not a hardcoded resolved trunk
- [x] an accept during a sprint whose branch HAS merged into the sprint branch reports it as landed, not as unlanded
- [x] a task genuinely absent from the integration target still warns, so the check does not become a rubber stamp
- [x] a test drives an accept against a non-main integration branch and asserts both outcomes
## Log
- 2026-08-10T19:59:17Z accepted by a-root
- 2026-08-10T19:59:17Z verified by `go test ./internal/features/acceptance/ ./internal/store/` (exit 0)
- 2026-08-10T19:59:17Z deliverable: no dacli/342-accept-s-landing-check-ignores-the-sprint-branch-so-every-accept-during-a branch — nothing to check against sprint/4
- 2026-08-10T19:59:17Z completed by a-root
