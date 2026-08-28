---
id: f-start-dry-run-accepts-a-resolved-reviewer-outside-the-requested-harness
kind: note
note_kind: finding
created: 2026-08-27T23:58:08Z
created_by: a-adversarial-reviewer-rvb458
about: "[[t-01KZXRP56538HNHDP3P4FJHGGP]]"
severity: major
---
# start dry-run accepts a resolved reviewer outside the requested harness
internal/features/orchestration/profile.go:403 resolves project-stack roles before :406 applies the requested harness; :603-606 then forwards the resolved reviewer through --review-role. Trigger: a recorded stack has its cheapest matching reviewer on harness A, the operator requests single harness B, and a compatible B reviewer exists. buildProfilePlan at profile.go:452 merely filters planning roles and dry-run returns success, but live executeProfile enters resolveLoopHarnessPolicy at orchestration.go:2765-2783, treats the forwarded reviewer as explicit, and refuses it as outside harness B instead of selecting the compatible reviewer. Wrong outcome: start --dry-run promises an executable plan that the identical live start refuses before work.
