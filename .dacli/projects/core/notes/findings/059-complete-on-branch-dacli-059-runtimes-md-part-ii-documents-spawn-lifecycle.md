---
id: f-059-complete-on-branch-dacli-059-runtimes-md-part-ii-documents-spawn-lifecycle
kind: note
note_kind: finding
created: 2026-07-22T18:14:01Z
created_by: a-5zfa3xx3z5
about: [[059]]
severity: major
---
# 059 complete on branch dacli/059; RUNTIMES.md Part II documents spawn lifecycle, gates, token calibration
Commit ce917eb by a-5zfa3xx3z5 on branch dacli/059-doc2-... Staged ONLY docs/RUNTIMES.md (git add + dacli commit --no-add), docs-only, 217 insertions. All 3 acceptance criteria met, verified against internal/features/execution + vcs + store:

(1) SPAWN FLAGS + LIFECYCLE — new §19 tables every spawn flag actually present (--advise, --claim, --detach, --worktree, --max-tokens, --force, --review [--pr-number N], --pr, --budget[recorded-not-enforced], --timeout, --cooperative, --grant, --role, --model, --runtime) and the six spawn-time gates in code order (role WIP/seniority/phase, runtime+PATH, --max-tokens cost gate, taint gate, sandbox ro-refusal, claim conflict). §20 documents the lifecycle commands (wait, agents/--tail, agents --max-rss/--max-runtime --reap, logs -f/--tail, kill/--all/--grace, runs list|show|prune, supervise). §21 documents the integration tail (commit claim-scope, accept/--all/--verify, integrate --tasks/--into/--project, ship pipeline+flags, merge, pr --base/--with-verdicts). Verified against execution.go usage string (execution.go:214) and vcs/lifecycle.go + acceptance.go + ship.go Briefs.

(2) TOKEN-ACTUALS PATH — new §23: usage_format: stream-json opt-in (runtime add --usage-format; execRuntime appends --output-format stream-json --verbose), usage.txt capture (teeStreamJSON->writeUsage: output_tokens/input_tokens/num_turns/cost_usd into run dir), detached harvest via finalizeRun in dacli wait, the Band{Role,Model,Runtime} join from invocation.txt, CalibSample.TokenRatio/MedianTokenRatio, the n>=10 AUTHORITATIVE-vs-provisional gate in calibrate/estimate, and how --advise (printAdvisory) DISPLAYS and --max-tokens (bandTokenBudget) ENFORCES the same MedianTokenRatio x Te.

(3) DOCS-ONLY, committed by an agent; go build ./... clean (no code touched). Status banner updated: §§1-18 marked original design, Part II §§19-23 the implemented reference.

NOTE: hit the worktree-path shadow trap — Edit first landed docs/RUNTIMES.md in the MAIN checkout by absolute path; restored main via git checkout and re-applied to the worktree copy, so the branch carries the change and main is untouched. Also filed a separate minor finding: the 'dacli pr' Brief cites --pr-number but that flag lives only on spawn --review (lifecycle.go never reads it). Box-checking refused for non-owner (only a-root) — owner: verify and close via dacli task check/done 059 + dacli merge --task 059.
