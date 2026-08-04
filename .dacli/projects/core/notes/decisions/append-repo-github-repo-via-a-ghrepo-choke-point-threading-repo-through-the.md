---
id: d-append-repo-github-repo-via-a-ghrepo-choke-point-threading-repo-through-the
kind: note
note_kind: decision
created: 2026-08-04T11:47:16Z
created_by: a-maintainer-nyj8xr
about: "[[221]]"
---
# Append --repo <github_repo> via a ghRepo() choke point, threading repo through the issue/label helpers
## Chose
Append --repo <github_repo> via a ghRepo() choke point, threading repo through the issue/label helpers
## Rejected
Prepend --repo before the subcommand, or change gh var signature to gh(w,repo,args...)
## Because
gh's --repo is a per-command (cobra persistent) flag, invalid at the root 'gh' level, so it must come AFTER the subcommand verb — selfreport.go and catalog.go both append it. A ghRepo(w,repo,args...) helper that appends --repo when repo!='' and delegates to the stubbable gh var keeps auth-status (which has no --repo) on the bare gh path, keeps the fail-open/marker tests (which stub gh and ignore args) green, and empty repo falls back to cwd resolution so the pre-link discovery paths (github doctor, github link) still work. Changing the gh var signature would force every test stub and the auth call to carry a repo they don't use.
