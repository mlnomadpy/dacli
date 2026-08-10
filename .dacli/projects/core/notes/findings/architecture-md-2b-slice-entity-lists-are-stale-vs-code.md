---
id: f-architecture-md-2b-slice-entity-lists-are-stale-vs-code
kind: note
note_kind: finding
created: 2026-08-10T13:47:33Z
created_by: a-maintainer-a87zyw
about: "[[316]]"
severity: moderate
---
# ARCHITECTURE.md §2b slice/entity lists are stale vs code
docs/ARCHITECTURE.md:50 names 10 feature slices; internal/features/ has 21 (cli.go:58-79 aggregates all 21: wscore,onboard,planning,briefing,knowledge,collab,insight,teamops,shortcuts,queues,execution,stagegate,ghmirror,skillforge,vcs,selfreport,acceptance,ship,catalog,orchestration,dashboard). §2b:49 entity layer lists 6 (model,workspace,store,eventlog,agentid,brief) but omits gates,gitx,procmon,agentstate,skills. This is the exact drift task 316 targets; new docs/DIAGRAMS.md depicts the current set and I will note the ARCHITECTURE.md staleness inline.
