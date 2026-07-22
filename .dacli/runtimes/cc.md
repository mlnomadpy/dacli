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
env_passthrough: [HOME, PATH, USER, LOGNAME, TMPDIR]
model_flag: --model
usage_format: stream-json
---
# cc
Flags here are assumptions until `dacli runtime doctor` verifies them against the installed binary.

Hand-corrected after run 01KY2K8N4C: `runtime add` mangled `--allowedTools`
into `true` because the flag parser reads a value starting with `--` as the
next flag (recorded as a finding on the core project). Read-only here means
read tools plus Bash scoped to the dacli binary — plan mode would block the
child from reporting at all.

ANTHROPIC_API_KEY removed 2026-07-21 on the owner's instruction: children
run as the user's own Claude Code login (keychain), never API billing. If
that variable leaked through, billing would silently flip to the API.
