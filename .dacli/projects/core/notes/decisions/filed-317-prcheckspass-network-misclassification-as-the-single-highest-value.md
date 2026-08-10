---
id: d-filed-317-prcheckspass-network-misclassification-as-the-single-highest-value
kind: note
note_kind: decision
created: 2026-08-10T13:56:49Z
created_by: a-go-auditor-5yxchj
about: "[[313]]"
---
# Filed 317 (prChecksPass network-misclassification) as the single highest-value audit item; recorded gates-stage-advance and CPM-anchor as findings, not tasks
## Chose
Filed 317 (prChecksPass network-misclassification) as the single highest-value audit item; recorded gates-stage-advance and CPM-anchor as findings, not tasks
## Rejected
Filing the stage-gate read-error pass, the CPM-anchor ordering divergence, or the CheckAllAcceptance section-rewrite data loss as the top task instead
## Because
Ranked by what an UNATTENDED run does wrong: 317 is the only defect that lands UNVERIFIED CODE ON TRUNK and records it as merged (top 'record disagreeing with reality' class), reachable every cycle on the loop's own ship/integrate --pr path, triggered by an ordinary CI check name containing 'timeout'/'eof'/'unreachable'. The gates read-error pass (major) needs a non-ENOENT I/O error to fire and only mis-advances a stage; the CPM-anchor divergence (moderate) fires every cycle but only degrades build ORDER, no wrong code; the acceptance section-rewrite is data loss but not a false pass. All three are verified at HEAD and recorded as findings so a later cycle can pick them up; each names its cheapest fix (317: stop deriving netErr from the gh pr checks table; gates: propagate the ListTasks/ListRisks error instead of ',_'; CPM: add '&& !t.IsLoopAnchor()' at orchestration.go:1826).
