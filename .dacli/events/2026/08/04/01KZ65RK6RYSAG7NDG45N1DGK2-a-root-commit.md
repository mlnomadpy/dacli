---
id: 01KZ65RK6RYSAG7NDG45N1DGK2
kind: event
event_kind: commit
created: 2026-08-04T10:40:28Z
created_by: a-root
about: "[[t-01KZ53KN27Y0WCPV4G9HNSC79W]]"
origin: agent
applied: true
---
2e1a28d 247: give the task-seq lock an owner, and steal only from a dead one

The seq lock's steal was ownerless. Past the deadline any waiter removed
whatever file was present and continued, so two waiters could both
believe they held it — which is 209's duplicate-seq bug reintroduced
through the lock that was supposed to prevent it. Release compounded it
by removing the lock by path, so an agent that had been stolen from
deleted the CURRENT holder's lock on its way out.

The lock file now carries {pid, pid_start, host, token, ts}. Release
removes a file only while it still carries its own token, via a
rename-aside-verify-relink that cannot overwrite a successor's claim. A
steal fires only against a demonstrably dead holder: an
identity-checked pid on this host (pid_start, so a recycled pid is not
mistaken for the holder), or a 60s-old lock from another host where no
probe is possible. An unreadable or partially written file counts as
live. Waiters back off 2ms to 100ms and get an error naming the holder
when the deadline passes — waiting never confers the lock.

Stealers serialize on their own O_EXCL guard. The first cut skipped it
and the 20-way concurrent test caught why: losers of the rename race
move aside a SUCCESSOR's live lock, acting on a decision about a file
that was already gone. The guard was earned by that failure.

Assumptions are in the doc comment: O_EXCL/rename/link atomic on the
workspace filesystem (local, NFSv3+); pid liveness consulted only for
same-host locks; cross-host clock skew charged against the age test.

Written test-first. Before the fix the new tests failed five distinct
ways, including 'seq 4 allocated to both concurrent-task-0 and
concurrent-task-3'. After: go test ./... clean, -race -count=5 on
internal/store green, gofmt and vet clean.

Implemented by a-maintainer (seq-lock task 247); committed from the
coordinator session because branch creation was denied in the agent's
own sandbox.
role: root
