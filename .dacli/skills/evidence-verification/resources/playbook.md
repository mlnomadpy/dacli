# Evidence and verification playbook

Use this sequence for a behavioral change:

1. Reproduce the user-visible symptom and capture the command's exit status.
2. Add the smallest public or package-level regression that reaches the defect.
3. Mutate the protected branch back to the broken behavior and confirm that
   exact test fails for the intended reason.
4. Restore the fix, run the focused test, then test race-sensitive packages and
   the repository-wide quality gates appropriate to the blast radius.
5. Record commands, artifact/commit identity, anything skipped, and why.

Prefer invariant tests when a rule spans a command registry or many handlers.
An error assertion must identify the expected error or message; “some error” is
usually not evidence. Never certify absence after a failed read. Acceptance is
an independent statement that the delivered artifact, not merely a working
directory, meets every criterion.
