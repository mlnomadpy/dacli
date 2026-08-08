---
id: d-loop-advise-groups-calibration-by-role-alone-not-the-full-role-model-runtime
kind: note
note_kind: decision
created: 2026-07-23T12:56:44Z
created_by: a-cbbzr945ja
about: [[100]]
---
# loop --advise groups calibration by ROLE alone, not the full role×model×runtime Band
## Chose
loop --advise groups calibration by ROLE alone, not the full role×model×runtime Band
## Rejected
require exact Band match (role+model+runtime) like spawn --advise/--max-tokens do
## Because
loop's --impl-role/--review-role pin only a role; the actual model/runtime a future spawn resolves to isn't known ahead of time, so an exact-Band join would silently return n=0 for every band-mixed role and never surface a figure. New store.TokensPerRun(samples, role) aggregates raw output-token actuals (not the tokens-per-point ratio spawn --advise uses, since a cycle projection has no per-task Te to multiply by) across every sample sharing that role, gated at the same n>=10 threshold as the rest of calibrate.
