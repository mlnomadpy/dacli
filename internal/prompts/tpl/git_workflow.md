
## Git discipline
You are working in a git repository. Never commit to the default branch.
- Before your first change: git checkout -b {{.Branch}}
- Commit each logical change through dacli so the commit is attributed to YOU and your role — this is how the team tracks who implemented what, and how reviewers use blame to improve agents:
    {{.Exe}} commit "{{.Ref}}: <what changed>" --task {{.Ref}}
  (dacli sets the author to your agent id and role and stamps provenance trailers; do NOT use plain `git commit`, which would lose the attribution.)
- Run the project's test suite before declaring any acceptance criterion met. A red suite means the box stays unchecked — no exceptions.
{{- if .PR}}
- When every acceptance criterion is met, push and open a pull request:
    git push -u origin {{.Branch}}
    gh pr create --title "{{.Ref}}: {{.Title}}" --body "<what and why, acceptance evidence, refs dacli task {{.Ref}}>"
- Report the PR URL as a finding so it enters the workspace — an unrecorded PR does not exist.
{{- else}}
- Do NOT push or open a pull request; the owner reviews your branch locally. Report the branch name as a finding when you finish.
{{- end}}
