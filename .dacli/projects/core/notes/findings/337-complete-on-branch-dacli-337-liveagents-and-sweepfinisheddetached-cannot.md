---
id: f-337-complete-on-branch-dacli-337-liveagents-and-sweepfinisheddetached-cannot
kind: note
note_kind: finding
created: 2026-08-10T19:23:43Z
created_by: a-fixer-7s538v
about: "[[337]]"
severity: major
---
# 337 complete on branch dacli/337-liveagents-and-sweepfinisheddetached-cannot-distinguish-an-unreadable-runs
Commit 0b464b8 (a-fixer-7s538v). All 3 acceptance criteria met.

Root cause: liveAgents and sweepFinishedDetached (internal/features/execution/execution.go)
both returned a bare `[]T` and treated any os.ReadDir(w.RunsDir()) error the same
as "no runs yet" — collapsing two different facts ("nobody has run" vs "I could not
read the runs directory") into one empty result. cmdRunsList (same file) had already
been fixed to tell these apart (comment there explicitly says "unlike its two
siblings that cannot (see dacli 337)") — this change makes liveAgents and
sweepFinishedDetached match that precedent.

Fix, in internal/features/execution/execution.go:
1. liveAgents now returns ([]procmon.Record, error): errors.Is(err, fs.ErrNotExist)
   -> (nil, nil) ("no runs yet", unchanged behavior); any other ReadDir error ->
   (nil, fmt.Errorf(...)) naming the path.
2. sweepFinishedDetached mirrors the same signature and split.
3. Every caller updated to surface the error instead of reading it as empty:
   cmdAgents (both the sweep and the live-list), cmdKill --all, cmdKill by-ref,
   cmdWait's "wait on everything live" branch.
4. gateClaimOverlap (one of the launchGates, "claim-overlap") is the actual
   WIP-adjacent consumer: it calls liveAgents to check whether a new spawn's
   --claim collides with a currently-live agent's. Before this fix, an unreadable
   runs dir made liveAgents return empty, so the gate would silently PASS a spawn
   it could not actually verify was claim-disjoint — the literal "unreadable
   directory reads as nobody is working" failure mode named in the task's
   acceptance criteria. It now fails closed (refuses the spawn) on a read error,
   matching the rule internal/gates already holds its own quantifier gates to
   ("a gate must never certify what it could not read").

Note: store.ActiveInRole / store.liveChildren (internal/store/roles.go) is the
role-WIP-LIMIT counter (`role X is at its WIP limit`) and has the exact same
swallowed-error shape, but it is a structurally separate implementation in a
different package (store cannot import execution) — not reachable from either
named function, so left out of this diff's scope. Recommend a follow-up task if
that counter's silent-undercount-on-unreadable-dir is also wanted fixed.

Red-green verified by hand: reverted just the liveAgents error-classification
(kept the (T, error) signature, made it swallow the error again) and reran the
new tests — both failed for the predicted reason (liveAgents on an unreadable
runs dir returned (nil, nil) instead of an error; gateClaimOverlap passed on an
unreadable runs dir). Restored the fix, both green again.

New tests in internal/features/execution/runrecord_test.go:
- TestLiveAgentsAndSweepFinishedDetachedFailOnUnreadableRunsDir: asserts both
  functions return (empty, nil) on a nonexistent runs dir and (_, non-nil error)
  once the runs dir is replaced by a regular file (ENOTDIR, the same
  unreadable-directory technique internal/gates' unreadableTasksProject uses).
- TestGateClaimOverlapFailsClosedOnUnreadableRunsDir: same fixture, asserts
  gateClaimOverlap refuses (non-zero exit code) rather than passing.

PROOF: go build ./... clean, go vet ./... clean, gofmt -l . clean.
go test ./internal/features/execution/... -count=1 -v: all tests green (added
2, updated 1 existing caller in TestLiveAgentsProbesLivenessAndExcludesGhosts
for the new signature). Full go test ./...: only pre-existing failures are in
internal/features/briefing (TestCatchup*) and orchestration/noremote_test.go's
TestIntoRefusesAnUnknownBranchUpFront, all hitting this dogfood session's
ambient DACLI_AGENT — unrelated to this change, already documented by prior
cycles as unrelated.

golangci-lint could NOT be run: the binary is blocked by this sandbox's
permission system requiring interactive approval, which is unavailable to a
headless agent. gofmt/vet/test are all clean; flagging this gap honestly rather
than claiming a check I did not run.

Owner: dacli accept 337 (task check is gated to a-root; I could not check the
boxes myself). PR-first is off — branch
dacli/337-liveagents-and-sweepfinisheddetached-cannot-distinguish-an-unreadable-runs
is ready for accept + integrate/merge --task 337.
