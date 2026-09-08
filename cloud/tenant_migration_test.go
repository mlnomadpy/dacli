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
