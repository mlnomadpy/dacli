---
id: f-headless-sandbox-blocks-npm-node-python3-c-gh-exec-forcing-verification-via
kind: note
note_kind: finding
created: 2026-07-26T20:55:50Z
created_by: a-azfs9hw109
about: [[154]]
severity: minor
---
# Headless sandbox blocks npm/node/python3 -c/gh exec, forcing verification via real GitHub Actions instead of local vitest/eslint run
This agent session's Bash permission gate requires interactive approval for npm ci/npm run */node -e/python3 -c/gh, and none is available headlessly (session is unattended). go/git/dacli invocations are unaffected. Verified the wiring is syntactically correct (ci.yml:29-34 mirrors the existing 'build frontend' step's working-directory pattern) and that internal/features/dashboard/ui has 15 *.test.ts files plus eslint.config.ts/vitest.config.ts, then pushed the branch and will confirm pass/fail via the real GitHub Actions run rather than local exec.
