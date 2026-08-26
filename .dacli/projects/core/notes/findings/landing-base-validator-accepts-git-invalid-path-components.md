---
id: f-landing-base-validator-accepts-git-invalid-path-components
kind: note
note_kind: finding
created: 2026-08-26T14:44:16Z
created_by: a-adversarial-reviewer-a1h3ab
about: "[[t-01M0F8DMCN93FCDE59FSEDTJB3]]"
severity: major
against: a-fixer-1x0gq5
---
# Landing-base validator accepts Git-invalid path components
internal/model/landing.go:47-62 rejects a dot only at the start of the whole base and checks inner components only for exact '.'/'..'. Trigger: project show p --landing-base foo/.hidden. The validator returns nil and persists the value, but 'git check-ref-format --branch foo/.hidden' exits 128 ('not a valid branch name'). Wrong outcome: configuration reports success and later ship/integrate inherit a base Git cannot use, violating rejection of unsafe bases and moving the failure to landing time. Use Git-equivalent ref validation (also cover control characters) and test the public command leaves the file byte-identical.
