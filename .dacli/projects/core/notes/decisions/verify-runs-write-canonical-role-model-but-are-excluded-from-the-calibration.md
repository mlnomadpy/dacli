---
id: d-verify-runs-write-canonical-role-model-but-are-excluded-from-the-calibration
kind: note
note_kind: decision
created: 2026-07-22T19:12:31Z
created_by: a-50a4mhky3r
about: [[072]]
---
# verify runs write canonical role/model but are EXCLUDED from the calibration band join
## Chose
verify runs write canonical role/model but are EXCLUDED from the calibration band join
## Rejected
empty-band guard alone
## Because
criterion 1 makes verify write role/model so its band is non-empty; an empty-only guard would let the newer verify run clobber the completing spawn's implementer band. readInvocation flags verify_panel_seat and runRecords skips verify seats for band and usage.
