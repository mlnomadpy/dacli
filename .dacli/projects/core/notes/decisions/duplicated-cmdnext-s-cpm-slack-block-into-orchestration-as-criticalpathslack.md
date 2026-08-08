---
id: d-duplicated-cmdnext-s-cpm-slack-block-into-orchestration-as-criticalpathslack
kind: note
note_kind: decision
created: 2026-07-23T13:38:36Z
created_by: a-hkm1s8wvp9
about: [[103]]
---
# duplicated cmdNext's CPM-slack block into orchestration as criticalPathSlack(), not shared via a new spm/store helper
## Chose
duplicated cmdNext's CPM-slack block into orchestration as criticalPathSlack(), not shared via a new spm/store helper
## Rejected
extracting a shared CPM-ranking helper into internal/spm or internal/store that both insight and orchestration would call
## Because
the task's STRICT scope is orchestration.go + its test; TestFeatureSlicesAreIsolated already forbids feature-to-feature imports so orchestration cannot import insight directly, and adding a new cross-cutting helper package touches spm/store which are shared entity layers outside this task's file list — duplicating the ~25-line CPM block (already noted with a comment pointing at its origin, same pattern as the existing taskBranch() duplication in this file) is the minimal, in-scope fix; both copies degrade identically (haveCPM=false) when any open task lacks an estimate
