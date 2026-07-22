---
id: f-029-d3-complete-on-branch-dacli-029-commit-4515cca-all-4-acceptance-met
kind: note
note_kind: finding
created: 2026-07-22T13:52:35Z
created_by: a-9v3qa4cstk
about: [[029]]
severity: moderate
---
# 029 D3 complete on branch dacli/029 — commit 4515cca, all 4 acceptance met
Committed 4515cca by a-9v3qa4cstk on branch dacli/029-...taint-gate via git add + dacli commit --no-add, staging ONLY the 4 scoped files: internal/brief/brief.go, internal/features/execution/{execution.go,verify.go}, internal/store/store.go. ACCEPTANCE: (1) TRUST-FLOOR: brief.go §8 labels each surfaced finding [trust: confirmed|refuted|unverified] from its note's trust: front key and adds a '**trust-floor: <worst>**' line = worst grade among surfaced findings, ordered refuted<unverified<confirmed (trustRank/rankTrust helpers); a pending finding event is ungraded->unverified. (2) VERIFY GRADES BEFORE briefs: store.GradeFinding(w,project,ref,trust) stamps trust: onto the matching finding NOTE (by id or level-1 title==claim) and re-saves via mdstore.WriteFile; verify.go cmdVerify calls it after the tally with trust=confirmed if confirmed>=require else refuted, best-effort (a pending-event claim with no note reports a miss, non-fatal). (3) TAINT GATE: cmdSpawn refuses (clikit.Refusedf, exit 3) when externalRadius(w,t) reports the task in store.Taint('external:')/ExposedBriefs blast radius, unless --force or --cooperative; the shared externalRadius helper also backs --advise so display and gate never diverge; usage adds [--force]. (4) go build ./... clean; go test ./internal/... all green (incl. internal/cli, TestFeatureSlicesAreIsolated). Verified end-to-end via a throwaway internal/cli test (deleted, not staged): unverified->confirmed->refuted floor transitions and the exit-3 refusal + --force override. NOTE: rebuilt-binary smoke blocked by headless sandbox (arbitrary-path exec needs approval), so verified via build+unit paths. Owner: verify and close via dacli task check/done + dacli merge --task 029.
