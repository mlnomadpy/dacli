package tenantrepo

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mlnomadpy/dacli/cloud/internal/migrations"
	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

func TestOpenRequiresExactSchema(t *testing.T) {
	for _, version := range []int{0, SchemaVersion - 1, SchemaVersion + 1} {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(MAX(version), 0) FROM controlplane_schema_migrations`)).
			WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(version))
		if _, err := Open(context.Background(), db); !errors.Is(err, ErrUnsupportedSchema) {
			t.Fatalf("Open(schema %d) = %v", version, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
		_ = db.Close()
	}
}

func TestOpenAcceptsHighestShippedMigrationVersion(t *testing.T) {
	catalog, err := migrations.Load(os.DirFS("../../migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) == 0 || catalog[len(catalog)-1].Version != SchemaVersion {
		t.Fatalf("repository schema=%d catalog=%+v", SchemaVersion, catalog)
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(MAX(version), 0) FROM controlplane_schema_migrations`)).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(SchemaVersion))
	if _, err := Open(context.Background(), db); err != nil {
		t.Fatalf("Open(latest schema) = %v", err)
	}
	assertExpectations(t, mock)
}

func TestProjectBindsTenantInTransactionAndQuery(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")

	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`SELECT tenant_id, project_id, name, state, version\s+FROM controlplane_projects WHERE tenant_id = \$1 AND project_id = \$2`).
		WithArgs("tenant-a", "same-id").
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "project_id", "name", "state", "version"}).
			AddRow("tenant-a", "same-id", "A", 1, 3))
	mock.ExpectCommit()

	project, err := repo.Project(context.Background(), scope, tenant.ProjectID("same-id"))
	if err != nil {
		t.Fatal(err)
	}
	if project.Tenant != scope.Organization || project.Name != "A" || project.Version != 3 {
		t.Fatalf("project = %+v", project)
	}
	assertExpectations(t, mock)
}

