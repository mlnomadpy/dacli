---
id: f-runtime-sandbox-probe-cache-trusts-unchanged-metadata-after-executable-bytes
kind: note
note_kind: finding
created: 2026-08-12T13:54:02Z
created_by: a-codex-loop-auditor-8f0nb8
about: "[[303]]"
severity: major
---
# Runtime sandbox probe cache trusts unchanged metadata after executable bytes change
Confirmed against internal/store/runtimefiles.go:403-414 and :416-433. runtimeProbeFingerprint hashes the configured binary string, resolved path, SandboxRO, file size, and ModTime, but not executable contents or a stable install identity. On this host, stat reports /usr/bin/true and /usr/bin/false both size 84128 and mtime 1779353822 while shasum -a 256 reports different digests (a73efc... vs 29a84c...). Copying either in-place with metadata preserved therefore leaves the fingerprint unchanged although the executable bytes changed; HydrateRuntimeROProbe will reuse a cached RuntimeROVerified verdict and sandboxFor can authorize an ro spawn without probing the installed binary now at that path. This is a fail-open trust-cache invalidation defect introduced by 365, not covered by tasks 364-373.
