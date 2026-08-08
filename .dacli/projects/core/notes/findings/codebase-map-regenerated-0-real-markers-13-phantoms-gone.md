---
id: f-codebase-map-regenerated-0-real-markers-13-phantoms-gone
kind: note
note_kind: finding
created: 2026-07-23T13:16:17Z
created_by: a-rd1mwxdxpf
about: [[101]]
severity: moderate
---
# Codebase map regenerated: 0 real markers, 13 phantoms gone
Ran 'dacli adopt --project core' to re-run the 087-fixed scanner (internal/features/onboard/onboard.go) against the live tree. Result: 'codebase map written (152 files, 3 languages)' and the resulting .dacli/projects/core/project.md now has NO 'Open markers' section at all (renderMap only emits it when len(r.todos)>0, onboard.go:359) — meaning the fixed commentText()/quote-tracking scan found zero real leading-comment TODO/FIXME/HACK/XXX markers anywhere in the tree. Verified independently: grep -rn '^\s*//\s*(TODO|FIXME|HACK|XXX)\b' --include='*.go' internal/ cmd/ (excluding _test.go) returns nothing. Manually inspected the two named false-positive sites: onboard.go's todoMarkers loop (was line 198, now ~line 243, a []string literal) and gates.go:448's retro-check string slice ("FIXME", "{{", "...") — both are string-literal array elements the quote-tracking commentText() correctly skips. Confirmed via 'dacli context 101': the regenerated brief's Codebase map section carries no Open markers block at all. Both acceptance criteria are satisfied by this regeneration; no source code change was needed (087 already fixed the scanner) — this task was pure data regeneration.
