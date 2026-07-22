---
id: 01KY3ETTPM8801Z90JYGFYQ5Q4
kind: event
event_kind: finding
created: 2026-07-21T23:05:34Z
created_by: a-hp8fwzbck0
about: [[t-01KY3EKR1MSTD09QSJGSW6RSTM]]
origin: agent
applied: true
---
Embedded, immutable templates re-read+re-parsed on every call (prompts.MCPDesc, gates.Get) and skill dirs scanned twice per load

prompts.go:84 MCPDesc re-reads embedded tpl/mcp_tools.md and full mdstore.Parse()s it on EVERY call — when the MCP server registers N tools it parses the whole doc N times; hoist behind sync.Once into a parsed map. prompts.go:41 Render re-parses the template text each call (content-keyed compiled-template cache would remove it if ever looped). gates.go:110 Get→Load parses all embedded templates + every workspace .md per call; gates.go:263 Advance also calls store.LoadProject twice (Status already loaded it at :230). skills.go:104+134 load() does os.ReadDir(dir) twice per skill (mainFile discards the entry list, then load re-reads the same dir) — double the directory syscalls across the whole library scan; have mainFile return the already-read entries.
