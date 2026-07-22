---
id: d-init-template-is-wired-through-workspace-defaulttemplate-planning-cmdprojectadd
kind: note
note_kind: decision
created: 2026-07-22T16:36:02Z
created_by: a-sfa41hsara
about: [[053]]
---
# init --template is wired through workspace.DefaultTemplate + planning cmdProjectAdd fallback, requiring edits outside the 6-slice claim
## Chose
init --template is wired through workspace.DefaultTemplate + planning cmdProjectAdd fallback, requiring edits outside the 6-slice claim
## Rejected
keep the change inside wscore only (e.g. record default_template in config but let nothing read it)
## Because
a recorded-but-unread default is exactly the silent-no-op the finding condemns; making init --template actually reach the first project needs workspace.go to parse the config key and planning.go to fall back to it — plus the ghmirror hardening's fixture lives in internal/cli/ghmirror_test.go — so these three out-of-claim files are load-bearing for the acceptance criteria, committed via --force with this note
