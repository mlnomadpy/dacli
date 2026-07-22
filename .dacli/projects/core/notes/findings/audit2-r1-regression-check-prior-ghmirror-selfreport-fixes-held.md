---
id: f-audit2-r1-regression-check-prior-ghmirror-selfreport-fixes-held
kind: note
note_kind: finding
created: 2026-07-22T18:23:33Z
created_by: a-drg65wknjt
about: [[t-01KY5GP5QJS16DCPAHQMTFBE5X]]
source_event: 01KY5GXTP25SN8KKTSF4G4CMFF
---
# AUDIT2 R1 regression check: prior ghmirror/selfreport fixes held
Verified by reading current source (build blocked by headless sandbox approval; read-only review). HELD: (1) searchByMarker now reads the strongly-consistent list endpoint 'gh issue list --state all --json number,body' + exact substring match, NOT the eventually-consistent tokenized --search index (ghmirror.go:722-740) — closes f-ghmirror-marker-idempotency and matches the 053 fix. (2) gh subprocesses are wrapped in context.WithTimeout(120s) via gh() (ghmirror.go:50-63) and selfreport ghOutput() (selfreport.go:93-101) — closes the no-timeout class (f-git-gh-subprocesses / f-selfreport-gh-no-timeout) for both files. (3) G4 inbound pull (cmdPull) is read-only against the remote and correctly ungated; push+finding-comments share one disclosureGate (ghmirror.go:162,238). (4) decisionMarker/findingMarker use distinct prefixes from the task marker and are unit-covered against cross-adoption. No regressions found; residual gaps are the 5 findings filed separately (consent scoping, per-loop searchByMarker, redundant SaveTask, loose about-match, report leak).
