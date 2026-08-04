---
id: f-ghmirror-push-pull-ignore-github-repo-write-to-cwd-remote
kind: note
note_kind: finding
created: 2026-08-04T11:47:08Z
created_by: a-maintainer-nyj8xr
about: "[[221]]"
severity: major
---
# ghmirror push/pull ignore github_repo, write to cwd remote
internal/features/ghmirror/ghmirror.go: gh helper (ghExec, line 57) runs every gh call with cmd.Dir=w.Root and NO --repo, so issue create/edit/list/comment/close and label create resolve the target from whatever the workspace-root git remote points at, not the project's stored github_repo (frontmatter set by github link). A dacli workspace can manage several projects each linked to a DIFFERENT repo, but the root has ONE remote, so all but one project's issues land in the WRONG repository. disclosureGate/repoView (line 77/363) likewise probe cwd visibility, not the linked repo's, so the public-repo gate judges the wrong repo. Contrast: catalog.go:345 repoView already passes --repo (dacli 167) and selfreport.go:116 dacli report already passes --repo. project.go already scopes via --owner/--url.
