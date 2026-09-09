package cloud_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestTenantMigrationEnforcesCompositeScopeAndRLS(t *testing.T) {
	raw, err := os.ReadFile("migrations/0003_tenant_repository.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	tables := []string{
		"organizations", "teams", "memberships", "devices", "projects", "environments", "tenant_audit_events",
	}
	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			name := "controlplane_" + table
			if !strings.Contains(sql, "CREATE TABLE "+name+" (") {
				t.Fatalf("%s is absent", name)
			}
			if !strings.Contains(sql, "ALTER TABLE "+name+" ENABLE ROW LEVEL SECURITY;") ||
				!strings.Contains(sql, "ALTER TABLE "+name+" FORCE ROW LEVEL SECURITY;") {
				t.Fatalf("%s does not force RLS", name)
			}
			policy := regexp.MustCompile(`(?s)CREATE POLICY ` + regexp.QuoteMeta(name) + `_tenant ON ` + regexp.QuoteMeta(name) + `.*?USING \(tenant_id = NULLIF\(current_setting\('dacli.tenant_id', true\), ''\)\).*?WITH CHECK \(tenant_id = NULLIF\(current_setting\('dacli.tenant_id', true\), ''\)\);`)
			if !policy.MatchString(sql) {
				t.Fatalf("%s lacks the closed tenant policy", name)
			}
		})
	}
	if !strings.Contains(sql, "controlplane_device_sessions") {
		raw, err := os.ReadFile("migrations/0004_device_sessions.sql")
		if err != nil {
			t.Fatal(err)
		}
		sessionSQL := string(raw)
		for _, required := range []string{
			"UNIQUE (tenant_id, credential_digest)",
			"FOREIGN KEY (tenant_id, account_id)",
			"FOREIGN KEY (tenant_id, device_id)",
			"ALTER TABLE controlplane_device_sessions FORCE ROW LEVEL SECURITY",
			"current_setting('dacli.tenant_id', true)",
		} {
			if !strings.Contains(sessionSQL, required) {
				t.Errorf("device-session migration lacks %q", required)
			}
		}
	}

	for _, table := range []string{"teams", "memberships", "devices", "projects", "environments", "tenant_audit_events"} {
		section := tableSection(t, sql, "controlplane_"+table)
		if !strings.Contains(section, "PRIMARY KEY (tenant_id,") {
			t.Errorf("controlplane_%s does not use a composite tenant key", table)
		}
	}
	if !strings.Contains(sql, "BEFORE UPDATE OR DELETE ON controlplane_tenant_audit_events") {
		t.Fatal("tenant audit stream can be mutated")
	}
}

func TestTenantRelationshipsCarryTenantIdentity(t *testing.T) {
	raw, err := os.ReadFile("migrations/0003_tenant_repository.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, relationship := range []string{
		"FOREIGN KEY (tenant_id, account_id)\n        REFERENCES controlplane_memberships (tenant_id, account_id)",
		"FOREIGN KEY (tenant_id, team_id)\n        REFERENCES controlplane_teams (tenant_id, team_id)",
		"FOREIGN KEY (tenant_id, project_id)\n        REFERENCES controlplane_projects (tenant_id, project_id)",
	} {
		if !strings.Contains(sql, relationship) {
			t.Errorf("tenant-scoped relationship absent: %s", relationship)
		}
	}
}

func TestTenantWorkflowMigrationIsScopedAndRecoverable(t *testing.T) {
	raw, err := os.ReadFile("migrations/0005_tenant_workflows.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, table := range []string{"invitations", "project_assignments", "environment_assignments"} {
		name := "controlplane_" + table
		if !strings.Contains(sql, "CREATE TABLE "+name+" (") ||
			!strings.Contains(sql, "ALTER TABLE "+name+" FORCE ROW LEVEL SECURITY;") ||
			!strings.Contains(sql, "CREATE POLICY "+name+"_tenant ON "+name) {
			t.Errorf("%s lacks a forced tenant boundary", name)
		}
	}
	for _, required := range []string{
		"UNIQUE (tenant_id, token_digest)",
		"FOREIGN KEY (tenant_id, project_id, environment_id)",
		"REFERENCES controlplane_memberships (tenant_id, account_id)",
		"CHECK (target_kind BETWEEN 1 AND 9)",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("tenant workflow migration lacks %q", required)
		}
	}
	if strings.Contains(strings.ToUpper(sql), "DELETE FROM") {
		t.Fatal("tenant workflow migration introduces a hard-delete surface")
	}
}

