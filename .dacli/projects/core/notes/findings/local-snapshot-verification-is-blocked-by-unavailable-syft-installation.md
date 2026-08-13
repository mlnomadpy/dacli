---
id: f-local-snapshot-verification-is-blocked-by-unavailable-syft-installation
kind: note
note_kind: finding
created: 2026-08-13T19:44:33Z
created_by: a-codex-maintainer-9gwn2s
about: "[[429]]"
severity: major
---
# Local snapshot verification is blocked by unavailable Syft installation
syft is absent locally; go install github.com/anchore/syft/cmd/syft@v1.50.0 failed because proxy.golang.org DNS could not resolve. The existing scripts/verify-release-artifacts.sh cannot run until Syft is available, so acceptance criterion 3 remains unverified locally.
