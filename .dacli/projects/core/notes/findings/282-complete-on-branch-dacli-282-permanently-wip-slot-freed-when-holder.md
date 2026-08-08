---
id: f-282-complete-on-branch-dacli-282-permanently-wip-slot-freed-when-holder
kind: note
note_kind: finding
created: 2026-08-04T20:42:20Z
created_by: a-maintainer-4pjwbf
about: "[[282]]"
severity: major
---
# 282 complete on branch dacli/282-...-permanently — WIP slot freed when holder finished-but-never-retired
Commit 4cbc22d by a-maintainer-4pjwbf. Root cause: store.ActiveInRole treated any non-retired agent as active ('Agents have no liveness'), so a finished-but-never-retired agent pinned its role's WIP forever — a wip:1 role became permanently unspawnable (gateRoleWIP execution.go:463-471 and doctor's wip-exceeded both read this denominator).

Fix (5 files): (1) store/store.go adds holdsWIPSlot(recs,now) + runRecordsByChild(w) + wipGraceWindow=1h: an agent holds its slot while ANY run is a live process (procmon.AliveRecord), OR its newest run started within 1h, OR it has NO run record at all (absence of a run is not death). It is free only once every run is provably dead AND the newest is past the window. (2) store/roles.go: ActiveInRole now returns len(live) from new RoleWIPHolders(w,role)(live,gone []string), which also exposes the finished set. (3) insight.go doctor: new 'wip-stale-holders' detector names a role whose WIP is held only by gone agents.

Acceptance mapping: (1) no live process + nothing recent -> holdsWIPSlot false -> not counted [TestActiveInRoleExcludesFinishedAgent/CountsOnlyLive]; (2) doctor names such roles [TestDoctorFlagsStaleWIPHolders]; (3) liveness/recency probed instead of the retired flag, and a slot is never freed on mere absence of 'retired' — retired handled separately in RoleWIPHolders.

Proof: go build ./... clean; go test ./... green (41 ok, 0 FAIL); go vet clean; gofmt -l internal/ empty. Failing-before verified by temporarily restoring the old ActiveInRole body: the two store tests returned 1/2 instead of 0/1, then passed after the fix. PR-first off: owner accept 282 + merge/integrate --task 282.
