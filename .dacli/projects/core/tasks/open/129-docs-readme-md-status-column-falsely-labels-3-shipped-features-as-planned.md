---
id: t-01KY88V1R8G3B9T9ZCWC9SDPNY
kind: task
created: 2026-07-23T19:57:02Z
created_by: a-zq4qdv7py6
owner: a-zq4qdv7py6
priority: should
---
# docs/README.md status column falsely labels 3 shipped features as planned/unimplemented — restore status honesty
## So that
the front-page doc index whose own line 3 says 'a spec that pretends to be implemented is worse than either' stops doing the inverse — pretending shipped features are unimplemented — which directly serves the project goal of honest planned()-stub status
## Acceptance
- [ ] docs/README.md line 12 SHORTCUTS status drops '(promote planned)': shortcut promote is shipped (internal/features/shortcuts/shortcuts.go:24 registration, :65 cmdPromote; commit 051d82d)
- [ ] docs/README.md line 14 SKILLS status drops '(promote planned)': skill promote is shipped (internal/features/skillforge/skillforge.go:27 registration, :71 cmdPromote; commit 5aa70e5)
- [ ] docs/README.md line 18 GITHUB status drops 'inbound planned': github pull/sync are shipped (internal/features/ghmirror/ghmirror.go:420 cmdPull, :489 cmdSync)
- [ ] docs/README.md line 24 prose no longer claims any 'genuinely unimplemented planned() stubs' remain: grep confirms zero clikit.Planned call sites in product code (only the definition at internal/clikit/clikit.go:76)
- [ ] docs-only change: no Go source edited; each restored status verified against the live command registration it names
## Log
