---
id: f-trailing-recursive-claims-were-compared-as-literal-star-segments
kind: note
note_kind: finding
created: 2026-08-16T18:36:18Z
created_by: a-maintainer-w9qqkt
about: "[[t-01KZYQ5E9PFVWRVMSWPB39E38K]]"
severity: major
---
# Trailing recursive claims were compared as literal star segments
internal/procmon/procmon.go PathsOverlap previously retained the terminal /** segment, so supabase/** did not overlap supabase/config.toml; internal/features/vcs/vcs.go delegates commit scope to that matcher. The pre-fix regression run refused both Supabase descendants alongside the genuinely outside scripts path.
