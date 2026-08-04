---
id: 01KZ66M4N99QNYZVCSGAZ0GHSC
kind: event
event_kind: commit
created: 2026-08-04T10:55:31Z
created_by: a-root
origin: agent
applied: false
---
d03cb09 205 review: make push refuse a truncated index instead of warning after the damage

Review on the PR: three surfaces read the same --limit cap and took
three different positions on it. listIssues (pull) refuses. itemSnapshot
(project, this PR) refuses. markerIndex (push) warned — at the END of
cmdPush, by which point every issue past the fetched page had already
been re-created as a duplicate, because none of them was in the index to
be adopted. Push is the only one of the three that writes to a live
repository, and it had the weakest policy.

The truncation is knowable before the first write: load runs on the
first find(), find() is the first idempotency check, and that check
precedes the first create. preflight() forces the load and refuses
there, so it costs no extra fetch.

A fetch FAILURE is deliberately still not an error. find() is fail-soft
by design after 208 — a transient gh error leaves the index unloaded so
a later find() retries. Truncation is the opposite case: the fetch
succeeded and what it returned is a confident wrong answer. Tested both
ways, plus that preflight reuses the single snapshot.

find()'s lazy fetch is now a named load(), so preflight does not have to
call find("") — which would match every body — to trigger it.
role: root
