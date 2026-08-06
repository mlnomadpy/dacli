---
id: f-a-windowed-github-push-published-15-decision-issues-nobody-asked-for-on-a
kind: note
note_kind: finding
created: 2026-08-04T20:13:08Z
created_by: a-root
severity: major
origin: internal/features/ghmirror/ghmirror.go
---
# A windowed github push published 15 decision issues nobody asked for, on a public repo
Ran `dacli github push core 268 269 270 271 272 273 274` — an explicit seven-task window — to test 275's marker-less adoption.

The adoption half worked exactly as designed: issues 336-342, which I had filed by hand with non-canonical titles and no dacli marker, were retitled to the canonical `NNN: <title>` form and ADOPTED. No duplicate task issue was created. That is the thing 275 was built for and it did it.

The window, though, only scopes the TASK mirror. The decision mirror (mirrorDecisions, the G2 feature) runs afterward over the whole project regardless, so the run began creating an issue per decision note in a workspace that holds well over a hundred. I killed it at 15 (issues 348-362).

Two things make this worse than untidy. The repo is PUBLIC, so an unintended publish cannot be taken back — only closed, and the content stays in the timeline. And the command gave no indication of blast radius before starting: a window was accepted, so the reasonable reading is that the window governs the run.

The 15 decisions themselves are legitimate mirrored records and I am leaving them rather than churning the repo further; the defect is that a scoped command did an unscoped thing. Filed as 298. Related: 294 (no --dry-run on any GitHub remote-mutating command) would have caught this before it wrote anything, which is exactly the argument for it.
