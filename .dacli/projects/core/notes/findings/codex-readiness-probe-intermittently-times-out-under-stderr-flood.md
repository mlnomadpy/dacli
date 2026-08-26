---
id: f-codex-readiness-probe-intermittently-times-out-under-stderr-flood
kind: note
note_kind: finding
created: 2026-08-22T22:05:43Z
created_by: a-adversarial-reviewer-h5rnfr
about: "[[t-01KZXRP56538HNHDP3P4FJHGGP]]"
severity: major
---
# Codex readiness probe intermittently times out under stderr flood
internal/features/execution/behavioral_preflight.go:137-183 concurrently scans stdout and drains stderr under one bounded deadline; internal/features/execution/preflight_test.go:254-271 supplies 5000 stderr warnings followed by a fragmented valid turn.started event. GOCACHE=/private/tmp/dacli-go-cache go test ./internal/features/execution -run 'TestCodexBehavioralPreflightReadinessStopsAndReapsHangingTree$' -count=10 failed 6/10 at line 271, returning transient/transport 'behavioral launch readiness exceeded bounded deadline' instead of LaunchCompatible. This makes runtime launch eligibility disagree with observed valid readiness under noisy startup.
