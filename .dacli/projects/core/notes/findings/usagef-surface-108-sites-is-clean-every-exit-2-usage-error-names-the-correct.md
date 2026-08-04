---
id: f-usagef-surface-108-sites-is-clean-every-exit-2-usage-error-names-the-correct
kind: note
note_kind: finding
created: 2026-08-04T18:18:12Z
created_by: a-go-auditor-z48ata
about: "[[t-01KZ6S9Z9ZXRW12XZA1GJJSGB6]]"
source_event: 01KZ6SNRQZ25NCZS1Y0W3QZ8AZ
---
# Usagef surface (108 sites) is clean: every exit-2 usage error names the correct invocation
Completes criterion-1 coverage ('every clikit.Refusedf AND Usagef checked'). Surveyed all 108 non-test clikit.Usagef call sites. All are genuine caller-syntax mistakes correctly at exit 2, and all name the way out:
  - the dominant shape is a full 'usage: dacli <cmd> <args...>' line (e.g. planning.go:156, execution.go:570, teamops.go:356) — the correct invocation IS the remedy;
  - value-validation errors name the fix: wscore.go:41/46 'unknown template %q — available: %s', new.go:190 'run `dacli template list`', new.go:405/412 'pass --stack %s', dashboard.go:86 '--port must be a number', planning.go:563 '--term requires --def', lifecycle.go:763 '--approve and --request-changes are opposites; pass one';
  - the not-linked family (ghmirror.go:204/557, project.go:405, codeowners.go:190, release.go:58) all name '`dacli github link %s` first'.
Two `Usagef("%v", err)` wrappers (planning.go:454 estimate parse, shortcuts.go:204 shortcut.Expand) defer to the wrapped error's text, both caller-input parse failures, correctly exit 2. No Usagef misclassifies a policy refusal as exit 2, and none is a dead end. Only gap worth noting is stylistic: some usage lines lead with the bare 'usage:' form without a one-clause statement of what was wrong, but the usage line itself is the remedy.
