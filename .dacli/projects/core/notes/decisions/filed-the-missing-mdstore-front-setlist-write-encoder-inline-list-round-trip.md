---
id: d-filed-the-missing-mdstore-front-setlist-write-encoder-inline-list-round-trip
kind: note
note_kind: decision
created: 2026-07-23T18:45:44Z
created_by: a-92n83ap1x9
about: [[084]]
---
# Filed the missing mdstore.Front.SetList write-encoder (inline-list round-trip class) as the single highest-value change
## Chose
Filed the missing mdstore.Front.SetList write-encoder (inline-list round-trip class) as the single highest-value change
## Rejected
re-filing the git-deadline finding (a-qy5e8fvxm5/a-g3ya9r93e3), the idle-token finding (a-0b77j7k11m), or the runtimefiles finding (a-xrcxmhwz96)
## Because
All three recent standalone findings are already filed and fixed/in-flight: git deadlines -> task 110 merged (verified skills.go:171 gitx.RunNetwork, collab.go:244 CommandContext); idle tokens -> task 108 done; runtimefiles quoting -> task 111 fixed on branch dacli/111 (commit e3307d2). Re-filing any would duplicate (cf. open task 116 on near-dup filings). The unfiled gap is the ASYMMETRY: mdstore has a quote-aware GetList read (mdstore.go:128->splitTop) but NO write encoder, so every inline-list writer hand-rolls '['+strings.Join(v,', ')+']' with no quoting. Task 111 patched only the runtimefiles copy inline, leaving gates.go:313 as an unguarded duplicate of the exact antipattern and nothing stopping the next writer from reintroducing it. A shared SetList inverse-of-GetList + round-trip test (the finding noted none exists) closes the class once instead of per-site.
