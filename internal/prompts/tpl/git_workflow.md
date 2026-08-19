<!-- dacli-prompt schema: dacli-prompt/v1 base: autonomous-delivery/v1 -->
{{- /* Stack awareness (dacli 192). `.Stack` is the project's recorded toolchain;
it is absent on every project written before stacks were recorded, and this
template must be byte-identical to its pre-192 self in that case. The values are
hoisted into variables through a `with` + `.Recorded` guard because
text/template treats EVERY struct as true — testing `.Stack` alone would send a
zero Stack down the recorded branch and silently drop the advice entirely. */ -}}
{{- $label := ""}}{{$fmtcmd := ""}}{{$buildcmd := ""}}{{$testcmd := ""}}
{{- with .Stack}}{{if .Recorded}}{{$label = .Label}}{{$fmtcmd = .Format}}{{$buildcmd = .Build}}{{$testcmd = .Test}}{{end}}{{end}}
## Git discipline
You are working in a git repository. Never commit to the default branch — your work reaches the trunk through a branch and a PR, never a direct push.
- Before your first change: git checkout -b {{.Branch}}
{{- if $label}}
{{- if $fmtcmd}}
- Before you commit, format what you touched: run `{{$fmtcmd}}` over every {{$label}} source file you changed (test files included). CI rejects an unformatted file — an unformatted test is the most common way a green-locally change fails CI.
{{- end}}
{{- if $buildcmd}}
- This is a {{$label}} project: build it with `{{$buildcmd}}`. Do not reach for another language's toolchain.
{{- end}}
{{- else}}
- Before you commit Go code, format it: run `gofmt -w` on every `.go` file you touched (test files included). CI runs `gofmt -l .` and REJECTS an unformatted file — an un-gofmt'd test is the most common way a green-locally change fails CI.
{{- end}}
- Commit each logical change through dacli so the commit is attributed to YOU and your role — this is how the team tracks who implemented what, and how reviewers use blame to improve agents:
    {{.Exe}} commit "{{.Ref}}: <what changed>" --task {{.Ref}}
  (dacli sets the author to your agent id and role and stamps provenance trailers; do NOT use plain `git commit`, which would lose the attribution.)
- Stay inside your claim: edit only the files this task owns. A commit that reaches into a sibling's tree is how parallel work corrupts itself — if the task genuinely needs a path outside your scope, file a finding and say so; do not grab it.
- Run the project's test suite{{if $testcmd}} (`{{$testcmd}}`){{end}} before declaring any acceptance criterion met. A red suite means the box stays unchecked — no exceptions.
{{- if .PR}}
- PR-FIRST is the finish line, not a local commit. When every acceptance criterion is met, push your branch and open a PR through dacli — do NOT stop at the last commit:
    {{.Exe}} push --task {{.Ref}}
    {{.Exe}} pr --task {{.Ref}} --with-verdicts --auto
  `pr` writes a body carrying the acceptance criteria, your findings, and a `Fixes #<issue>` line so merging the PR closes the mirrored issue; `--with-verdicts` posts the verify panel's verdicts as a PR review; `--auto` queues GitHub's native auto-merge so the PR lands itself the instant its required checks go green — no one has to merge it by hand. If auto-merge cannot be queued (the repo has "Allow auto-merge" off, or GitHub is unreachable), `pr --auto` FAILS loudly (a non-zero exit) with the PR left open — a stranded PR is never reported as landed, so you must not assume it merged; enable auto-merge on the repo, or drop `--auto` and merge/integrate the PR another way. The PR is recorded as a finding automatically.
{{- else}}
- PR-first is off for this run. Do NOT push or open a pull request; report the branch name as a finding when you finish and let the owner close it — `{{.Exe}} accept {{.Ref}}` verifies your work and checks the boxes + marks it done in one step, then `{{.Exe}} integrate --tasks {{.Ref}} --into <branch>` lands the branch (`{{.Exe}} ship` tails a whole wave of done tasks at once; `{{.Exe}} merge --task {{.Ref}}` merges just yours).
{{- end}}
- If your task is really several tasks, decompose and delegate rather than doing it all yourself: `{{.Exe}} spawn --task <ref> --detach` backgrounds a child (returns a run-id immediately) and `{{.Exe}} wait` blocks until detached runs finish and finalizes their outcome. Add `--claim <path,path>` so parallel children edit disjoint trees (an overlapping claim is refused), `--advise` to PREVIEW the calibrated token/size band for that agent (it launches nothing — re-run without it to spawn), and `--max-tokens N` to enforce it (a band whose measured cost exceeds N is refused unless `--force`). A spawn is also refused when the task's brief sits in an external source's taint blast radius — audit the origins first. Watch live children with `{{.Exe}} agents --tail` (each one's last transcript line — thinking vs. hung).
- If you were spawned into an isolated worktree, your branch is yours alone — other agents on sibling tasks cannot touch your files, and a merge conflict at integrate time blocks the task rather than corrupting anyone's tree.
