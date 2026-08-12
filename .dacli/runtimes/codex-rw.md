---
id: rt-codex-rw
kind: runtime
created: 2026-08-12T13:07:41Z
created_by: a-root
name: codex-rw
binary: /Applications/ChatGPT.app/Contents/Resources/codex
invoke_mode: stdin
invoke_args: "[--ask-for-approval, never, exec, --ignore-user-config, --sandbox, workspace-write, --ephemeral, --color, never]"
env_passthrough: "[HOME, PATH, USER, LOGNAME, TMPDIR]"
model_flag: --model
---
# codex-rw
Flags here are assumptions until `dacli runtime doctor` verifies them against the installed binary.

Bootstrap adapter for task 371. `--ignore-user-config` keeps unattended dacli
runs independent of operator plugins and hooks; Codex authentication still
uses `CODEX_HOME`. Replace this hand-authored adapter with the shipped preset
once task 371 lands.
