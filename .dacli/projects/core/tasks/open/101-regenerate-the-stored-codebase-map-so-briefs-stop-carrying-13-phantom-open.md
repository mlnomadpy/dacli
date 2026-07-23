---
id: t-01KY78T5W8Y3TX8VK2G8TZZHJS
kind: task
created: 2026-07-23T10:37:19Z
created_by: a-yf43x5hk79
owner: a-yf43x5hk79
priority: should
---
# Regenerate the stored codebase map so briefs stop carrying 13 phantom 'Open markers'
## Acceptance
- [ ] The 'Open markers' block in .dacli/projects/core/project.md reflects the 087-fixed scanner: every listed entry is a real leading-comment TODO/FIXME/HACK/XXX, and none of the string-literal/prose false positives (onboard.go:198, gates.go:448, the fmt.Fprintf/test-fixture matches) remain
- [ ] A fresh brief for a core task (dacli context / task show brief) no longer contains any of the 13 phantom markers; verified by inspecting the regenerated codebase-map section
## Log
