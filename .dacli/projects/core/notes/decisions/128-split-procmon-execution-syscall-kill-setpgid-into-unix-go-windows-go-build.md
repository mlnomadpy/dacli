---
id: d-128-split-procmon-execution-syscall-kill-setpgid-into-unix-go-windows-go-build
kind: note
note_kind: decision
created: 2026-07-23T19:39:16Z
created_by: a-z2xq5q3axy
about: [[128]]
---
# 128: split procmon/execution syscall.Kill+Setpgid into *_unix.go/*_windows.go build-tagged files
## Chose
128: split procmon/execution syscall.Kill+Setpgid into *_unix.go/*_windows.go build-tagged files
## Rejected
runtime.GOOS branches inside the shared functions
## Because
syscall.Kill and syscall.SysProcAttr.Setpgid are compile errors on GOOS=windows (the fields/functions don't exist for that GOOS), not runtime-only differences -- an if runtime.GOOS==... branch still fails to compile because both branches are type-checked for every GOOS. Only //go:build-tagged separate files exclude the unix-only syscall calls from the windows build unit. Windows fallback uses tasklist (Alive/GroupAlive) and taskkill /T /F (KillTree/killProcessGroup) since Windows has no signal-0 probe or POSIX process groups.
