---
id: d-move-native-macos-verification-to-the-release-workflow
kind: note
note_kind: decision
created: 2026-08-28T10:03:31Z
created_by: a-fixer-fv8pny
about: "[[t-01M13X19WKEC3MXWMS475GCSR2]]"
---
# Move native macOS verification to the release workflow
## Chose
Move native macOS verification to the release workflow
## Rejected
Keep the full Go/frontend suite on macOS for every pull request
## Because
Linux retains every routine full-suite and cross-compile check, while a narrow release-time Darwin runtime test catches host-specific process behavior without paying the macOS runner multiplier on each PR.
