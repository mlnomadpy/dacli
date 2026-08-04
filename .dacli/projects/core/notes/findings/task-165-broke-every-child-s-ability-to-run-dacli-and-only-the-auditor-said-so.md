---
id: f-task-165-broke-every-child-s-ability-to-run-dacli-and-only-the-auditor-said-so
kind: note
note_kind: finding
created: 2026-08-04T14:05:08Z
created_by: a-root
severity: major
origin: internal/features/execution/execution.go:1439
---
# Task 165 broke every child's ability to run dacli, and only the auditor said so
protocolPreamble tells the child which dacli to invoke using os.Executable() — the path of the dacli that spawned it. The runtime allowlist names an absolute path. Those two have to agree, and nothing checks that they do.

Task 165 moved the allowlist from <repo>/dacli to ~/go/bin/dacli. I kept spawning with ./dacli, so every child after that change was told to run /Users/tahabsn/Documents/GitHub/dacli/dacli — a path no longer on its allowlist. Claude Code answered 'requires approval' to every dacli invocation, with no approver present in a headless run.

The agent on task 264 reported it in plain terms: 'The dacli binary is sandbox-blocked in this headless run — every dacli invocation (whoami, note add, task list) returns requires approval with no approver present (prior siblings hit the same wall).' It did the audit anyway, read the backlog by opening task files directly to check for duplicates, found a real defect, and handed me the exact command to file it with. That is the correct behaviour under a broken tool and it is the only reason the finding survived.

'Prior siblings hit the same wall' is the part that stings: the agent on task 200 completed and wrote nothing, and I read that as the agent failing. It was this. I re-did 200 by hand instead of investigating, so the same regression cost a second run before anyone named it.

Two things follow. Operationally, spawn with the installed binary — the one the allowlist names — not ./dacli. Structurally, filed as 267: spawn should refuse, or at least warn loudly, when os.Executable() is not a path the runtime permits. It is the same family as 250 (a role's grant and its runtime disagreeing) and it fails the same way: silently, one wasted run at a time.
