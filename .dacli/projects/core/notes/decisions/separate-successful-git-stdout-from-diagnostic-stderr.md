---
id: d-separate-successful-git-stdout-from-diagnostic-stderr
kind: note
note_kind: decision
created: 2026-08-12T18:25:54Z
created_by: a-codex-maintainer-f85g9w
about: "[[391]]"
github:
  issue: 514
  repo: mlnomadpy/dacli
---
# Separate successful git stdout from diagnostic stderr
## Chose
Separate successful git stdout from diagnostic stderr
## Rejected
Skip the restricted-platform fixture or force the child commit past claim enforcement
## Because
The transcript proves the worker started and dacli misparsed a git warning as a path; skipping would misclassify a coordination defect, while --force would stop the fixture from exercising the claim gate.
