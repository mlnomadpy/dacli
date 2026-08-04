---
id: f-210-complete-on-branch-dacli-210-proposals-consumed-only-after-durable-close
kind: note
note_kind: finding
created: 2026-08-04T10:18:23Z
created_by: a-maintainer-qy7k6x
about: "[[210]]"
severity: major
---
# 210 complete on branch dacli/210-...: proposals consumed only after durable close
Fix on branch dacli/210-acceptance-proposals-are-consumed-before-the-close-is-durable, commit c3c8f80.

BUG: acceptance.go acceptOne (was line 130) and acceptAll (was line 200) called applyProposals() -> eventlog.MarkApplied() BEFORE store.CloseTask(). If CloseTask fails (e.g. MoveTask's os.Rename/MkdirAll on the done/ status dir), the proposal is already marked applied on disk but the task stays open. proposedTasks() only lists PENDING proposals, so the next 'accept' can never re-find the task -> completed work is permanently invisible (the exact failure the task names).

FIX: split applyProposals into pendingProposals() (read-only) + markProposalsApplied() (consume); both accept paths now read the proposals, write the accept-log/close, and call markProposalsApplied() ONLY after CloseTask() returns nil. On a CloseTask error both paths return before the mark, so proposals stay pending and the task is re-found on retry. Acceptance-log count uses len(proposals) (known before close) so the 'accepted by X (applied N proposal(s))' record is unchanged.

VERIFIED by reproduction: new TestProposalStaysPendingWhenCloseFails (acceptance_test.go) blocks the done/ dir with a regular file so CloseTask's MoveTask fails, then asserts the proposal is still pending. FAILED before the fix ('proposal was consumed before the close became durable'), PASSES after. go build ./... clean; go test ./internal/features/acceptance/ green; go vet ./... clean; gofmt -l internal/ empty. Full go test ./... green with DACLI_AGENT stripped (the one catalog failure is the pre-existing DACLI_AGENT test-isolation leak, unrelated).

Owner: close via 'dacli accept 210' (box 1 + done).
