---
id: f-072-complete-on-branch-dacli-072-calibration-band-join-fixed-build-tests-green
kind: note
note_kind: finding
created: 2026-07-22T19:13:05Z
created_by: a-50a4mhky3r
about: [[072]]
severity: moderate
---
# 072 complete on branch dacli/072-...: calibration band join fixed, build+tests green
Commit 2105d56 by a-50a4mhky3r on branch dacli/072-fix-cal-calibration-band-join-invocation-records-role-model-runrecords-no. Staged ONLY 4 files (git add + dacli commit --no-add): internal/store/calibration.go, internal/store/calibration_test.go, internal/features/execution/execution.go, internal/features/execution/verify.go. ACCEPTANCE: (1) supervise invocation.txt now writes role/model in canonical OrDash form (execution.go:~748: role/model = clikit.OrDash(f.Get(role))/OrDash(modelName)); verify invocation.txt now writes role: verifier, model: - (OrDash empty), runtime: rt.Name plus the preserved verify_panel_seat: marker (verify.go:~124) — so supervise-completed bands finally match the {OrDash,OrDash,rt} band the calibrate gate/advise compare against. (2) runRecords (calibration.go:160) no longer clobbers: readInvocation now returns isVerify (verify_panel_seat line); runRecords skips verify seats entirely and only overwrites band when !band.Empty(). This kills BOTH clobber vectors — a legacy empty band AND a newer verify seat's non-empty {verifier,-,rt} band — so the completing spawn/supervise implementer band survives. New calibration_test.go proves verify-does-not-clobber, empty-does-not-clobber, supervise-band-kept, verify-only-task-has-no-band. (3) go build ./... clean; go test ./internal/... all green (store 3.5s incl. 4 new tests; DACLI_AGENT stripped for cli isolation as siblings noted). Decision note [[d-verify-runs-write-canonical-role-model-but-are-excluded-from-the-calibration]] records why verify writes a canonical band yet is excluded from the join. Owner: verify and close via dacli task check/done + dacli merge --task 072.
