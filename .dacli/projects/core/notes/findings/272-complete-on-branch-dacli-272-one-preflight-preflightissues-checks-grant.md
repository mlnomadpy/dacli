---
id: f-272-complete-on-branch-dacli-272-one-preflight-preflightissues-checks-grant
kind: note
note_kind: finding
created: 2026-08-05T13:56:44Z
created_by: a-fixer-p5ee58
about: "[[272]]"
severity: major
---
# 272 complete on branch dacli/272-one-preflight: preflightIssues checks grant-write/binary-allowlist/prompt-tools in one pass, spawn and a new preflight command both use it
Commit 3ddcc83 (a-fixer-p5ee58). All 3 acceptance criteria met.

(1) A single command checks all three classes: internal/features/execution/preflight.go's preflightIssues(rt, role, hasRole, grant, cooperative, exe) runs class 1 (grant vs store.RuntimeWritable, dacli 250's rw check), class 2 (dacli binary path vs allowlist, reusing exeAllowlistWarning/store.RuntimeAllowsDacli from dacli 267), and class 3 (the role prompt's named tools vs what the runtime permits — new store.NamedTools(prompt) extracts backtick-quoted tool identifiers from the role's markdown body, matched against a fixed Claude-Code tool vocabulary; new store.RuntimeAllowsTool(args, tool) checks the runtime's --allowedTools allowlist, mirroring RuntimeWritable's 'no allowlist pinned = nothing to contradict' convention). The new 'dacli preflight --role <name> [--runtime name] [--grant ro|rw] [--cooperative]' command wraps this standalone (execution.go Commands table), resolving role/runtime/grant the same way spawn's resolveLaunch does, minus the task.

(2) Reports every mismatch in one pass: preflightIssues never returns early — it always evaluates all 3 classes and returns every issue found. cmdPreflight prints every issue (warn/refuse) before deciding whether to return a refusal. resolveLaunch (execution.go) now calls preflightIssues BEFORE sandboxFor, printing every non-refusing warning first — so a grant-write refusal (which sandboxFor still owns, unchanged message) no longer hides a binary-allowlist or prompt-tools warning that would previously never have been reached, since sandboxFor used to return early on refusal before warnExeAllowlist (now deleted, subsumed into preflightIssues) ever ran.

(3) spawn runs it, refuse/warn per existing convention: resolveLaunch's new preflightIssues call feeds warn-class issues (binary-allowlist, prompt-tools) to stderr as warnings (exit 0, matching dacli 267's existing warn-not-refuse convention); the refuse-class issue (grant-write) is still actually enforced by sandboxFor's unchanged rw-branch check (exit 3, dacli 250's convention) — preflightIssues surfaces it for reporting but the real gate is unmoved, so no existing refusal message or exit code changed.

PROOF: go build ./... clean, go vet ./... clean, gofmt -l internal/ clean. go test -exec 'env -u DACLI_AGENT' ./... — all packages green, including the new TestPreflightIssuesReportsEveryMismatchInOnePass (verifies all 3 classes fire together on one mismatched role/runtime), TestPreflightIssuesCooperativeDowngradesGrantWriteToWarn, TestPreflightIssuesNoMismatches, TestCmdPreflightReportsAllAndRefuses, TestCmdPreflightNoMismatches, TestCmdPreflightUnknownRoleIsNotFound, and TestSpawnReportsEveryPreflightMismatchBeforeRefusing — the regression test proving the old early-return bug: a role with both a grant-write mismatch AND a prompt-tools mismatch now surfaces the prompt-tools warning in spawn's stderr even though the launch still refuses for the (unchanged) grant-write reason. Also 6 new store-level tests (TestNamedTools, TestRuntimeAllowsTool) verified against a precision discipline matching dacliBashPrefixes: prose verbs ('Write the failing test') are not tool names, only a backtick-quoted exact match is.

Note (out of scope, filed separately as a finding): docs/ROSTER.md:5 still claims the rw grant-vs-write-capability direction is 'not checked' — that was true before dacli 250 landed sandboxFor's rw refusal, and is now stale; not fixed here since it predates this task and touches an unrelated file.

Owner: dacli accept 272 (task check is gated to root now, per the propose:done fix — I could not check the boxes myself).
