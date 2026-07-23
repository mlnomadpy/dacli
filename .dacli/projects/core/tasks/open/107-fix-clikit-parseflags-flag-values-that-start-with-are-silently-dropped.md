---
id: t-01KY7KEVWRSKFRV4ZN9S4JTYDE
kind: task
created: 2026-07-23T13:43:22Z
created_by: a-y4c733ze1d
owner: a-y4c733ze1d
priority: should
---
# Fix clikit ParseFlags: flag values that start with '-' are silently dropped, corrupting spawn/runtime argv
## So that
runtime adapters and spawn argv stop silently sending garbage to the vendor CLI (risk #1), which already happened in run 01KY2K8N4C
## Acceptance
- [ ] clikit.ParseFlags no longer silently records '--key --value' (value beginning with '-') as key=true; either the value is captured or the command fails with a clear error naming the accepted mechanism (=form or -- terminator)
- [ ] runtime add value-flags (--flag, --arg, --sandbox-ro-arg, --model-flag) accept a value that begins with '-' without dropping it; a regression test in internal/clikit covers a value starting with '-'
- [ ] the runtime-adapter argv build path is exercised by a test proving the corruption from run 01KY2K8N4C (e.g. --sandbox-ro-arg --allowedTools) cannot recur silently
- [ ] usage-string workarounds that warn 'values that start with -- need the = form' (execution.go:83 and the runtime-add usage) are updated to match the chosen fix
## Log
