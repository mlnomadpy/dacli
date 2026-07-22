---
id: 01KY59QM9M7663NKC9X5WSF72B
kind: event
event_kind: finding
created: 2026-07-22T16:14:55Z
created_by: a-9y38s7w8e2
about: [[t-01KY59FNFK27A1084PQ8R2CJ5S]]
origin: agent
applied: true
---
governance slice docstring is stale: claims to stub subsystems that are all now built

governance.go:1-5 package docstring: 'holds the honestly-stubbed command surface for subsystems that are specified but not built: templates/gates, the GitHub mirror, skill compilation, verification panels.' All four of those are now BUILT and shipped: templates/gates in stagegate, the GitHub mirror in ghmirror (github push/link/doctor implemented), skill compilation in skillforge (skill compile implemented), verification panels in features/execution/verify.go. The only command actually left in governance.Commands is 'shortcut promote' (governance.go:10). The docstring overstates the unbuilt surface by 4 subsystems, misrepresenting the roadmap it claims to honestly track. Fix: narrow the docstring to the one remaining stub (shortcut promote), or fold it into the slice that owns the surface.
