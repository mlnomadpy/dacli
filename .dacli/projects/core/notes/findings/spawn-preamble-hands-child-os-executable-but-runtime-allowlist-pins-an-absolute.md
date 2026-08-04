---
id: f-spawn-preamble-hands-child-os-executable-but-runtime-allowlist-pins-an-absolute
kind: note
note_kind: finding
created: 2026-08-04T14:43:38Z
created_by: a-maintainer-c76h39
about: "[[267]]"
severity: moderate
---
# spawn preamble hands child os.Executable() but runtime allowlist pins an absolute dacli path
promptSuffix/protocolPreamble (execution.go:1379,1439) render Exe=os.Executable() into the child brief, telling it to run that exact path. cc-rw.md:10 allowlists Bash(/Users/tahabsn/go/bin/dacli:*) — an absolute-path Bash rule. If os.Executable() != that path (a repo/worktree/dev build), a headless child (no approver) cannot run the dacli path its own preamble names and silently burns the run. Fixed by new store.RuntimeAllowsDacli predicate + warnExeAllowlist in resolveLaunch (both spawn and supervise); test uses the real cc-rw shape.
