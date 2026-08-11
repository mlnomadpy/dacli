---
id: f-a-comment-that-contradicts-the-code-beside-it-is-evidence-the-code-changed-not
kind: note
note_kind: finding
created: 2026-08-11T16:20:19Z
created_by: a-root
severity: moderate
scope: workspace
origin: internal/features/ghmirror/ghmirror.go
---
# A comment that contradicts the code beside it is evidence the code changed, not the comment
cmdPull carried a comment stating it was deliberately NOT Reject-guarded because 'github sync forwards push's flags through the same args, and pull must ignore them, not refuse'. Two lines above it, f.Reject("dry-run") did exactly what the comment said was not done - so 'github sync <proj> --since 2h' exited 2 at the inbound half and push never ran.

The gate was added later and the comment left behind. Reading the file was not enough to notice: the comment reads as the authority on the guard, and it was describing a state that no longer existed.

Two things follow. First: when a comment and the code beside it disagree, the comment is the stale one far more often, and the disagreement is itself a bug report worth acting on - it names a behaviour someone once needed. Second: the fix belonged at the seam, not in the guard. Widening pull's allowlist would have made a direct 'github pull --since 2h' silently ignore a flag the caller believed was applied - trading a loud bug for a quiet one. sync now tells pull which of push's flags to tolerate, so the tolerance lives on the only path that needs it.
