---
id: d-123-gated-color-on-ctx-stdout-being-an-os-file-character-device-clikit-palette
kind: note
note_kind: decision
created: 2026-07-23T19:53:43Z
created_by: a-fgmsw5w6rd
about: [[123]]
---
# 123: gated color on ctx.Stdout being an *os.File character device (clikit.Palette), not a separate agent-detection flag
## Chose
123: gated color on ctx.Stdout being an *os.File character device (clikit.Palette), not a separate agent-detection flag
## Rejected
a --color/--no-color flag or an explicit MCP-mode bit threaded through Ctx
## Because
the MCP executor and every test harness already write to a bytes.Buffer, never an *os.File; checking isatty on Stdout is the one mechanism that keeps ANSI codes out of agent-facing and test output for free, with zero new plumbing and zero risk of a flag being forgotten on a new front end
