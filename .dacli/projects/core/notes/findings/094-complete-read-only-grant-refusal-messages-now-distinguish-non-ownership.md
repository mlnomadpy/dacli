---
id: f-094-complete-read-only-grant-refusal-messages-now-distinguish-non-ownership
kind: note
note_kind: finding
created: 2026-07-23T10:52:24Z
created_by: a-n9n6r0nn4w
about: [[094]]
severity: minor
---
# 094 complete: read-only-grant refusal messages now distinguish non-ownership from actual ro grant
Commit 7209743 on branch dacli/094-fix-misleading-read-only-grant-messages-the-real-reason-is-non-ownership-not, PR #57 (auto-merge queued). Added Identity.MutateRefusal() (internal/agentid/agentid.go) returning 'read-only grant' when Grant != RW, else 'not the owner' — CanMutate's actual two failure modes. Wired into all 4 messages: task claim (planning.go:244), task done (planning.go:315), task block (planning.go:361), and accept's propose() (acceptance.go:94-101, which additionally names the root --force override since accept is the only command with that recourse). No behavior change — only the message text. go build clean, go test ./internal/... all green. Owner: verify and close via task check/done + merge --task 094.
