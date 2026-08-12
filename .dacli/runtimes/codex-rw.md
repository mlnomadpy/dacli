---
id: rt-codex-rw
kind: runtime
created: 2026-08-12T13:07:41Z
created_by: a-root
name: codex-rw
binary: /Applications/ChatGPT.app/Contents/Resources/codex
invoke_mode: stdin
invoke_args: "[--ask-for-approval, never, exec, --ignore-user-config, --disable, plugins, --disable, plugin_sharing, --disable, remote_plugin, --sandbox, workspace-write, --add-dir, /Users/tahabsn/Documents/GitHub/dacli/.git, --add-dir, /Users/tahabsn/Documents/GitHub/dacli/.dacli, --ephemeral, --color, never, --json]"
env_passthrough: "[HOME, PATH, USER, LOGNAME, TMPDIR]"
model_flag: --model
---
# codex-rw
Flags here are assumptions until `dacli runtime doctor` verifies them against the installed binary.

Bootstrap adapter for task 371. JSONL mode supplies an explicit terminal event
and exited cleanly after repository tool use; the formatted mode repeatedly
left the CLI/helper tree alive after its apparent final answer. Plugins are
disabled to keep unattended runs independent of operator extensions. Replace
this hand-authored adapter with the shipped preset once task 371 lands.

The extra writable directories are the shared linked-worktree Git index and
dacli event store. Code remains confined to the task worktree; these two paths
allow the claimed worker to commit through `dacli commit` and propose task
events for the owner to sync.
