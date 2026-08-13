---
id: f-installed-gemini-stream-json-is-not-claude-compatible
kind: note
note_kind: finding
created: 2026-08-13T10:16:16Z
created_by: a-codex-maintainer-1d99qt
about: "[[405]]"
severity: major
---
# Installed Gemini stream-json is not Claude-compatible
Gemini CLI 0.41.1 advertises --approval-mode plan and --output-format stream-json but not Claude's --verbose; its terminal result carries stats.input_tokens/output_tokens and assistant output uses message.content. execution.go currently appends --verbose and parses only Claude result.
