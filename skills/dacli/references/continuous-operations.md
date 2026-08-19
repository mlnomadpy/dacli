# Continuous operations and recovery

Long-running means many finite cycles. Use a rolling budget, fixed WIP, idle backoff, and `touch .dacli/STOP` for a checkpointed stop. Leases/heartbeats are the live-agent/run record observed by `agents`, `runs show`, and `loop status`; do not reclaim a writer merely because a transcript is quiet. The cycle and landing journals are recovery ledgers: inspect status, run/PR state, and trunk before resuming a partial cycle.

Circuit breakers/cooldowns stop retry storms; terminal failures become blocked or dead-lettered work rather than invisible retries. Record observability in the event log, transcripts, findings, `doctor`, and `retro`. Recover by fixing the named condition, then resume the bounded transaction. Publication remains default-off: a loop may prepare and land ordinary PRs under policy, but never publishes a release without explicit authority.
