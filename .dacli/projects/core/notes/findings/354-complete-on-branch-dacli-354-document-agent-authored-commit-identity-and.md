---
id: f-354-complete-on-branch-dacli-354-document-agent-authored-commit-identity-and
kind: note
note_kind: finding
created: 2026-08-11T09:47:18Z
created_by: a-fixer-1p9jwx
about: "[[354]]"
severity: minor
---
# 354 complete on branch dacli/354-document-agent-authored-commit-identity-and-how-self-hosted-changes-are
Commit 97d1760 (a-fixer-1p9jwx). All 3 acceptance criteria met, in docs/SELFHOSTING.md:

(1) Trailers explained per-trailer, each with what it lets a reader reconstruct: Dacli-Agent -> the agent's other findings/decisions/commits via contrib/blame and its identity file; Dacli-Role -> per-role defect-rate rollup (dacli contrib), the signal for improving a role's prompt/scope/model; Dacli-Task -> the task's acceptance criteria/findings/decisions, the why behind the diff. Cites internal/features/vcs/vcs.go:150-162 (verified current).

(2) New section 'How agent-written code is reviewed before it lands' states both landing paths: PR-first (CI-gated by default, human-gated only under --no-merge) vs local integrate (dacli integrate/merge/ship with no --pr), and states plainly that local integrate is a plain git merge (mergeTask, internal/features/vcs/lifecycle.go:1090) that reads no diff -- only conflict detection. Grounded in a real example from this repo's own history (PR #429/task 342, commit c81c3fb: CI caught a stale assertion the author's own truncated local test run missed) to show what CI-gated PR-first catches that local integrate structurally cannot. Also notes dacli verify is claim-verification (adversarial panel refuting one claim), not diff review, and the loop's periodic reviewPhase (orchestration.go:1642) is a backlog self-audit that files new tasks, not a gate on the task about to merge.

(3) New section 'Which commits are agent-authored, and how to tell' gives reproducible counts as of 2026-08-11: 616 commits on main, 290 carry a Dacli-Agent trailer (git log main --grep='^Dacli-Agent:' -E --oneline | wc -l), 221 have an @agent.dacli author identity (undercounts because most PRs squash-merge under the human merger's identity while preserving trailers in the squash body). Also corrects the doc's prior overclaim that '96 pull requests... every one authored by a dacli agent' -- verified false: PR #440 (commit a007429) carries zero Dacli-Agent trailers, only Co-Authored-By: Claude Opus 5 (an interactive session, not a spawned agent).

PROOF: docs-only change (docs/SELFHOSTING.md), no Go files touched. gofmt -l . clean, go vet ./... clean. Every git log count cited was re-run and verified against this repo's actual main branch immediately before writing it, not copied from memory.

Also filed a separate minor finding: README.md and docs/index.md both still cite the old, now-inaccurate '96 merged PRs, every one authored by a dacli agent' claim -- left untouched as out of scope for this task's acceptance criteria (which are about SELFHOSTING.md).

Owner: dacli accept 354 (task check is gated to a-root). PR-first is off -- branch dacli/354-document-agent-authored-commit-identity-and-how-self-hosted-changes-are is ready for accept + integrate/merge --task 354.
