---
id: d-marker-must-lead-its-comment-not-just-appear-as-a-standalone-token-anywhere-in
kind: note
note_kind: decision
created: 2026-07-23T00:06:06Z
created_by: a-bndgc6d73j
about: [[087]]
---
# marker must lead its comment, not just appear as a standalone token anywhere in it
## Chose
marker must lead its comment, not just appear as a standalone token anywhere in it
## Rejected
match any standalone (word-boundary) TODO/FIXME/HACK/XXX token found anywhere inside a real (string-literal-aware) comment
## Because
onboard.go's own doc comment mentions 'the TODO markers' and a header comment literally began with 'TODO markers - cheap scan...' - both are prose about the feature, not actionable items, and a plain standalone-token-anywhere rule would still self-match them; requiring the marker to LEAD the trimmed comment (as in the real // TODO: handle x / # FIXME ... convention) excludes prose mentions while still catching genuine markers, and the one header comment that did lead with 'TODO' was reworded to 'Scan for TODO/FIXME/HACK/XXX markers' since it was describing the feature, not flagging a to-do
