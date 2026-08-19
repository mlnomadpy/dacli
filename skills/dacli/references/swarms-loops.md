# Waves, swarms, supervision, and loops

## Contents

- Preparing a wave
- Claims and worktrees
- Spawning and finalizing
- Verification
- Governed loops
- Monitoring and stopping

## Preparing a wave

Parallelize only tasks whose dependencies and write paths are independent:

```bash
dacli task list --status open --project <project>
dacli critical-path --project <project>
dacli next --project <project> --parallel <width>
dacli team assign <ref>
dacli context <ref>
```

Estimate tasks first. Without estimates, scheduling falls back to priority and
sequence and cannot expose meaningful critical-path slack.

For each candidate, identify the smallest truthful code-path claim. If claims
overlap, schedule sequential waves. Do not dishonestly narrow a claim merely to
bypass a conflict refusal.

## Claims and worktrees

Use both mechanisms for parallel write agents:

- `--worktree` creates a task branch and isolated current checkout.
- `--claim path` reserves the future merge surface against live agents and
  constrains `dacli commit` staging.

```bash
dacli spawn --task <ref> --role <role> --grant rw --worktree \
  --claim internal/feature --pr --detach
```

Claim package or feature roots, not the entire repository. Include tests and
fixtures the task is expected to modify. Current claim matching has open defects
for some glob/directory presentations (#629 and #641); if a truthful claim is
refused, capture the exact symptom and use the named safe recovery rather than
bypassing claim enforcement.

Inside a dacli worktree, edit relative to the worktree's current directory.
Absolute paths pointing to the main checkout can put changes on the wrong
branch. The shared `.dacli` record intentionally resolves to the main workspace.

## Spawning and finalizing

Start detached agents only after preflight:

```bash
dacli preflight --role <role>
dacli spawn --task <ref> --role <role> --worktree \
  --claim <path> --pr --detach
dacli agents --tail
dacli logs <run-id-prefix|child-id> -f
dacli catchup --since 20m
dacli wait
dacli sync
```

Use `agents` and `logs` to monitor. Use `wait` to finalize: it classifies the
outcome and records silent/no-result runs. A finished process that has not been
wait-finalized may still occupy operator attention and leave an incomplete
record.

Read-only agents propose workspace mutations as events. Run `sync` before
concluding that an auditor produced nothing. Retire completed agents when a WIP
slot remains occupied:

```bash
dacli agent retire <agent-id>
```

## Verification

Require evidence at three levels:

1. Task-local acceptance criteria.
2. Targeted regression tests that fail when the protected behavior is broken.
3. Repository-wide quality gates appropriate to the change's blast radius.

Use `dacli verify --task <ref> --panel <runtime-a>,<runtime-b> --require 2`
for an adversarial panel. Prefer heterogeneous runtimes/models for independent
failure modes. Do not let the implementing agent be the only certifier when
independence is warranted; enforce that owner gate with
`dacli accept <ref> --require-independent --verify "<command>"` after the
panel and landing evidence exist.

Do not weaken or remove a failing test to make a wave green. If the test is
wrong, explain the false premise and replace it with a test that fails under a
specific relevant mutation.

## Governed loops

For GitHub-first collaboration, configure `landing.mode: pr` and its base when
required checks, CODEOWNERS/reviews, and a human-visible PR are part of the
landing contract. The loop resolves CLI override > project configuration >
legacy local default, previews the result, and journals it across restarts.
Remote failures leave tasks open; repair GitHub access and rerun so canonical
branches and existing PRs are reused.

Preview one cycle before unattended operation:

```bash
dacli loop --project <project> --width <n> --impl-role <implementer> \
  --review-role <reviewer> --max-cycles <n> --dry-run
dacli loop --project <project> --width <n> --impl-role <implementer> \
  --review-role <reviewer> --advise
```

Then choose explicit governance:

```bash
dacli loop --project <project> --width <n> --impl-role <implementer> \
  --review-role <reviewer> --max-cycles <n>

dacli loop --project <project> --width <n> --impl-role <implementer> \
  --review-role <reviewer> --yolo \
  --max-cycles <n> --window-tokens <tokens> --token-window 24h \
  --halt-after-idle 3
```

One cycle plans ready work, spawns implementers, waits/tests, lands, runs the
continuous-improvement reviewer, and records a retro. The review phase may file
the next evidence-backed task; it must not invent work when evidence is empty.

Use `--worker-timeout` when task estimates imply that the derived timeout is
inappropriate. Keep width within role WIP, machine capacity, token budget, and
the number of truly disjoint claims. More workers do not help when they contend
on the same package or CI bottleneck.

Run landing loops that share one repository/trunk sequentially unless repository
policy explicitly proves concurrent integration is safe. Project flags scope
scheduling; they do not give each project a separate Git history, event log, or
security boundary.

The stock `--review-role` is the cycle's post-land continuous-improvement
reviewer. It is not a blocking security approval for every implementation PR.
For pre-merge security review, make a cross-provider reviewer or external check
a required GitHub gate, or explicitly spawn a reviewer before integration:

```bash
dacli spawn --task <ref> --role <security-reviewer> \
  --review --pr-number <number> --advise
# inspect the preview, then repeat without --advise; wait and sync before land
```

One loop uses one explicit `--impl-role`. To guarantee multiple providers in a
single project, run explicit provider-assigned waves or separate bounded loop
runs; do not assume one loop alternates runtimes automatically.

## Monitoring and stopping

```bash
dacli loop status --project <project>
dacli agents --tail
dacli pr status --task <ref>
dacli events tail
```

The loop measures progress by trunk advancement, not by optimistic task-status
changes. Auto-merge can land after the spawning cycle; inspect PR and trunk
state before diagnosing a no-progress halt.

Stop at the next safe checkpoint:

```bash
touch .dacli/STOP
```

The stop file latches. Remove it only after inspecting state and intentionally
resuming. Use the no-progress halt as a thrash guard, not as evidence that no
work was attempted.

After a wave, preview cleanup:

```bash
dacli worktree prune --into main --dry-run
dacli worktree prune --into main
dacli runs prune --keep 20
```

Never delete a worktree with unlanded material merely because its process has
finished. Review the dry-run classification and PR/task state first.
