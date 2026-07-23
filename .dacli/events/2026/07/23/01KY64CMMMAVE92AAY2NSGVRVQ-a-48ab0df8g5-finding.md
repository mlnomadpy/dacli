---
id: 01KY64CMMMAVE92AAY2NSGVRVQ
kind: event
event_kind: finding
created: 2026-07-23T00:00:46Z
created_by: a-48ab0df8g5
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
onboard TODO-scanner matches marker names as bare substrings in string literals and comments, polluting every brief's codebase map

internal/features/onboard/onboard.go:250 scanTodos uses strings.Index(line, marker) to match TODO/FIXME/HACK/XXX anywhere on a line, with no word-boundary or comment check. This matches the marker names inside Go string literals and descriptive comments in dacli's OWN source: onboard.go:249 []string{"TODO","FIXME","HACK","XXX"}, gates.go:465 []string{"TBD","TODO","FIXME","{{","..."}, the command Brief at onboard.go:30, and onboard_test.go:16-18,52,55 fixture/assertion strings. Observable evidence: this session's own brief lists 'Open markers (13)' under ## Codebase map, and the large majority are these self-referential false positives (e.g. 'TODO internal/gates/gates.go:448', 'TODO internal/features/onboard/onboard.go:30 — tasks", Run: cmdAdopt},'), not real open work. The codebase map is written onto the project by adopt and rides into every brief, so the noise misleads every agent; worse, adopt --todos (onboard.go:104-122) would seed FALSE tasks from these matches. Filed task 087. Fix: require the marker to be a standalone token in a comment (word boundary + comment introducer), not any substring.
