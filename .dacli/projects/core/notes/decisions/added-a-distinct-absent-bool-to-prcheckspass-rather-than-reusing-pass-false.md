---
id: d-added-a-distinct-absent-bool-to-prcheckspass-rather-than-reusing-pass-false
kind: note
note_kind: decision
created: 2026-08-04T00:10:23Z
created_by: a-fixer-2675tw
about: "[[216]]"
---
# added a distinct absent bool to prChecksPass rather than reusing pass=false
## Chose
added a distinct absent bool to prChecksPass rather than reusing pass=false
## Rejected
leaving prChecksPass's 3-value signature (pass, detail, netErr) alone and just returning pass=false with detail='no checks reported', reusing the existing checks-not-passing message path
## Because
the acceptance criterion is that absent checks are DISTINGUISHABLE from passing checks, not merely that they don't merge; collapsing 'no CI configured' into the same pass=false branch as 'red/pending check' would have produced an identical 'checks not passing ... merge once CI is green' message for a repo that has no CI to go green, which is not distinguishable to the operator reading integrate's output. A fourth absent return value lets the caller print a message that names the actual condition (no checks configured) and points at the real remedies (merge manually, set up CI, or use --auto/--no-merge) instead of telling someone to wait on CI that doesn't exist.
