package cloud_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecoveryRunbookAndSmokeHarnessStayFailClosed(t *testing.T) {
	runbook := readRecoveryFile(t, filepath.Join("..", "docs", "CONTROL_PLANE_RECOVERY.md"))
	for _, required := range []string{
		"Control-plane recovery and retention runbook v1", "PostgreSQL 17.6",
		"recovery point objective", "recovery time objective", "Never restore over it",
		"NOBYPASSRLS", "last_sequence", "replay_floor", "idempotency",
		"optimistic resource", "does not claim immediate erasure from backups",
		"Service credential leaked", "Tenant-isolation suspected",
		"Poisoned/dead-letter queue", "Cursor lost or corrupt",
		"not a production backup service",
	} {
		if !strings.Contains(runbook, required) {
			t.Errorf("recovery runbook lacks %q", required)
		}
	}

	harness := readRecoveryFile(t, filepath.Join("..", "scripts", "control-plane-recovery-smoke.sh"))
	for _, required := range []string{
		"postgres:17.6-bookworm", "refusing unreviewed PostgreSQL image",
		"pg_dump --format=custom --no-owner", "pg_restore --exit-on-error --no-owner",
		"controlplane_schema_migrations", "SOURCE_LEDGER", "RESTORED_LEDGER",
		"SOURCE_RECOVERY_STATE", "RESTORED_RECOVERY_STATE", "NOBYPASSRLS",
		"set_config('dacli.tenant_id'", "trap cleanup EXIT INT TERM",
		"docker exec --interactive",
	} {
		if !strings.Contains(harness, required) {
			t.Errorf("recovery harness lacks %q", required)
		}
	}
	for _, forbidden := range []string{"--clean", "DROP DATABASE", "TRUNCATE", "controlplane_envelope_inbox_identities WHERE", "controlplane_envelope_streams WHERE"} {
		if strings.Contains(harness, forbidden) {
			t.Errorf("recovery harness contains destructive/identity-reset operation %q", forbidden)
		}
	}
}

func readRecoveryFile(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
