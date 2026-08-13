---
id: f-commit-wrapper-transiently-produced-malformed-root-attributed-commit
kind: note
note_kind: finding
created: 2026-08-13T15:58:04Z
created_by: a-fixer-f6typj
about: "[[423]]"
severity: major
---
# Commit wrapper transiently produced malformed root-attributed commit
The documented commit invocation returned exit 2 saying nothing staged, but HEAD became 9cf790e with subject -m, author a-root, and root provenance trailers even though whoami resolved a-fixer-f6typj. Recovery was git reset --soft HEAD^ followed by the same wrapper with DACLI_AGENT explicitly set, producing correctly attributed f174c05. Upstream dacli report was attempted once and failed because gh authentication is invalid; it was not retried.
