# Runtime and process-safety playbook

For every coding CLI adapter, record and probe separately: binary/version,
prompt transport, model flag, result channel, usage fidelity, timeout,
cancellation, read-only enforcement, workspace-write access, and exit mapping.
A declared capability is not a verified one, and installation is not proof of
authentication.

Make configuration provenance explicit. The invocation record should name
which system/user/project instructions, plugins, MCP servers, skills, and
environment variables were admitted. An isolation flag must be behaviorally
tested; unexpected global context is a finding, not harmless decoration.

Track the whole process group, guard against PID reuse, and persist terminal
state atomically before releasing claims. Startup failure, cancellation, and
timeout need the same durable result path as success. A terminal child must
have an audited worktree handoff/reclaim path; never require recovery through a
lost one-time credential.
