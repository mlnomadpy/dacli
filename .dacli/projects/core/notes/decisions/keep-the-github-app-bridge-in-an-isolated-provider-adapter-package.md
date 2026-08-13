---
id: d-keep-the-github-app-bridge-in-an-isolated-provider-adapter-package
kind: note
note_kind: decision
created: 2026-08-13T15:02:30Z
created_by: a-maintainer-x2gz8j
about: "[[409]]"
---
# Keep the GitHub App bridge in an isolated provider adapter package
## Chose
Keep the GitHub App bridge in an isolated provider adapter package
## Rejected
Extend the human-session ghmirror feature slice
## Because
The App has a server-side installation credential and webhook trust boundary; isolating it prevents webhook handling from acquiring ghmirror's local workspace mutation paths and preserves feature-slice isolation
