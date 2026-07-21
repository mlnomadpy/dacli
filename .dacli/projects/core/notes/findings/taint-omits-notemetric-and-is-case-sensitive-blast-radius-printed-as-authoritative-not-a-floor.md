---
id: f-taint-omits-notemetric-and-is-case-sensitive-blast-radius-printed-as-authoritative-not-a-floor
kind: note
note_kind: finding
created: 2026-07-21T21:31:33Z
created_by: a-root
about: [[006]]
severity: moderate
---
# taint omits NoteMetric and is case-sensitive; blast radius printed as authoritative not a floor
REVIEWER: (2) loop omits model.NoteMetric though note add metric --origin is supported. (3) strings.Contains case-sensitive + no path normalization: file:Configs/Evil.yml evades file:configs/evil.yml. (4) origin self-reported; unlabeled artifacts carry 'agent' and are invisible — output should say lower bound. (5) applied events double-count with their synced notes; exposed briefs mix ULID and slug labels for the same task.
