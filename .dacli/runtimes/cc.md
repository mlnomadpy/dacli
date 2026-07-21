---
id: rt-cc
kind: runtime
created: 2026-07-21T15:02:50Z
created_by: a-root
name: cc
binary: claude
invoke_mode: arg
invoke_flag: -p
sandbox_ro_args: ["--allowedTools", "Read,Grep,Glob,LS,Bash(/Users/tahabsn/Documents/GitHub/dacli/dacli:*)"]
env_passthrough: [HOME, PATH, ANTHROPIC_API_KEY, USER, LOGNAME, TMPDIR]
---
# cc
Flags here are assumptions until `dacli runtime doctor` verifies them against the installed binary.

Hand-corrected after run 01KY2K8N4C: `runtime add` mangled `--allowedTools`
into `true` because the flag parser reads a value starting with `--` as the
next flag (recorded as a finding on the core project). Read-only here means
read tools plus Bash scoped to the dacli binary — plan mode would block the
child from reporting at all.