func TestStablePaginationMigrationUsesImmutableMonotonicOrdinals(t *testing.T) {
	raw, err := os.ReadFile("migrations/0006_stable_pagination.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, table := range []string{"projects", "environments"} {
		for _, required := range []string{
			"ALTER TABLE controlplane_" + table,
			"ADD COLUMN list_ordinal BIGINT GENERATED ALWAYS AS IDENTITY",
			"UNIQUE (tenant_id, list_ordinal)",
			"ON controlplane_" + table + " (tenant_id, list_ordinal)",
		} {
			if !strings.Contains(sql, required) {
				t.Errorf("%s stable pagination lacks %q", table, required)
			}
		}
	}
	if strings.Contains(sql, "UPDATE controlplane_") {
		t.Fatal("pagination migration rewrites stable ordinals")
	}
}

func TestEnvelopeWorkerMigrationKeepsDurableIdentityAndTenantBoundaries(t *testing.T) {
	raw, err := os.ReadFile("migrations/0007_envelope_worker.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, table := range []string{"envelope_streams", "envelope_inbox_identities", "envelope_inbox_payloads", "envelope_outbox", "envelope_audit_events"} {
		name := "controlplane_" + table
		if !strings.Contains(sql, "CREATE TABLE "+name+" (") || !strings.Contains(sql, "ALTER TABLE "+name+" ENABLE ROW LEVEL SECURITY;") || !strings.Contains(sql, "ALTER TABLE "+name+" FORCE ROW LEVEL SECURITY;") || !strings.Contains(sql, "CREATE POLICY "+name+"_tenant ON "+name) {
			t.Errorf("%s lacks a forced tenant boundary", name)
		}
	}
	for _, required := range []string{
		"PRIMARY KEY (tenant_id, project_id, producer_key_id)",
		"UNIQUE (tenant_id, project_id, idempotency_key)",
		"UNIQUE (tenant_id, project_id, producer_key_id, producer_sequence)",
		"BEFORE UPDATE OR DELETE ON controlplane_envelope_inbox_identities",
		"BEFORE DELETE ON controlplane_envelope_streams",
		"BEFORE UPDATE OR DELETE ON controlplane_envelope_audit_events",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("envelope worker migration lacks %q", required)
		}
	}
	identitySection := tableSection(t, sql, "controlplane_envelope_inbox_identities")
	if strings.Contains(identitySection, "expires_") {
		t.Fatal("durable inbox identity is coupled to payload retention")
	}
	if strings.Contains(sql, "DELETE FROM controlplane_envelope_inbox_identities") || strings.Contains(sql, "DELETE FROM controlplane_envelope_streams") {
		t.Fatal("migration can erase replay or idempotency state")
	}
}

func TestMutationAuditMigrationBackfillsExactIdentityAndClosedResult(t *testing.T) {
	raw, err := os.ReadFile("migrations/0008_mutation_audit_identity.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, required := range []string{
		"ADD COLUMN action_digest BYTEA",
		"DISABLE TRIGGER controlplane_tenant_audit_append_only",
		"SET action_digest = after_digest",
		"ENABLE TRIGGER controlplane_tenant_audit_append_only",
		"ALTER COLUMN action_digest SET NOT NULL",
		"CHECK (octet_length(action_digest) = 32)",
		"CHECK (result IN ('succeeded', 'refused', 'conflict', 'failed'))",
		"CHECK (char_length(reason) BETWEEN 1 AND 64)",
		"ALTER COLUMN result DROP DEFAULT",
		"ALTER COLUMN reason DROP DEFAULT",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("mutation audit migration lacks %q", required)
		}
	}
}

func tableSection(t *testing.T, sql, table string) string {
	t.Helper()
	start := strings.Index(sql, "CREATE TABLE "+table+" (")
	if start < 0 {
		t.Fatalf("table %s absent", table)
	}
	end := strings.Index(sql[start:], ");")
	if end < 0 {
		t.Fatalf("table %s is unterminated", table)
	}
	return sql[start : start+end+2]
}
