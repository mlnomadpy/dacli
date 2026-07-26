---
id: f-153-complete-ro-grant-runs-excluded-from-burn-rate-evidence-for-all-4
kind: note
note_kind: finding
created: 2026-07-26T21:24:37Z
created_by: a-sz4h77f3rf
about: [[153]]
severity: moderate
---
# 153 complete: ro-grant runs excluded from burn Rate, evidence for all 4 acceptance criteria
Commit c1468f5. internal/features/dashboard/burn.go:261-283 reviewerRoles now excludes a role when r.Kind=="reviewer" OR r.Grant==string(model.GrantRO) (not Kind-only), so an ro-grant role that omits role_kind is still dropped. Doc comment on reviewerRoles updated to state the ro-grant exclusion rule and why (an ro role never becomes a calibration sample). New unit test internal/features/dashboard/burn_test.go TestBurnRateExcludesROGrantRoleWithoutKind: creates role 'reviewer' with Grant:"ro" and no Kind set, plus role 'builder' with Grant:"rw"; proves the reviewer's 20-token run is dropped from the latest day's Series/Rate (240 tokens, 1 run) while the builder's 240-token run is counted. .dacli/roles/reviewer.md backfilled with role_kind: reviewer as belt-and-suspenders data hygiene. go build ./... clean; go test ./internal/... all green incl. internal/features/dashboard (TestBurnRateExcludesROGrantRoleWithoutKind and existing TestBurnAlertIgnoresReviewAndVerifyDilution both PASS).
