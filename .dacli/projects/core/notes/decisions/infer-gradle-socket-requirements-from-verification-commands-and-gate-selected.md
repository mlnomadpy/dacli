---
id: d-infer-gradle-socket-requirements-from-verification-commands-and-gate-selected
kind: note
note_kind: decision
created: 2026-08-27T22:47:15Z
created_by: a-maintainer-ptwdk2
about: "[[t-01M1068MEG379NZ2SE5EH6DYZC]]"
---
# Infer Gradle socket requirements from verification commands and gate selected runtimes before profile execution
## Chose
Infer Gradle socket requirements from verification commands and gate selected runtimes before profile execution
## Rejected
Treat successful runtime startup as build-tool verification
## Because
Issue #799 demonstrates these are independent capabilities; provider-neutral execution capability vocabulary lets adapter contracts vary without scheduler vendor branches.
