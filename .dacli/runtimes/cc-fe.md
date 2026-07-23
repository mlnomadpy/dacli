---
id: rt-cc-fe
kind: runtime
created: 2026-07-23T22:40:00Z
created_by: a-root
name: cc-fe
binary: claude
invoke_mode: arg
invoke_flag: -p
invoke_args: [--allowedTools, Edit,Write,Read,Grep,Glob,LS,Bash(/Users/tahabsn/Documents/GitHub/dacli/dacli:*),Bash(git:*),Bash(npm:*),Bash(npx:*),Bash(node:*),Bash(pnpm:*),Bash(go:*),Bash(gofmt:*)]
env_passthrough: [HOME, PATH, USER, LOGNAME, TMPDIR]
model_flag: --model
usage_format: stream-json
---
# cc-fe
Frontend runtime: claude-code with node/npm/pnpm/npx allowed (plus git, go/gofmt for the embed step, and the dacli binary for commit/pr) so an agent can install deps, run Vite builds, and test the Vue dashboard. Read-write; verify with `dacli runtime doctor`.
