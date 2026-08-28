---
id: f-runtime-launch-ignores-transcript-creation-failure-and-runs-without-durable
kind: note
note_kind: finding
created: 2026-08-28T00:24:55Z
created_by: a-adversarial-reviewer-yth0jg
about: "[[t-01KZXRP56538HNHDP3P4FJHGGP]]"
severity: major
---
# Runtime launch ignores transcript creation failure and runs without durable evidence
internal/features/execution/execution.go:1998-2001 assigns sink, _ = os.Create(transcriptPath), discarding the creation error. Trigger: the run directory or transcript path becomes unwritable/missing between run setup and execRuntime (permission drift, cleanup race, or storage failure). Because sink remains nil, both detached and foreground branches still start the guardian and return launch success while writing no transcript; agents --tail, wait activity detection, usage capture, recovery evidence, and reporting then disagree with the executed runtime. The full suite passes under GOCACHE=/tmp/dacli-go-cache go test ./..., and no open/active core task names this transcript-creation failure. Requested outcome: surface transcript creation failure before starting the runtime and regress both detached and foreground paths.
