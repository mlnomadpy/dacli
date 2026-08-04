---
id: d-256-prevent-git-background-auto-maintenance-in-shipenv-instead-of-reaping-the
kind: note
note_kind: decision
created: 2026-08-04T11:53:52Z
created_by: a-maintainer-g2363w
about: "[[256]]"
---
# 256: prevent git background auto-maintenance in shipEnv instead of reaping the detached PID
## Chose
256: prevent git background auto-maintenance in shipEnv instead of reaping the detached PID
## Rejected
register a t.Cleanup that finds and kills git's detached gc/maintenance child before TempDir removal
## Because
the racing writer is a git-spawned process reparented to init once it detaches; a test cannot reliably find or wait on it. Setting repo-local gc.auto=0 + gc.autoDetach=false + maintenance.auto=false overrides the CI-global config that triggers the auto path, so no detached git process is ever created to race t.TempDir's RemoveAll -- prevention is deterministic where reaping is racy.
