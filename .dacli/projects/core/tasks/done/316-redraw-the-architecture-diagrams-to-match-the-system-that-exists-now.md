---
id: t-01KZNYJP1YYB8W3K5MTKG28QD0
kind: task
created: 2026-08-10T13:42:46Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Redraw the architecture diagrams to match the system that exists now
## So that
a reader onboarding from the docs sees today's dispatcher gates, event channels and record branch rather than the design before them
## Acceptance
- [x] diagrams are committed as text (mermaid) in docs/ so they diff in review and cannot drift silently from the prose beside them
- [x] the set covers structure, one end-to-end flow, and the state a task moves through — a component view, a sequence for spawn through landing, and a task lifecycle
- [x] each diagram is checked against the code it depicts, naming the file that implements each edge
## Log
- 2026-08-10T13:43:22Z claimed by a-maintainer-a87zyw
- 2026-08-10T14:26:58Z accepted by a-root (applied 1 proposal(s))
- 2026-08-10T14:26:58Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T14:26:58Z deliverable: dacli/316-redraw-the-architecture-diagrams-to-match-the-system-that-exists-now exists but is NOT in trunk — closed anyway
- 2026-08-10T14:26:58Z completed by a-root
