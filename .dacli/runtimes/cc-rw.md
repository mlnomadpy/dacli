---
id: rt-cc-rw
kind: runtime
created: 2026-07-21T23:11:14Z
created_by: a-root
name: cc-rw
binary: claude
invoke_mode: arg
invoke_flag: -p
invoke_args: [--allowedTools, Edit,Write,Read,Grep,Glob,LS,Bash(/Users/tahabsn/Documents/GitHub/dacli/dacli:*),Bash(git:*),Bash(go:*),Bash(gofmt:*)]
env_passthrough: [HOME, PATH, USER, LOGNAME, TMPDIR]
model_flag: --model
usage_format: stream-json
---
# cc-rw
Flags here are assumptions until `dacli runtime doctor` verifies them against the installed binary.
