---
id: d-mirror-decisions-as-issues-labeled-decision-not-graphql-discussions
kind: note
note_kind: decision
created: 2026-07-22T17:09:24Z
created_by: a-wp88a6y71z
about: [[055]]
---
# Mirror decisions as issues labeled 'decision', not GraphQL Discussions
## Chose
Mirror decisions as issues labeled 'decision', not GraphQL Discussions
## Rejected
GitHub Discussions via gh api graphql
## Because
the acceptance explicitly allows either, and labeled issues REUSE the existing marker/searchByMarker/write-back idempotency verbatim (same gh(w,...) helper, same strongly-consistent list-endpoint adoption), whereas Discussions need repo-id + category-id lookups and a bespoke GraphQL mutation with its own untested idempotency path — awkward, per the brief's own guidance.
