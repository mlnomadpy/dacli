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
