---
id: d-traced-ship-s-accept-integrate-seam-and-filed-the-durable-stale-landing
kind: note
note_kind: decision
created: 2026-08-10T16:40:50Z
created_by: a-seam-auditor-qzz7rf
about: "[[325]]"
---
# Traced ship's accept->integrate seam and filed the durable stale-landing-evidence record divergence
## Chose
Traced ship's accept->integrate seam and filed the durable stale-landing-evidence record divergence
## Rejected
Re-filing the already-known ship --push record-strand seam (dacli 323, PR #417) or the loop-close deadlock (task 312), or tracing sync/spawn where siblings already have open findings
## Because
ship's accept-before-integrate ordering is a fresh, unreported composition: two individually-correct steps (accept records landing at close time; integrate lands after) whose fixed order — forced because integrate refuses non-done tasks — writes a permanent, COMMITTED 'NOT in trunk — closed anyway' line into the trajectory (the product) on tasks ship itself lands seconds later. It is the #382 false-record class inverted and is ship-specific (the loop gates accept behind prLandStatus==merged, orchestration.go:904-907), so it is a concrete wrong-report sequence, not a layering opinion — exactly the seam class this role exists to find.
