# GitHub delivery playbook

Create or confirm a detailed issue before implementation: symptom, reproduction,
risk, suspected cause, workaround, design constraints, and checkable acceptance.
Adopt it into dacli so one identity connects issue, task, branch, run, commit,
PR, checks, and close.

Preview broad pull/push/sync operations. Remote creates need stable markers and
lookup-before-create recovery so a lost response cannot duplicate work. After
any interrupted command, inspect GitHub directly; an exit code or plan line is
not proof that every mutation happened.

Use a narrow branch and PR. Review the diff against acceptance, wait for all
required checks, and confirm the merge commit is on trunk. Only then accept the
task and close/synchronize its issue. Hosted GitHub App credentials are an
optional control-plane adapter, not a requirement for local-first operation.
