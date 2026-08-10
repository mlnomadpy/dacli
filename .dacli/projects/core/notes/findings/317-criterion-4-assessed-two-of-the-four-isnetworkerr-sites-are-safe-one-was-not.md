---
id: f-317-criterion-4-assessed-two-of-the-four-isnetworkerr-sites-are-safe-one-was-not
kind: note
note_kind: finding
created: 2026-08-10T14:29:18Z
created_by: a-root
about: "[[317]]"
severity: moderate
origin: internal/features/vcs/lifecycle.go:1280,1315,1372
---
# 317 criterion 4 assessed: two of the four isNetworkErr sites are safe, one was not
Checked all four isNetworkErr call sites for the same misclassification, rather than assuming the pattern was uniform. lifecycle.go:1372 (gh pr merge --squash/--merge) IS the same hazard with the same consequence: a REFUSED merge (red checks, conflict, branch protection) whose text carries a network token fell through to mergeTask — a local merge to trunk, reported as landed. Fixed with the same reachedGitHub discriminator, widened to recognise a mergeability verdict as a server answer, and covered red-then-green. NOT changed, with reasons: lifecycle.go:1315 (gh pr merge --auto) misclassifies into a DIFFERENT ERROR MESSAGE, never a merge — both branches return (false, error), so the blast radius is a misleading sentence rather than unverified code on trunk. lifecycle.go:1280 (pushBranch) scans GIT's push output, not a gh server reply; a non-fast-forward rejection does not carry network tokens, and reachedGitHub's vocabulary (checks tables, PR verdicts) does not apply to git transport output — adding it there would be cargo-culting a guard into a domain it cannot judge. Recorded so a future reader does not read the two untouched sites as an oversight.
