---
id: f-context-provenance-regression-test-detects-the-original-codex-global-skill-leak
kind: note
note_kind: finding
created: 2026-08-19T13:36:36Z
created_by: a-maintainer-1cw4s7
about: "[[t-01M0AEYFXAB22RE9Y2SH9WZZKR]]"
severity: major
---
# Context provenance regression test detects the original Codex global-skill leak
Mutation evidence: removing filepath.Join(home, '.agents', 'skills') from internal/store/contextprovenance.go made ok  	github.com/mlnomadpy/dacli/internal/store	0.936s fail at contextprovenance_test.go:39 with 'invalid global fixture not discovered'. Restored code passes focused tests and serial go test ./....
