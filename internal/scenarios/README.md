# Release-readiness scenarios

Run the complete deterministic, offline scenario suite with:

```bash
go test -v ./internal/scenarios
```

The suite drives the real `dacli` binary in disposable Git repositories. Each
scenario asserts an end state, not how many commands happened to run:

- feature work reaches `main` and closes its task;
- a regression repair preserves and passes the reproducing test;
- a failed prerequisite keeps its dependent work out of the ready queue;
- conflicting edits leave `main` clean and block the unmerged task; and
- externally sourced malicious instructions refuse execution and create no
  attacker-controlled file.

`TestScenarioAssertionMutations` runs a deliberately broken variant of every
fixture and requires its outcome assertion to fail. This is the mutation proof
for the fixtures themselves; adding a scenario without a detectable mutation
makes the suite fail.
