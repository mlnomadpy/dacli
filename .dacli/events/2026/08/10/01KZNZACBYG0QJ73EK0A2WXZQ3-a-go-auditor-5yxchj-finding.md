---
id: 01KZNZACBYG0QJ73EK0A2WXZQ3
kind: event
event_kind: finding
created: 2026-08-10T13:55:42Z
created_by: a-go-auditor-5yxchj
about: "[[t-01KZNYHPH9DVJJYD6P0WAVFNB5]]"
origin: agent
applied: true
---
stage gates report PASS on a swallowed ListTasks/ListRisks read error (empty set => vacuous true), advancing a stage with zero tasks examined

internal/gates/gates.go evaluate(): the universal-quantifier predicates discard the store read error and then pass on an empty slice.
- gates.go:418  tasks,_ := store.ListTasks(w,p.Slug,'')  feeds all_have_acceptance (420-427), all_have_estimate (428-435), musts_done (436-443).
- gates.go:448  risks,_ := store.ListRisks(w,p.Slug)  feeds rank1_have_action (447-455).
Each is a 'for every X, P(X)' check, so an empty slice yields OK:true. store.ListRisks (internal/store/risk.go:72) and store.ListNotes/ListTasks deliberately return an error on a real failure with the comment 'a real I/O/permission failure must not read as empty' — evaluate() drops exactly that signal.

FAILURE SCENARIO: a project at the build/ship stage of the shipped 'standard' template (internal/gates/tpl/standard.md:20-28). If .dacli/projects/<slug>/tasks/ hits any non-ENOENT read error (permission denied, transient I/O, a bad symlink), ListTasks returns (nil,err); the error is dropped; tasks==nil; no task is flagged; every task gate returns OK:true. dacli stage advance <project> (internal/features/stagegate/stagegate.go:154) sees len(unmet)==0 and advances, printing 'advanced to stage…'/'template complete — every gate passed' while ZERO tasks were examined. Used by every shipped template (standard, product, research-paper, tdd). CHEAPEST FIX: propagate the read error as an evaluation failure (gate OK:false with the error as detail) instead of ',_'. decisions/glossary/retro use the same pattern but fail closed (n>=1), so only these four quantifier predicates are affected.
