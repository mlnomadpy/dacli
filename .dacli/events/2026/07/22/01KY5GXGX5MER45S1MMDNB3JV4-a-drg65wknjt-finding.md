---
id: 01KY5GXGX5MER45S1MMDNB3JV4
kind: event
event_kind: finding
created: 2026-07-22T18:20:28Z
created_by: a-drg65wknjt
about: [[t-01KY5GP5QJS16DCPAHQMTFBE5X]]
origin: agent
applied: true
---
dacli report attaches workspace name and raw transcript tail to the public upstream repo with no disclosure gate

selfreport.go cmdReport builds the issue body with the workspace name (selfreport.go:56) and, when --run is passed, a 30-line tail of the run's transcript.log (selfreport.go:58-61, runExcerpt:103-113), then files it to buildinfo.Repo — dacli's OWN (public) tracker — via gh issue create (selfreport.go:80). Unlike ghmirror, there is NO disclosure gate or redaction: a transcript tail can contain user-project content, paths, or secrets that then land on a public upstream issue. It is explicit (never automatic) and the agent chooses --run, but the leak surface is unguarded. Also, for detached stream-json runs the tail is raw JSON (sibling f-detached-stream-json-runs-write-raw-json), so the excerpt is barely readable. Fix: note the disclosure in the report path and/or scrub obvious secrets before attaching the excerpt.
