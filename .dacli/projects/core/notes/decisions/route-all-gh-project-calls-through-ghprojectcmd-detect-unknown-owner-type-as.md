---
id: d-route-all-gh-project-calls-through-ghprojectcmd-detect-unknown-owner-type-as
kind: note
note_kind: decision
created: 2026-07-22T22:12:55Z
created_by: a-aztk8559eb
about: [[082]]
---
# Route all gh project calls through ghProjectCmd, detect 'unknown owner type' as the missing-scope signal
## Chose
Route all gh project calls through ghProjectCmd, detect 'unknown owner type' as the missing-scope signal
## Rejected
Match gh's exact scope-error wording, or only wrap the first project call
## Because
gh emits the opaque 'unknown owner type' (not a scope-named error) when the token lacks the 'project' scope; matching that substring case-insensitively at a single wrapper (ghProjectCmd in internal/features/ghmirror/project.go) means the actionable 'gh auth refresh -s project' hint fires no matter which project subcommand hits the missing scope first, without depending on gh's fragile error phrasing
