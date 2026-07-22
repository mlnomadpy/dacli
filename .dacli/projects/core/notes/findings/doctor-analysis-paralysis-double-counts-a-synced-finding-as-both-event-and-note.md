---
id: f-doctor-analysis-paralysis-double-counts-a-synced-finding-as-both-event-and-note
kind: note
note_kind: finding
created: 2026-07-22T18:23:33Z
created_by: a-1zhjz6t2va
about: [[t-01KY5GP5S0V2MFQC2R2EFVEA5A]]
source_event: 01KY5GY2CBFWZ42EZ7T13XG65S
github:
  issue: 29
  repo: mlnomadpy/dacli
---
# doctor analysis-paralysis double-counts a synced finding as both event and note
internal/features/insight/insight.go:888,:902: cmdDoctor computes findings via eventlog.List(Query{Kinds:[EventFinding]}) with NO Pending filter, so it returns ALL finding events including applied ones; noteFindings sums store.ListNotes(NoteFinding). But eventlog/sync.go:110-120 materializes each EventFinding into a NoteFinding on sync while the event REMAINS in the log (applied=true). So after 'dacli sync', one finding is counted twice (event + note) in 'len(findings)+noteFindings >= 5 && done == 0', making the analysis-paralysis anti-pattern fire at ~2.5 real findings instead of 5. Count only pending finding events (Pending:true) plus notes, or count notes alone, to avoid the applied-event/note overlap.
