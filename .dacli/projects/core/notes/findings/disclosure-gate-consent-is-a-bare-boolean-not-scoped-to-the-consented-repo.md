---
id: f-disclosure-gate-consent-is-a-bare-boolean-not-scoped-to-the-consented-repo
kind: note
note_kind: finding
created: 2026-07-22T18:23:33Z
created_by: a-drg65wknjt
about: [[t-01KY5GP5QJS16DCPAHQMTFBE5X]]
source_event: 01KY5GWHXB0B20Z59Z48RNY8KY
github:
  issue: 28
  repo: mlnomadpy/dacli
---
# Disclosure-gate consent is a bare boolean, not scoped to the consented repo
ghmirror.go:129 records consent as a bare 'github_public_confirmed=true' and disclosureGate (ghmirror.go:238-248) checks only live visibility + that flag — it never verifies the flag was granted for the CURRENTLY-linked repo. Two exploit paths for the risk-rank-2 leak: (1) Re-link: cmdLink (ghmirror.go:120-130) sets github_public_confirmed=true when linking a PUBLIC repo but never CLEARS it when re-linking to a private repo; if that private repo later flips public, disclosureGate passes on stale consent given for a different repo. (2) Remote drift: repoView reads the cwd git remote live, so if the remote is repointed to a different public repo B, consent recorded for public repo A authorizes a findings/decisions push to B. Fix: store the repo name the consent was granted for (or clear the flag on re-link) and require it to equal the live repo in disclosureGate.
