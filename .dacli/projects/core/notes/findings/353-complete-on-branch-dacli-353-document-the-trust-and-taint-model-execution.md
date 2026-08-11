---
id: f-353-complete-on-branch-dacli-353-document-the-trust-and-taint-model-execution
kind: note
note_kind: finding
created: 2026-08-11T10:14:24Z
created_by: a-fixer-kf182p
about: "[[353]]"
severity: major
---
# 353 complete on branch dacli/353-document-the-trust-and-taint-model-execution-boundaries-secret-handling-and
Commit f81dea2 (a-fixer-kf182p). All 4 acceptance criteria met.

New file docs/TRUST.md (393 lines), added to the docs index (docs/README.md), plus a new invariant test internal/cli/trust_test.go.

(1) Every command declaring Mutates: true is covered. Enumerated the live command table (55 commands today, printed directly from the internal/cli 'commands' aggregate, not hand-counted from grep) and documented each one's what-it-changes / what-gates-it / how-to-undo-it in docs/TRUST.md § 5, grouped by feature slice: workspace/onboarding, projects/tasks/templates/stage-gates, team/agents/roles/skills, execution (runtimes/spawn/supervise/kill), version control/landing, ship/orchestration/collaboration, GitHub mirror, shortcuts/queues/ad-hoc execution. Detail was gathered by three parallel research passes tracing each handler to its actual write call (file:line), not guessed from command briefs.

(2) The taint model and untrusted-content boundary are stated in one place: docs/TRUST.md § 2. Synthesizes internal/store/taint.go's own doc comment ('does NOT prevent injection... converts attribution into a command'), the self-declared/opt-in nature of the origin field (internal/eventlog/eventlog.go, defaults to "agent" when unset), the taint gate at spawn time (RUNTIMES.md § 19 gate 4), and the brief's quoted-block attribution (ARCHITECTURE.md § 6) — with an explicit 'untrusted-content boundary' paragraph naming what's trusted (authored through dacli) vs untrusted (files/PRs/issues/transcripts from outside dacli's own writers).

(3) Secret handling (§ 4) states what dacli reads (DACLI_AGENT via os.LookupEnv, the operator's own gh/Claude Code logins — never a credential dacli itself holds), what it never writes to a record (token hash only in agents/<id>.md, env var NAMES only — never values — in invocation.txt, credential-shaped env_passthrough denied outright at both runtime-add and spawn-time), and a table of exactly where an agent token can and cannot appear (DACLI_AGENT env var and one-time stdout print: yes; agents/<id>.md, invocation.txt, commit trailers, wikilinks: no — hash or id only).

(4) internal/cli/trust_test.go: TestTrustDocListsEveryMutatingCommand enumerates internal/cli's live 'commands' table, filters to Mutates: true, and asserts each Path string appears in docs/TRUST.md — so a new Mutates command that isn't documented fails the build rather than aging quietly out of date (the same pattern as TestFlagTakingCommandsDocumentTheirFlags, issue #436). Red-green verified by hand: temporarily replaced every occurrence of 'codeowners' in the doc with a placeholder via Edit, reran — failed exactly as predicted ('docs/TRUST.md does not name 1 of 55 Mutates commands: github codeowners'), then reverted and confirmed green again.

Two genuine, verified inconsistencies were found while researching this doc and are filed as separate findings rather than silently smoothed over or fixed (out of scope for a documentation task): [[f-dacli-taint-is-mutates-true-but-performs-no-write-so-a-read-only-agent-is]] (taint declares Mutates: true but its whole call path is read-only, confirmed independently by two research passes) and [[f-escalate-is-mutates-true-at-the-dispatcher-silently-defeating-its-own-open-to]] (escalate's dispatcher-level Mutates: true gate blocks a ro caller from the WHOLE command, contradicting the handler's own comment that the local, --github-less escalation is 'open to any agent — that is the point'). Both are documented as the doc's actual, current, verified behavior in docs/TRUST.md § 5 and § 6, not the aspirational comments.

PROOF: gofmt -l . clean, go vet ./... clean. go test ./internal/cli/... -run TestTrustDocListsEveryMutatingCommand -v: PASS. Full go test ./...: internal/cli is fully green (including every existing dispatcher/flag/taint test alongside the new one); the only failures anywhere are the pre-existing, well-documented ambient-DACLI_AGENT dogfood-session artifact in internal/features/briefing (TestCatchup*) and internal/features/orchestration (TestIntoRefusesAnUnknownBranchUpFront) and one in internal/features/teamops (TestAgentSpawnFailsClosedWhenTheWIPCountCannotBeRead) — all three fail with 'agent token not recognized in this workspace' because this session's own DACLI_AGENT is set and those tests don't unset it; none touch anything this task changed.

golangci-lint could NOT be run: the binary requires interactive approval in this sandbox, unavailable to a headless agent — flagging this gap honestly rather than claiming a check I did not run (same limitation prior fixers in this project have documented).

Owner: dacli accept 353 (task check is gated to a-root; I could not check the boxes myself — confirmed exit 3, 'only the owner (a-root) checks acceptance boxes'). PR-first is off — branch dacli/353-document-the-trust-and-taint-model-execution-boundaries-secret-handling-and is ready for accept + integrate/merge --task 353.
