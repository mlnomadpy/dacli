---
id: f-commit-attribution-refusal-behavior-needs-owner-inspection
kind: note
note_kind: finding
created: 2026-08-13T11:46:55Z
created_by: a-fixer-ge5keg
about: "[[411]]"
severity: moderate
---
# Commit attribution/refusal behavior needs owner inspection
After dacli commit returned an outside-claim refusal for internal/store/runtimefiles.go, HEAD was commit ab3b764 containing the claimed execution changes but authored/stamped a-root with a different message. dacli report could not file upstream because gh authentication is invalid. Inspect run 01KZXETK5RXW8YD3JVTVN93NR6 to distinguish concurrent owner activity from non-atomic refused commit behavior.
