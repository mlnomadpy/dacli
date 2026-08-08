---
id: f-cmdpr-records-the-pr-url-as-an-eventfinding-permanently-dragging-the-task-s
kind: note
note_kind: finding
created: 2026-07-22T18:23:33Z
created_by: a-7sy0x8b84g
about: [[t-01KY5GP5RHAPGVT35FDFS4Z0PB]]
source_event: 01KY5H0N6VK88MZM2WR43QTYWR
github:
  issue: 25
  repo: mlnomadpy/dacli
---
# cmdPR records the PR URL as an EventFinding, permanently dragging the task's brief trust-floor to unverified
internal/features/vcs/lifecycle.go:161 — after gh pr create, cmdPR does eventlog.Append(w, id.ID, model.EventFinding, t.ID, "", "PR opened: "+url). eventlog.Sync materializes any EventFinding into a durable NoteFinding note (sync.go:110-133). Consequences in brief.go 'What siblings found' (brief.go:225-272): (1) the PR-opened item (pending event, then synced note) is rendered as a finding and, being never graded, calls noteFloor("")/noteFloor(trust="") -> pulls the trust-floor to 'unverified' for EVERY future brief of the task, even when all real findings are confirmed — directly defeating D3's trust-floor signal. (2) It consumes one of the MillerCap=7 finding slots (brief.go:237), potentially evicting a real finding. (3) After sync the note body is empty (whole text is the level-1 title), so it renders as an empty '[trust: unverified] [[id]]' line — noise. A PR-opened operational fact is not a code defect; it should be an EventComment (or a dedicated note kind), not EventFinding.
