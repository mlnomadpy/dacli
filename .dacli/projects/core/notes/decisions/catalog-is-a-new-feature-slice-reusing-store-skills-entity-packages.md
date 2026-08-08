---
id: d-catalog-is-a-new-feature-slice-reusing-store-skills-entity-packages
kind: note
note_kind: decision
created: 2026-07-22T20:53:22Z
created_by: a-0bsqxp2kpx
about: [[069]]
---
# catalog is a new feature slice reusing store+skills entity packages, reimplementing the disclosure gate locally
## Chose
catalog is a new feature slice reusing store+skills entity packages, reimplementing the disclosure gate locally
## Rejected
put catalog in teamops, or import ghmirror's disclosureGate
## Because
arch_test forbids feature→feature imports; catalog reads via store.LoadRoles/skills.LoadSkills/store.FileChangelog (entity layer, allowed) and reimplements the scoped-consent gate (consentCoversRepo) so wiki publish honors the same PUBLIC-repo disclosure rule as github push without coupling slices
