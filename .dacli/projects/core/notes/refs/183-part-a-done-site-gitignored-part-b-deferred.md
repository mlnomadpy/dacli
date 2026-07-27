---
id: r-183-part-a-done-site-gitignored-part-b-deferred
kind: note
note_kind: ref
created: 2026-07-27T23:03:55Z
created_by: a-root
about: "[[183]]"
---
# 183 part A done (site/ gitignored); part B deferred
site/ is gitignored. node_modules-in-go-build is local-dev-only (node_modules is gitignored, absent in CI/clean clones) and the nested-go.mod fix breaks the ui/dist go:embed, so it is left as-is.