func TestProjectCrossTenantLookupIsIndistinguishableFromMissing(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-b")
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_projects WHERE tenant_id = \$1 AND project_id = \$2`).
		WithArgs("tenant-b", "same-id").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, err := repo.Project(context.Background(), scope, tenant.ProjectID("same-id"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant lookup = %v", err)
	}
	assertExpectations(t, mock)
}

func TestCreateProjectCommitsStateAndAuditTogether(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	project := tenant.Project{Tenant: scope.Organization, ID: "project-1", Name: "One", State: tenant.LifecycleActive, Version: 1}

	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectExec(`INSERT INTO controlplane_projects`).
		WithArgs(project.Tenant, project.ID, project.Name, project.State, project.Version).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).
		WithArgs(project.Tenant, "corr-1", tenant.AccountID("actor-1"), tenant.DeviceID("device-1"), tenant.AuditActionCreate,
			tenant.TargetProject, "project-1", tenant.Version(0), tenant.Version(1), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "succeeded", "committed", int64(1234)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.CreateProject(context.Background(), scope, mutation("corr-1"), project); err != nil {
		t.Fatal(err)
	}
	assertExpectations(t, mock)
}

func TestCreateProjectRollsBackWhenAuditAppendFails(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	project := tenant.Project{Tenant: scope.Organization, ID: "project-1", Name: "One", State: tenant.LifecycleActive, Version: 1}

	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectExec(`INSERT INTO controlplane_projects`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnError(errors.New("audit unavailable"))
	mock.ExpectRollback()

	err := repo.CreateProject(context.Background(), scope, mutation("corr-rollback"), project)
	if err == nil || !regexp.MustCompile(`append tenant audit event`).MatchString(err.Error()) {
		t.Fatalf("CreateProject = %v", err)
	}
	assertExpectations(t, mock)
}

func TestCreateProjectMapsDuplicateToStableConflict(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	project := tenant.Project{Tenant: scope.Organization, ID: "project-1", Name: "One", State: tenant.LifecycleActive, Version: 1}

	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectExec(`INSERT INTO controlplane_projects`).WillReturnError(postgresError{state: "23505"})
	mock.ExpectRollback()

	if err := repo.CreateProject(context.Background(), scope, mutation("corr-duplicate"), project); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate CreateProject = %v", err)
	}
	assertExpectations(t, mock)
}

func TestUpdateProjectUsesOptimisticVersionAndAudits(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	project := tenant.Project{Tenant: scope.Organization, ID: "project-1", Name: "Renamed", State: tenant.LifecycleActive, Version: 2}

	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_projects WHERE tenant_id = \$1 AND project_id = \$2`).
		WithArgs("tenant-a", "project-1").
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "project_id", "name", "state", "version"}).
			AddRow("tenant-a", "project-1", "Before", 1, 1))
	mock.ExpectExec(`UPDATE controlplane_projects`).
		WithArgs(project.Tenant, project.ID, project.Name, project.State, project.Version, tenant.Version(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.UpdateProject(context.Background(), scope, mutation("corr-update"), 1, project); err != nil {
		t.Fatal(err)
	}
	assertExpectations(t, mock)
}

func TestUpdateProjectRefusesStaleVersionBeforeWrite(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	project := tenant.Project{Tenant: scope.Organization, ID: "project-1", Name: "Stale", State: tenant.LifecycleActive, Version: 2}

	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_projects WHERE tenant_id = \$1 AND project_id = \$2`).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "project_id", "name", "state", "version"}).
			AddRow("tenant-a", "project-1", "Current", 1, 4))
	mock.ExpectRollback()

	if err := repo.UpdateProject(context.Background(), scope, mutation("corr-stale"), 1, project); !errors.Is(err, ErrConflict) {
		t.Fatalf("UpdateProject = %v", err)
	}
	assertExpectations(t, mock)
}

func TestArchiveProjectUsesArchiveAuditAction(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	project := tenant.Project{Tenant: scope.Organization, ID: "project-1", Name: "Project", State: tenant.LifecycleArchived, Version: 2}
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_projects WHERE tenant_id = \$1 AND project_id = \$2`).WillReturnRows(
		sqlmock.NewRows([]string{"tenant_id", "project_id", "name", "state", "version"}).AddRow("tenant-a", "project-1", "Project", tenant.LifecycleActive, 1))
	mock.ExpectExec(`UPDATE controlplane_projects`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WithArgs(scope.Organization, "project-archive", sqlmock.AnyArg(), sqlmock.AnyArg(), tenant.AuditActionArchive, tenant.TargetProject, string(project.ID), tenant.Version(1), tenant.Version(2), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "succeeded", "committed", int64(1234)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.UpdateProject(context.Background(), scope, mutation("project-archive"), 1, project); err != nil {
		t.Fatal(err)
	}
	assertExpectations(t, mock)
}

func TestCurrentMembershipIsScopedAndClosed(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_memberships WHERE tenant_id = \$1 AND account_id = \$2`).
		WithArgs("tenant-a", "account-1").
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "account_id", "team_id", "roles", "state", "version", "expires_unix"}).
			AddRow("tenant-a", "account-1", "", []byte(`["developer","reviewer"]`), 1, 2, 0))
	mock.ExpectCommit()

	membership, err := repo.CurrentMembership(context.Background(), scope, "account-1")
	if err != nil {
		t.Fatal(err)
	}
	if !membership.Roles.Has(tenant.RoleDeveloper) || !membership.Roles.Has(tenant.RoleReviewer) || len(membership.StableRoles()) != 2 {
		t.Fatalf("membership roles = %v", membership.StableRoles())
	}
	assertExpectations(t, mock)
}

func TestInvalidScopeAndMutationFailBeforeTransaction(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	if _, err := repo.Project(context.Background(), tenant.Scope{}, "project-1"); err == nil {
		t.Fatal("empty tenant scope was accepted")
	}
	scope := mustScope(t, "tenant-a")
	project := tenant.Project{Tenant: scope.Organization, ID: "project-1", Name: "One", State: tenant.LifecycleActive, Version: 1}
	if err := repo.CreateProject(context.Background(), scope, Mutation{}, project); err == nil {
		t.Fatal("empty mutation identity was accepted")
	}
	assertExpectations(t, mock)
}

func testRepository(t *testing.T) (*Repository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	return &Repository{db: db}, mock, func() { _ = db.Close() }
}

func mustScope(t *testing.T, id string) tenant.Scope {
	t.Helper()
	organization, err := tenant.NewOrganizationID(id)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := tenant.NewScope(organization)
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func mutation(correlation string) Mutation {
	return Mutation{Actor: "actor-1", Device: "device-1", CorrelationID: correlation, OccurredAt: time.UnixMilli(1234)}
}

func expectTenantBinding(mock sqlmock.Sqlmock, scope tenant.Scope) {
	mock.ExpectExec(regexp.QuoteMeta(`SELECT set_config('dacli.tenant_id', $1, true)`)).
		WithArgs(scope.Organization).WillReturnResult(sqlmock.NewResult(0, 1))
}

func assertExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type postgresError struct{ state string }

func (e postgresError) Error() string    { return "postgres error " + e.state }
func (e postgresError) SQLState() string { return e.state }
