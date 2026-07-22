---
id: f-swallowed-i-o-errors-readdir-failures-reported-as-empty-run-record-writes-discarded-writefile-orphans-its-temp-file-on-rename-failure
kind: note
note_kind: finding
created: 2026-07-21T23:09:25Z
created_by: a-hp8fwzbck0
about: [[t-01KY3EKR1MSTD09QSJGSW6RSTM]]
---
# Swallowed I/O errors: ReadDir failures reported as empty, run-record writes discarded, WriteFile orphans its temp file on rename failure
ListNotes (store.go:493), ListRisks (risk.go:69), LoadRoles (roles.go:74), LoadRuntimes (runtimefiles.go:98), LoadShortcuts (shortcutfiles.go:53), ListQueues (queue.go:104) all return nil,nil on ANY os.ReadDir error, conflating "dir does not exist yet" with a real I/O/permission error — a genuinely unreadable notes dir silently presents as "no notes", so taint/lessons walks miss findings rather than failing. ListProjects (store.go:125) does it right (checks os.IsNotExist first); the others should follow. execution.go:257 writeRun (and 448-455, verify.go:120-133) do _=os.WriteFile for the replay-capture brief/invocation/outcome — a failed write is swallowed and later surfaces as "brief not recorded" with no hint a write failed. mdstore.go:471 WriteFile removes its temp file on the write/close error paths but NOT when os.Rename fails, so failed renames leak .dacli-tmp-* files into object dirs.
