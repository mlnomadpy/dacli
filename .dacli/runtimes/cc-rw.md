---
id: rt-cc-rw
kind: runtime
created: 2026-07-21T23:11:14Z
created_by: a-root
name: cc-rw
binary: claude
invoke_mode: arg
invoke_flag: -p
invoke_args: [--allowedTools, Edit,Write,Read,Grep,Glob,LS,Bash(/Users/tahabsn/go/bin/dacli:*),Bash(git:*),Bash(go:*),Bash(gofmt:*)]
env_passthrough: [HOME, PATH, USER, LOGNAME, TMPDIR]
model_flag: --model
usage_format: stream-json
---
# cc-rw
Flags here are assumptions until `dacli runtime doctor` verifies them against the installed binary.

The allowlisted dacli path is the INSTALLED binary (`go install ./cmd/dacli`
→ `~/go/bin/dacli`), never the repo's build output. It used to point at
`<repo>/dacli`, which sits inside the tree this runtime hands the child
`Edit` and `Write` over: the child could overwrite the very binary the
allowlist then let it execute (dacli 165). Children must be able to run
dacli — the whole using-dacli contract is dacli commands — so the fix is to
move the target out of the writable tree, not to drop it from the list.

Residual, stated rather than hidden: `Bash(go:*)` is on this list, so a
determined child could `go build -o ~/go/bin/dacli`. Closing that needs the
binary at a path the agent's user cannot write at all (a root-owned
`/usr/local/bin/dacli`), which is an install the operator must perform.
Re-run `go install ./cmd/dacli` after pulling, or the allowlisted binary
drifts behind the source.
