---
id: f-task-490-review-defects-fixed-and-mutation-proven
kind: note
note_kind: finding
created: 2026-08-26T15:12:23Z
created_by: a-root
about: "[[490]]"
severity: major
---
# Task 490 review defects fixed and mutation-proven
PR #795 review fixes: flagless human/JSON project show stays available to a read-only agent while either landing flag is refused at the shared dispatcher; changing that read-shape predicate to false makes TestProjectShowReadOnlyInspectionDoesNotRequireWriteGrant fail with exit 3. Landing-base validation now rejects Git-invalid hidden path components and control/ref hazards; weakening the component check makes TestValidateLandingPolicyRejectsGitInvalidBranchNames fail for foo/.hidden. Focused tests pass after restoration.
