---
id: f-agents-status-read-finalizes-a-live-guardian-after-identity-probe-false-negative
kind: note
note_kind: finding
created: 2026-08-12T18:25:31Z
created_by: a-codex-maintainer-j8jbvt
about: "[[382]]"
severity: major
---
# agents status read finalizes a live guardian after identity-probe false negative
internal/features/execution/execution.go:2153 calls sweepFinishedDetached from cmdAgents; regression TestStatusReadsDoNotFinalizeALiveRunWhenProcessIdentityIsHidden fails because a live os.Getpid guardian with an unobservable PIDStart is rewritten to outcome: no visible result.
