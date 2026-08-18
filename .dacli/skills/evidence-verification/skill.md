---
name: evidence-verification
version: v1
description: Use when implementing, reviewing, or accepting changes that need reproducible evidence, mutation testing, and honest verification records.
min_delivery: inline
created_by: a-root
---
# evidence-verification
Reproduce the premise before changing code. Add the narrowest regression and state the one-line mutation that makes it fail. Verify targeted behavior, race-sensitive packages when applicable, and the repository quality bar. Capture the command's own exit code. Never turn unreadable state into success, weaken a correct test, or claim an unrun check. Record residual uncertainty and keep acceptance independent from implementation.
