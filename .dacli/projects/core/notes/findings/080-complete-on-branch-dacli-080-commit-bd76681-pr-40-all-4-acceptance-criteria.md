---
id: f-080-complete-on-branch-dacli-080-commit-bd76681-pr-40-all-4-acceptance-criteria
kind: note
note_kind: finding
created: 2026-07-22T22:01:00Z
created_by: a-gx269dxyzs
about: [[080]]
severity: moderate
---
# 080 complete on branch dacli/080 (commit bd76681, PR #40) — all 4 acceptance criteria met
Upgraded 4 prompt templates in internal/prompts/tpl/. (1) mcp_tools.md: rewrote the ## cli escape-hatch section to cover the CURRENT admin surface — now names github project + catalog (both were MISSING) and the --pr merge path on ship/integrate; verified every verb against feature Path: tables (vcs/lifecycle.go, ship.go, ghmirror.go, execution.go, insight.go, catalog.go) and spawn flags --advise/--claim/--detach/--max-tokens/--force at execution.go:271-322; added a tiered-surface + refusal-as-result intro. (2) review_workflow.md:8 adds the verify trust-floor (refuted<unverified<confirmed) and pr --with-verdicts to the real flow; gh pr diff / blame / --request-changes already present. (3) verify_refute.md sharpened adversarial-refute framing (per-runtime seat, attack-not-audit, evidence-missing=refute, asymmetry rationale) while PRESERVING the exact 'verdict: confirmed —'/'verdict: refuted —' title strings that verify.go:242-246 verdictFor parses. (4) brief_header.md keeps its emphasized SYSTEM data-not-instructions line unchanged. go build ./... clean; go test ./internal/... all green (prompts, mcp, cli included). Box-checking refused for non-owner (exit 3, owner-only) — owner verify + close via dacli accept/task done + merge --task 080.
