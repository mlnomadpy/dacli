---
id: d-filed-re-integration-of-task-127-s-mdstore-quotelistelem-escaping-fix-as-the
kind: note
note_kind: decision
created: 2026-07-26T21:13:25Z
created_by: a-c5y9g9f56d
about: [[084]]
---
# Filed re-integration of task 127's mdstore quoteListElem escaping fix as the single highest-value evidence-based change
## Chose
Filed re-integration of task 127's mdstore quoteListElem escaping fix as the single highest-value evidence-based change
## Rejected
Filing a fresh code review nit, re-filing the burn population/DAG issues (already tasks 149/150/153), or acting on task 157 (its premise is false: 154's CI change IS on main, 6e142c9 is an ancestor of HEAD)
## Because
quoteListElem on main (internal/mdstore/mdstore.go:153-161) still uses the pre-127 single/double-quote heuristic with NO escaping, so an element carrying both quote chars plus a comma (e.g. it's "a,b") encodes to "it's "a,b"" and GetList splits it into two corrupted entries -- a concrete data-corruption round-trip bug in the core persistence primitive. Task 127 (marked DONE, finding says complete on branch 37c691b) fixed it with a tested unescapeDouble inverse, but git branch --no-merged shows dacli/127 is NOT an ancestor of HEAD: the accepted+closed work was orphaned and never landed. Open task 136 would route 5 MORE raw writers through this same buggy encoder, amplifying the corruption, so re-landing 127 is a prerequisite. This outranks task-157 (false premise) and the already-filed burn/DAG defects.
