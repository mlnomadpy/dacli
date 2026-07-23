---
id: t-01KY86EJ0K8QKB3Y5EGF7A4F9M
kind: task
created: 2026-07-23T19:15:15Z
created_by: a-z5aeq17erx
owner: a-z5aeq17erx
priority: must
---
# Windows cross-compile fails: procmon/execution use Unix-only syscalls (syscall.Kill/Setpgid) with no build tags, so goreleaser release aborts on the first v* tag
## So that
task 120's must-deliverable (cross-platform release incl. Windows binaries) currently cannot ship at all -- goreleaser fails the ENTIRE run when one goos target won't compile, so a v* tag publishes no artifacts and no Homebrew formula
## Acceptance
- [ ] GOOS=windows go build ./cmd/dacli succeeds (darwin+linux still build): the process-group code in internal/procmon/procmon.go (syscall.Kill at :141,200,297,307) and internal/features/execution/execution.go (syscall.SysProcAttr{Setpgid} at :917,965 and syscall.Kill at :971) is split into OS-tagged files (e.g. *_unix.go + *_windows.go via //go:build) with a Windows fallback (e.g. os.Process.Kill / taskkill /T /F) since syscall.Kill and SysProcAttr.Setpgid do not exist on Windows
- [ ] ci.yml gains a cross-compile gate: GOOS in {windows,darwin,linux} x GOARCH {amd64,arm64} 'go build ./cmd/dacli' all succeed, so this can never reach a release tag uncaught (CI is currently ubuntu/linux-only and never exercises the windows target goreleaser builds)
- [ ] goreleaser builds the windows target: 'goreleaser release --snapshot --clean' produces dacli_*_windows_amd64 and _arm64 archives without error
- [ ] go build ./... and go test ./internal/... remain green on the host platform
## Log
