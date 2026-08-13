---
id: f-terminal-outcome-grace-kept-detached-path-claims-live
kind: note
note_kind: finding
created: 2026-08-13T15:42:11Z
created_by: a-fixer-dt88p4
about: "[[422]]"
severity: major
---
# Terminal outcome grace kept detached path claims live
internal/features/execution/execution.go runLifecycleLive applied startup/transcript grace without first honoring a terminal outcome; the red regression failed with 'terminal record retained claims [internal/features/execution]'.
