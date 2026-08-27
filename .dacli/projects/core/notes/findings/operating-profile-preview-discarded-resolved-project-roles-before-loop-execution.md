---
id: f-operating-profile-preview-discarded-resolved-project-roles-before-loop-execution
kind: note
note_kind: finding
created: 2026-08-27T23:16:30Z
created_by: a-maintainer-mzxznz
about: "[[t-01M1068MNDZ72H5R35YRZ9MASK]]"
severity: major
---
# Operating profile preview discarded resolved project roles before loop execution
internal/features/orchestration/profile.go profileLoopArgs forwarded budgets and landing but no implementation/review role, while buildProfilePlan independently selected task roles; an Android dry-run could therefore disagree with the loop defaults. Regression mutation failed at profile_test.go:283 when review forwarding was removed.
