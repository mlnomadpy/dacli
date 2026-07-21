
## Git discipline
You are working in a git repository. Never commit to the default branch.
- Before your first change: git checkout -b {{.Branch}}
- Commit each logical change through dacli so the commit is attributed to YOU and your role — this is how the team tracks who implemented what, and how reviewers use blame to improve agents:
    {{.Exe}} commit "{{.Ref}}: <what changed>" --task {{.Ref}}
  (dacli sets the author to your agent id and role and stamps provenance trailers; do NOT use plain `git commit`, which would lose the attribution.)
- Run the project's test suite before declaring any acceptance criterion met. A red suite means the box stays unchecked — no exceptions.
{{- if .PR}}
- When every acceptance criterion is met, push your branch and open a PR through dacli (records the PR as a finding automatically):
    {{.Exe}} push --task {{.Ref}}
    {{.Exe}} pr --task {{.Ref}}
{{- else}}
- Do NOT push or open a pull request; the owner reviews your branch and merges with `dacli merge --task {{.Ref}}`. Report the branch name as a finding when you finish.
{{- end}}
- You are working in an isolated worktree if you were spawned with one — your branch is yours alone; other agents on sibling tasks cannot touch your files. The owner integrates all done branches with `dacli integrate`, and a merge conflict blocks the task rather than corrupting anyone's tree.
