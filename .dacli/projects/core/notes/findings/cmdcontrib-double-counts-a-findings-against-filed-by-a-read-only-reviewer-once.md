---
id: f-cmdcontrib-double-counts-a-findings-against-filed-by-a-read-only-reviewer-once
kind: note
note_kind: finding
created: 2026-07-22T18:23:33Z
created_by: a-7sy0x8b84g
about: [[t-01KY5GP5RHAPGVT35FDFS4Z0PB]]
source_event: 01KY5H17X1ZJWNZKQSK0J80701
github:
  issue: 23
  repo: mlnomadpy/dacli
---
# cmdContrib double-counts a findings-against filed by a read-only reviewer (once as the event, again as its synced note)
internal/features/vcs/vcs.go cmdContrib — the defect-rate metric counts 'findings-against' twice for the primary review path. A read-only reviewer's 'note add finding --against X' is stored as an EventFinding (knowledge.go:40-50 AppendFinding, carrying against); an rw reviewer's creates a note directly (knowledge.go:57). cmdContrib's event loop (vcs.go:308-312) counts e.Against for every EventFinding returned by eventlog.List — which includes APPLIED events (eventlog.go:157-159 only filters when q.Pending, and this query sets no Pending). Then the notes loop (vcs.go:318-327) ALSO counts each NoteFinding's 'against'. After 'dacli sync' the ro reviewer's EventFinding becomes a NoteFinding note (sync.go:120-123 carries Against), so the SAME finding is counted in both loops => againstBy[X] is doubled. Net: ro-reviewer findings count 2x while rw-reviewer findings count 1x, so the per-role defect rate that drives 'which role to improve' (:345-360) over-reports inconsistently. Fix: dedupe (e.g. skip EventFinding events that have a synced note, or count notes only).
