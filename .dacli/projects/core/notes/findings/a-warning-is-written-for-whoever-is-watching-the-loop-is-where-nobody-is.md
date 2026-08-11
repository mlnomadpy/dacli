---
id: f-a-warning-is-written-for-whoever-is-watching-the-loop-is-where-nobody-is
kind: note
note_kind: finding
created: 2026-08-11T19:36:54Z
created_by: a-root
severity: major
scope: workspace
origin: internal/features/acceptance/acceptance.go
---
# A warning is written for whoever is watching; the loop is where nobody is
The landing check refused only under --require-verify and otherwise printed a warning to stderr. It closed a task four seconds after its PR opened, over work that was still not in main six days later (issue #443).

Two lessons, and the second is the reusable one.

1. In a tool whose whole point is autonomous operation, a warning on the default path is not a safety measure. It is a note to a reader who by construction is not there. Any guard whose failure mode is 'someone should notice this' belongs as a refusal with a named escape hatch, not as text on stderr. --allow-unlanded already existed and was the right shape all along; the default was simply on the wrong side of it.

2. Every existing test of that branch passed requireVerify=true, so the DEFAULT path had zero coverage and the suite stayed green whichever way the branch went. CONTRIBUTING.md already documents this exact shape from a prior occurrence ('a safety gate whose refusal branch was reachable by zero tests') and it recurred anyway, one flag over.

The generalisable check: for any guard with a strictness flag, ask which side of the flag the TESTS are on. If they all set the strict flag, the default is untested by construction, and the default is what production runs. Grep for a test file where every call passes the same boolean literal.
