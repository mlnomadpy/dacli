# Continuous operations and recovery

Long-running means many finite cycles. Use a rolling budget, fixed WIP, idle backoff, and `touch .dacli/STOP` for a checkpointed stop. Leases/heartbeats are represented by the live agent and run records observed with `agents`, `runs list`, `runs show`, and `loop status`; do not reclaim a writer merely because a transcript is quiet. The cycle and landing journals are recovery ledgers: after a restart, inspect status, the recorded runs, PR state, and trunk before resuming a partial cycle.

Shipped local safeguards include finite cycle limits, rolling token windows, idle/no-progress halts, runtime health cooldowns, STOP, and durable run/landing records. No dedicated runtime-cooldown clear or expiry command is shipped. Record observability in the event log, transcripts, findings, `doctor`, and `retro`. Recover by fixing the named condition, then resume the bounded transaction.

An always-on service additionally needs circuit breakers across repeated failures and a dead-letter queue for terminal work. Those are future control-plane requirements, not shipped `dacli` commands. Publication remains default-off: a loop may prepare and land ordinary PRs under policy, but never publishes a release without explicit authority.
