---
id: 01KZRRVW9MWWXRXDQZ8T15TDPD
kind: event
event_kind: finding
created: 2026-08-11T16:00:39Z
created_by: a-go-auditor-7sx6nh
about: "[[t-01KZ93DW7P6BQ1HHDWY7MEH2KJ]]"
origin: agent
applied: false
---
github sync with any push-only flag (--since/--findings-as-issues/--with-tasks) refuses instead of ignoring, so nothing syncs

cmdSync (internal/features/ghmirror/ghmirror.go:1020) forwards its args verbatim to BOTH cmdPull(args) and cmdPush(args). But cmdPull at ghmirror.go:937 does f.Reject('dry-run') -- an allowlist permitting only --dry-run -- directly contradicting its own comment 16 lines below (943-945) that says pull is NOT Reject-guarded because sync forwards push's flags and 'pull must ignore them, not refuse.' cmdPush (ghmirror.go:215) legitimately allowlists --since/--findings-as-issues/--with-tasks. Repro: 'dacli github sync core --since 2h' -> cmdPull parses --since into f.vals['since'], f.Reject('dry-run') sees 'since' not in allowlist and returns Usagef('unknown flag(s): --since') (exit 2), cmdSync returns immediately -- neither pull NOR push runs. Only bare 'sync <proj>' and 'sync <proj> --dry-run' work. Breaks the exact windowed-bidirectional-sync case the pull comment was written to support. Fix: make cmdPull tolerate the push flag set (e.g. f.Reject('dry-run','since','findings-as-issues','with-tasks')), or have cmdSync filter args before forwarding to pull. Found by subagent audit; code path verified by direct read of the Reject call and the cmdSync forwarding.
