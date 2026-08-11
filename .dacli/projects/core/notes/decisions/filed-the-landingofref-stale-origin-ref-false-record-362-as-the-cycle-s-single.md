---
id: d-filed-the-landingofref-stale-origin-ref-false-record-362-as-the-cycle-s-single
kind: note
note_kind: decision
created: 2026-08-11T16:01:13Z
created_by: a-go-auditor-7sx6nh
about: "[[303]]"
---
# Filed the LandingOfRef stale-origin-ref false record (362) as the cycle's single highest-value item, over the shortcut SafeSegment gap and the github-sync flag refusal
## Chose
Filed the LandingOfRef stale-origin-ref false record (362) as the cycle's single highest-value item, over the shortcut SafeSegment gap and the github-sync flag refusal
## Rejected
Filing the CreateShortcut missing-SafeSegment path-traversal (a real #3 validation gap, but needs an rw grant + a crafted ../ name and is a loud-on-read containment breach) or the github sync --since refusal (a real bug but LOUD -- exit-2 usage error, trivial workaround: run pull/push separately) as the task instead
## Because
362 is the #1 audit class -- the record disagreeing with reality -- on the DEFAULT ship --push path: recordWaveLanding->LandingOfRef (landing.go:83-95) measures the captured branch sha against origin/<trunk>, which is provably stale at record time (local --no-ff merge done, push is a later step at ship.go:283), short-circuits on the first-resolving ref and never checks refs/heads/<trunk> that holds the merge, so a permanent, committed, false 'NOT in main - closed anyway' line lands on every task ship itself just merged. It is code-cited, reachable on the common origin-present config, and a residual of dacli 329 (which fixed the TIMING, not the ref-selection) so it survives the near-duplicate check as distinct scope. The other two I filed as findings (A major, B moderate) so they are not lost -- both are true and evidence-backed but lower-severity: A is gated behind rw and a deliberate name, B is a loud refusal with a trivial workaround.
