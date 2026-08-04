---
id: rt-cc
kind: runtime
created: 2026-07-21T15:02:50Z
created_by: a-root
name: cc
binary: claude
invoke_mode: arg
invoke_flag: -p
sandbox_ro_args: ["--allowedTools", "Read,Grep,Glob,LS,Bash(/Users/tahabsn/go/bin/dacli:*)"]
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

The allowlisted dacli path is the INSTALLED binary (`go install
./cmd/dacli` → `~/go/bin/dacli`), never the repo's build output. It used to
point at `<repo>/dacli`, which sits inside the tree an rw child can write:
the child could overwrite the very binary the allowlist then let it execute
(dacli 165). Children must be able to run dacli — the whole using-dacli
contract is dacli commands — so the fix is to move the target out of the
writable tree, not to drop it from the list.

Residual, stated rather than hidden: `cc-rw` also allows `Bash(go:*)`, so a
determined child could `go build -o ~/go/bin/dacli`. Closing that needs the
binary at a path the agent's user cannot write at all (a root-owned
`/usr/local/bin/dacli`), which is an install the operator must perform.
Re-run `go install ./cmd/dacli` after pulling, or the allowlisted binary
drifts behind the source.
