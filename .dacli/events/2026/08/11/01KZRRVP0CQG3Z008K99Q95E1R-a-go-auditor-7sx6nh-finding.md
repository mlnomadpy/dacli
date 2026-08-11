---
id: 01KZRRVP0CQG3Z008K99Q95E1R
kind: event
event_kind: finding
created: 2026-08-11T16:00:32Z
created_by: a-go-auditor-7sx6nh
about: "[[t-01KZ93DW7P6BQ1HHDWY7MEH2KJ]]"
origin: agent
applied: false
---
CreateShortcut lacks the SafeSegment guard its siblings have, so a shortcut name can write outside .dacli

internal/store/shortcutfiles.go:24 (CreateShortcut) builds path := w.ShortcutPath(name) = filepath.Join(ShortcutsDir(), name+".md") (workspace.go:307-309) from raw user input (shortcuts.go:60 f.Pos[0], and cmdPromote at :123) with NO workspace.SafeSegment(name) check. Every sibling guards: CreateQueue (queue.go:39, comment 'reject traversal so it cannot escape .dacli, dacli 200') and removeObject/RemoveShortcut (remove.go:98-99). The shortcut CREATE path is the one hole, while shortcut REMOVE right beside it is guarded -- the inconsistency-between-neighbours tell. Repro: an rw agent runs 'dacli shortcut add ../../evil --command x --effect read'; the path resolves to <root>/evil.md, a file written OUTSIDE .dacli (more ../ escapes the repo). cmdAdd's only bar is RequireRW. The os.Stat existing-file check only blocks overwrite, not new-file creation at an arbitrary location; mdstore.WriteFile does os.MkdirAll(Dir(path)) then creates. Violates the documented containment invariant SafeSegment upholds (DESIGN.md:136, TRUST.md:33). Report says 'shortcut %q defined' -- a normal-write record while the file landed outside the boundary. Fix: add SafeSegment(name) guard at the top of CreateShortcut, mirroring CreateQueue:39. Verified against HEAD by direct read.
