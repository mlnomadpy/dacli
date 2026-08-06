---
id: d-guard-runstilllive-with-procmon-alive-rec-pid-to-split-recycled-leader-from
kind: note
note_kind: decision
created: 2026-08-04T20:12:29Z
created_by: a-maintainer-1e8nm5
about: "[[285]]"
---
# Guard runStillLive with procmon.Alive(rec.PID) to split recycled-leader from dead-leader before the GroupAlive fallback
## Chose
Guard runStillLive with procmon.Alive(rec.PID) to split recycled-leader from dead-leader before the GroupAlive fallback
## Rejected
Re-fingerprinting the pgid's members' start times against the record (a second, group-level identity mechanism)
## Because
The task scopes the fix to the ALIVE-recycled leader: when Alive(rec.PID) is true but AliveRecord is false, the PID was recycled and rec.PGID is a stranger's group, so returning false reuses the existing AliveIdentity leader guard with no new mechanism and no change to KillTree/liveAgents/cmdKill. Group-member fingerprinting was rejected because dacli-177's legitimate survivors (a mid-commit child) have unrelated start times by design, so there is no leader identity left to check once the leader PID is genuinely gone — that dead-leader/reused-pgid residual is explicitly left for a follow-up.
