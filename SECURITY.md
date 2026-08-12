# Security policy

## Reporting a vulnerability

Report privately through GitHub's
[private vulnerability reporting](https://github.com/mlnomadpy/dacli/security/advisories/new).
Please do not open a public issue for anything exploitable.

Include what you would put in a bug report — the observed behaviour, how to
reproduce it, and what you had to do by hand — plus what an attacker gains.

## What is in scope

dacli runs agent processes, shells out to `git` and `gh`, and writes files under
a workspace path. The things most worth reporting:

- **Capability escape.** A read-only agent causing a write it should not be able
  to make — to the workspace, the repository, the machine, or a remote. The
  grant model is cooperative by design (see `internal/agentid`), and
  `Command.Mutates` is the enforcement point; a mutating command that does not
  declare itself is a real finding.
- **Path traversal.** A task ref, project slug, role name or skill name that
  escapes the workspace directory.
- **Argument or command injection** into a spawned `git`, `gh` or runtime
  invocation. Subprocesses are invoked with an argv slice and never through a
  shell — a path that breaks that property is in scope.
- **Token disclosure.** An agent token reaching a log, a commit, an issue, or a
  brief assembled for a different agent.
- **Unconsented outward action.** Anything that pushes, publishes, opens an
  issue or creates a repository without the operator having granted it.

## What is not

- Prompt injection *through content an agent was asked to read*. An agent that
  acts on instructions inside a task it was told to read is doing what it was
  told; the mitigation is the untrusted-content boundary in the brief, and
  improvements there are welcome as ordinary issues.
- A runtime that grants an agent broad tool access. That is an operator choice,
  and dacli records it rather than preventing it.
- Findings from a scanner with no reachable path demonstrated.

## Supported versions

dacli is pre-1.0. Only the latest release and `main` are supported; fixes land
on `main` and may be included in the next tagged release. There are no
backports to older tags.
