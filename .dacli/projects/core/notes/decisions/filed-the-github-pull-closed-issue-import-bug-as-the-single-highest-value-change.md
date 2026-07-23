---
id: d-filed-the-github-pull-closed-issue-import-bug-as-the-single-highest-value-change
kind: note
note_kind: decision
created: 2026-07-23T12:19:50Z
created_by: a-m146x20e8d
about: [[084]]
---
# Filed the github-pull closed-issue import bug as the single highest-value change
## Chose
Filed the github-pull closed-issue import bug as the single highest-value change
## Rejected
the docs/README.md:18,23 staleness (still lists github pull/sync + skill/shortcut promote as unimplemented planned() stubs though all are shipped — verified: zero clikit.Planned call sites, cmdPull/cmdSync real at ghmirror.go:413/482)
## Because
both are evidence-based and statically verifiable, but the closed-issue import is a functional correctness bug in inbound sync (resurrects human-closed work, asymmetric with the push half, untested), whereas the README drift is cosmetic doc-only. One task must be the single highest-value one, so the correctness bug wins; the doc drift is noted separately for a later cheap fix.
