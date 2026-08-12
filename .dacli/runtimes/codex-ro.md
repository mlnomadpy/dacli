---
id: rt-codex-ro
kind: runtime
created: 2026-08-12T13:07:41Z
created_by: a-root
name: codex-ro
binary: /Applications/ChatGPT.app/Contents/Resources/codex
invoke_mode: stdin
invoke_args: "[--ask-for-approval, never, exec, --ignore-user-config, --disable, plugins, --disable, plugin_sharing, --disable, remote_plugin, --ephemeral, --color, never, --json]"
sandbox_ro_args: "[--sandbox, read-only]"
env_passthrough: "[HOME, PATH, USER, LOGNAME, TMPDIR]"
model_flag: --model
---
# codex-ro
Flags here are assumptions until `dacli runtime doctor` verifies them against the installed binary.

Bootstrap adapter for task 371. It remains ineligible for strict `ro` work
until a local behavioral probe verifies Codex's sandbox.
