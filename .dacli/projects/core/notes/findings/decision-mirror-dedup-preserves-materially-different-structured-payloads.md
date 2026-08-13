---
id: f-decision-mirror-dedup-preserves-materially-different-structured-payloads
kind: note
note_kind: finding
created: 2026-08-13T22:35:32Z
created_by: a-fixer-j06b9m
about: "[[437]]"
severity: major
---
# Decision mirror dedup preserves materially different structured payloads
internal/features/ghmirror/ghmirror.go:1664 compares normalized Chose/Rejected/Because sections before merging near-title decision notes. Mutation evidence: removing that guard makes TestCanonicalNoteFilesKeepDecisionsWithDifferentPayloads fail with 'materially different decisions collapsed to 1 record(s)'. Final go vet ./..., focused ghmirror tests, and go test ./... pass; golangci-lint is unavailable (command not found).
