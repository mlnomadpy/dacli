---
id: f-runtime-list-frontmatter-inline-list-encoding-splits-values-that-contain-a
kind: note
note_kind: finding
created: 2026-07-23T15:12:55Z
created_by: a-3s7f6bmn06
about: [[107]]
severity: minor
---
# runtime list frontmatter inline-list encoding splits values that contain a literal comma
internal/store/runtimefiles.go:85 setInline("sandbox_ro_args", rt.SandboxRO) and :116 GetList("sandbox_ro_args") round-trip []string through a comma-joined frontmatter scalar. A SandboxRO/Args/Env element that itself contains a comma (e.g. the claude-code preset's own --allowedTools value "Read,Grep,Glob,LS,Bash(dacli:*)") gets silently re-split into extra elements on load. Discovered while testing 107's ParseFlags fix with a --sandbox-ro-arg value containing a comma; unrelated to ParseFlags, not fixed here (out of scope) -- filed for a future task.
