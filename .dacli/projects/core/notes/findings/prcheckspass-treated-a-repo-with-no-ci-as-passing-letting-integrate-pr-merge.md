---
id: f-prcheckspass-treated-a-repo-with-no-ci-as-passing-letting-integrate-pr-merge
kind: note
note_kind: finding
created: 2026-08-04T00:10:17Z
created_by: a-fixer-2675tw
about: "[[216]]"
severity: major
---
# prChecksPass treated a repo with no CI as passing, letting integrate --pr merge everything green
internal/features/vcs/lifecycle.go:1191-1203 (prChecksPass): when gh pr checks reports 'no checks reported' (a repo with zero CI configured), the function returned pass=true, identical to a genuinely all-green check run. dacli integrate --pr (default gated mode, lifecycle.go:1135-1148) merged on pass=true, so a repo with no CI at all would auto-merge every PR through the check gate meant to block red/pending work. Fixed by adding a distinct absent return value: absent checks now leave the PR open with a 'no checks configured on this repo' notice, never conflated with 'checks not passing' (pending/failing) or with a real pass. Covered by TestIntegratePRLeavesOpenWhenNoChecksReported in printegrate_test.go, written first and confirmed red before the fix.
