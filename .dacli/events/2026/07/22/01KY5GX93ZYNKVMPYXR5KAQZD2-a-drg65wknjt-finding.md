---
id: 01KY5GX93ZYNKVMPYXR5KAQZD2
kind: event
event_kind: finding
created: 2026-07-22T18:20:20Z
created_by: a-drg65wknjt
about: [[t-01KY5GP5QJS16DCPAHQMTFBE5X]]
origin: agent
applied: true
---
findingAboutTask matches the task by loose substring, missing unpadded refs and risking cross-matches

ghmirror.go:406-409 findingAboutTask does strings.Contains(about, t.ID) || strings.Contains(about, fmt.Sprintf('%03d', t.Seq)). RW-filed finding notes store about verbatim as '[[<ref>]]' (store.go:766; the CreateNote path at knowledge.go:57-58 passes --about raw, unlike the ro EVENT path at knowledge.go:44-48 which resolves to t.ID). So a note filed '--about 61' becomes '[[61]]' and never matches task seq 61 (padded '061' is not a substring of '[[61]]', and t.ID differs) — the finding is silently NOT mirrored. The %03d substring is also loose: a note about a 4-digit seq whose zero-padded form contains a shorter task's 3-digit code could mis-match. Same fragile check is duplicated in verify.go:219 and vcs/lifecycle.go. Fix: resolve the about ref to the canonical task id at match time (store.FindTask) instead of substring-matching a formatted seq.
