---
id: 01KY59T2D9YDAH289R2WT5HBBK
kind: event
event_kind: finding
created: 2026-07-22T16:16:15Z
created_by: a-0htd7wjqyt
about: [[t-01KY59FNFAEE0KT7PWV8HAAY4A]]
origin: agent
applied: true
---
eventlog.List silently drops malformed or unreadable events

internal/eventlog/eventlog.go: List ignores the WalkDir error (:100 '_ = filepath.WalkDir') and, per file, 'continue's on any mdstore.ReadFile error (:127-130) with no counter, log, or return. The package doc (:1-11) sells the event log as append-only and lossless — 'work not reported does not exist' — but a single corrupted/half-written/unparseable event .md just VANISHES: it never appears in List output AND it never applies in Sync (Sync only iterates List's results), with zero signal that anything was skipped. A partial write that mdstore.Parse rejects (e.g. 'unterminated frontmatter') thus silently erases a claim/finding/propose from the durable log. Same silent-skip class as sibling [[f-swallowed-i-o-errors-readdir-failures-reported-as-empty-run-record-writes-discarded-writefile-orphans-its-temp-file-on-rename-failure]] but this is the eventlog instance, which is the write path's own integrity guarantee. Fix: count/surface parse failures (e.g. a doctor check flagging unreadable event files) rather than dropping them.
