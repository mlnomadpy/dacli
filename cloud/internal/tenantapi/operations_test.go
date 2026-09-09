package tenantapi

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
	"github.com/mlnomadpy/dacli/cloud/internal/tenantcache"
	"github.com/mlnomadpy/dacli/cloud/internal/tenantrepo"
	"github.com/mlnomadpy/dacli/cloud/internal/tenantresource"
)

func operationsFixture(t *testing.T) (*Operations, sqlmock.Sqlmock, tenant.VerifiedIdentity, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(MAX(version), 0) FROM controlplane_schema_migrations`)).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(tenantrepo.SchemaVersion))
	repository, err := tenantrepo.Open(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	resources, err := tenantresource.New(repository, repository, func() time.Time { return time.Unix(2_000, 0) })
	if err != nil {
		t.Fatal(err)
	}
	operations, err := New(repository, resources, tenantcache.New[tenant.Project](8), 3)
	if err != nil {
		t.Fatal(err)
	}
	scope, _ := tenant.NewScope("tenant-a")
	identity := tenant.VerifiedIdentity{Scope: scope, Account: "account-a", Device: "device-a", MembershipVersion: 7, PolicyRevision: 3}
	return operations, mock, identity, func() { _ = db.Close() }
}

func expectBinding(mock sqlmock.Sqlmock, identity tenant.VerifiedIdentity) {
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`SELECT set_config('dacli.tenant_id', $1, true)`)).WithArgs(identity.Scope.Organization).WillReturnResult(sqlmock.NewResult(0, 1))
}

func expectMembership(mock sqlmock.Sqlmock, identity tenant.VerifiedIdentity, state tenant.Lifecycle) {
	expectBinding(mock, identity)
	mock.ExpectQuery(`FROM controlplane_memberships WHERE tenant_id = \$1 AND account_id = \$2`).WithArgs(identity.Scope.Organization, identity.Account).WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "account_id", "team_id", "roles", "state", "version", "expires_unix"}).AddRow(identity.Scope.Organization, identity.Account, "", []byte(`["manager"]`), state, identity.MembershipVersion, 0))
	mock.ExpectCommit()
}

func expectAttempt(mock sqlmock.Sqlmock, identity tenant.VerifiedIdentity, correlation string, action tenant.AuditAction, kind tenant.TargetKind, target string, before, after tenant.Version, result, reason string) {
	expectBinding(mock, identity)
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WithArgs(identity.Scope.Organization, correlation, identity.Account, identity.Device, action, kind, target, before, after, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), result, reason, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
}

func TestProjectCacheStillReloadsAuthorizationAndCannotServeRevokedMembership(t *testing.T) {
	operations, mock, identity, closeDB := operationsFixture(t)
	defer closeDB()
	expectMembership(mock, identity, tenant.LifecycleActive)
	expectBinding(mock, identity)
	mock.ExpectQuery(`FROM controlplane_projects WHERE tenant_id = \$1 AND project_id = \$2`).WithArgs(identity.Scope.Organization, tenant.ProjectID("project-a")).WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "project_id", "name", "state", "version"}).AddRow("tenant-a", "project-a", "Project", 1, 4))
	mock.ExpectCommit()
	first, err := operations.Project(context.Background(), identity, "project-a", 4)
	if err != nil || first.ID != "project-a" {
		t.Fatalf("first=%+v error=%v", first, err)
	}

	expectMembership(mock, identity, tenant.LifecycleActive)
	second, err := operations.Project(context.Background(), identity, "project-a", 4)
	if err != nil || second != first {
		t.Fatalf("cached=%+v error=%v", second, err)
	}

	revoked := identity
	revoked.MembershipVersion++
	expectMembership(mock, revoked, tenant.LifecycleRevoked)
	if _, err := operations.Project(context.Background(), revoked, "project-a", 4); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("revoked cache read = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProjectListAuthorizesBeforeTenantScopedPagination(t *testing.T) {
	operations, mock, identity, closeDB := operationsFixture(t)
	defer closeDB()
	expectMembership(mock, identity, tenant.LifecycleActive)
	expectBinding(mock, identity)
	mock.ExpectQuery(`MAX\(list_ordinal\).*controlplane_projects`).WithArgs(identity.Scope.Organization).WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(3))
	mock.ExpectQuery(`FROM controlplane_projects.*WHERE tenant_id = \$1`).WithArgs(identity.Scope.Organization, int64(3), int64(0), 2).WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "project_id", "name", "state", "version", "list_ordinal"}).AddRow("tenant-a", "project-a", "Project", 1, 1, 3))
	mock.ExpectCommit()
	page, err := operations.ListProjects(context.Background(), identity, tenantrepo.PageRequest{Limit: 1})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("page=%+v error=%v", page, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProjectUpdatePropagatesOptimisticConflictAfterAuthorization(t *testing.T) {
	operations, mock, identity, closeDB := operationsFixture(t)
	defer closeDB()
	expectMembership(mock, identity, tenant.LifecycleActive)
	expectBinding(mock, identity)
	mock.ExpectQuery(`FROM controlplane_projects WHERE tenant_id = \$1 AND project_id = \$2`).WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "project_id", "name", "state", "version"}).AddRow("tenant-a", "project-a", "Before", 1, 2))
	mock.ExpectExec(`UPDATE controlplane_projects`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()
	expectAttempt(mock, identity, "request-a", tenant.AuditActionUpdate, tenant.TargetProject, "project-a", 2, 3, "conflict", "version_conflict")
	value := tenant.Project{Tenant: identity.Scope.Organization, ID: "project-a", Name: "After", State: tenant.LifecycleActive, Version: 3}
	change := tenant.Mutation{Actor: identity.Account, Device: identity.Device, CorrelationID: "request-a"}
	if err := operations.UpdateProject(context.Background(), identity, change, 2, value); !errors.Is(err, ErrConflict) {
		t.Fatalf("update conflict = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProjectUpdateReloadsAuthorizationAndCommitsAuditEndToEnd(t *testing.T) {
	operations, mock, identity, closeDB := operationsFixture(t)
	defer closeDB()
	expectMembership(mock, identity, tenant.LifecycleActive)
	expectBinding(mock, identity)
	mock.ExpectQuery(`FROM controlplane_projects WHERE tenant_id = \$1 AND project_id = \$2`).WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "project_id", "name", "state", "version"}).AddRow("tenant-a", "project-a", "Before", 1, 2))
	mock.ExpectExec(`UPDATE controlplane_projects`).WithArgs(identity.Scope.Organization, tenant.ProjectID("project-a"), "After", tenant.LifecycleActive, tenant.Version(3), tenant.Version(2)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	value := tenant.Project{Tenant: identity.Scope.Organization, ID: "project-a", Name: "After", State: tenant.LifecycleActive, Version: 3}
	change := tenant.Mutation{Actor: identity.Account, Device: identity.Device, CorrelationID: "request-success"}
	if err := operations.UpdateProject(context.Background(), identity, change, 2, value); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthenticatedInvalidAndMissingMutationsRecordClosedRefusals(t *testing.T) {
	operations, mock, identity, closeDB := operationsFixture(t)
	defer closeDB()
	operations.now = func() time.Time { return time.UnixMilli(4444) }

	invalid := tenant.Environment{Tenant: identity.Scope.Organization, Project: "project-a", ID: "environment-a", State: tenant.LifecycleUnknown, Version: 1}
	expectAttempt(mock, identity, "request-invalid", tenant.AuditActionCreate, tenant.TargetEnvironment, "environment-a", 0, 1, "refused", "invalid_state")
	if err := operations.CreateEnvironment(context.Background(), identity, tenant.Mutation{Actor: identity.Account, Device: identity.Device, CorrelationID: "request-invalid"}, invalid); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("invalid state = %v", err)
	}

	expectMembership(mock, identity, tenant.LifecycleActive)
	expectBinding(mock, identity)
	mock.ExpectQuery(`FROM controlplane_projects WHERE tenant_id = \$1 AND project_id = \$2`).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
	missing := tenant.Project{Tenant: identity.Scope.Organization, ID: "same-id", Name: "Missing", State: tenant.LifecycleActive, Version: 2}
	expectAttempt(mock, identity, "request-missing", tenant.AuditActionUpdate, tenant.TargetProject, "same-id", 1, 2, "refused", "resource_unavailable")
	if err := operations.UpdateProject(context.Background(), identity, tenant.Mutation{Actor: identity.Account, Device: identity.Device, CorrelationID: "request-missing"}, 1, missing); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("missing resource = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPersistenceFailureIsAuditedAndAuditFailurePreventsEvidenceClaim(t *testing.T) {
	operations, mock, identity, closeDB := operationsFixture(t)
	defer closeDB()
	project := tenant.Project{Tenant: identity.Scope.Organization, ID: "project-a", Name: "Project", State: tenant.LifecycleActive, Version: 1}
	change := tenant.Mutation{Actor: identity.Account, Device: identity.Device, CorrelationID: "request-storage"}

	expectMembership(mock, identity, tenant.LifecycleActive)
	expectBinding(mock, identity)
	mock.ExpectExec(`INSERT INTO controlplane_projects`).WillReturnError(errors.New("storage offline"))
	mock.ExpectRollback()
	expectAttempt(mock, identity, change.CorrelationID, tenant.AuditActionCreate, tenant.TargetProject, "project-a", 0, 1, "failed", "persistence_failed")
	if err := operations.CreateProject(context.Background(), identity, change, project); err == nil || !regexp.MustCompile(`insert project`).MatchString(err.Error()) {
		t.Fatalf("persistence failure = %v", err)
	}

	change.CorrelationID = "request-audit-offline"
	expectMembership(mock, identity, tenant.LifecycleRevoked)
	expectBinding(mock, identity)
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnError(errors.New("audit offline"))
	mock.ExpectRollback()
	if err := operations.CreateProject(context.Background(), identity, change, project); err == nil || !regexp.MustCompile(`record authenticated mutation audit`).MatchString(err.Error()) {
		t.Fatalf("audit failure = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOperationsRejectsMissingDependenciesAndSpoofedActor(t *testing.T) {
	if _, err := New(nil, nil, nil, 0); err == nil {
		t.Fatal("missing dependencies accepted")
	}
	operations, mock, identity, closeDB := operationsFixture(t)
	defer closeDB()
	value := tenant.Project{Tenant: identity.Scope.Organization, ID: "project-a", Name: "After", State: tenant.LifecycleActive, Version: 2}
	expectAttempt(mock, identity, "request-a", tenant.AuditActionUpdate, tenant.TargetProject, "project-a", 1, 2, "refused", "authorization_denied")
	if err := operations.UpdateProject(context.Background(), identity, tenant.Mutation{Actor: "account-b", CorrelationID: "request-a"}, 1, value); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("spoofed actor = %v", err)
	}
	stalePolicy := identity
	stalePolicy.PolicyRevision--
	if _, err := operations.Project(context.Background(), stalePolicy, "project-a", 1); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("stale policy = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProjectAndEnvironmentLifecycleFlowsThroughAuthorizedRepository(t *testing.T) {
	operations, mock, identity, closeDB := operationsFixture(t)
	defer closeDB()
	change := tenant.Mutation{Actor: identity.Account, Device: identity.Device, CorrelationID: "request-create"}
	project := tenant.Project{Tenant: identity.Scope.Organization, ID: "project-a", Name: "Project", State: tenant.LifecycleActive, Version: 1}
	expectMembership(mock, identity, tenant.LifecycleActive)
	expectBinding(mock, identity)
	mock.ExpectExec(`INSERT INTO controlplane_projects`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := operations.CreateProject(context.Background(), identity, change, project); err != nil {
		t.Fatal(err)
	}

	environment := tenant.Environment{Tenant: identity.Scope.Organization, Project: project.ID, ID: "environment-a", Name: "Production", Kind: tenant.EnvironmentProduction, State: tenant.LifecycleActive, Version: 1}
	expectMembership(mock, identity, tenant.LifecycleActive)
	expectBinding(mock, identity)
	mock.ExpectExec(`INSERT INTO controlplane_environments`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	change.CorrelationID = "request-environment"
	if err := operations.CreateEnvironment(context.Background(), identity, change, environment); err != nil {
		t.Fatal(err)
	}

	expectMembership(mock, identity, tenant.LifecycleActive)
	expectBinding(mock, identity)
	mock.ExpectQuery(`MAX\(list_ordinal\).*controlplane_environments`).WithArgs(identity.Scope.Organization, project.ID).WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(4))
	mock.ExpectQuery(`FROM controlplane_environments.*WHERE tenant_id = \$1`).WithArgs(identity.Scope.Organization, project.ID, int64(4), int64(0), 2).WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "project_id", "environment_id", "name", "kind", "state", "version", "list_ordinal"}).AddRow("tenant-a", "project-a", "environment-a", "Production", 3, 1, 1, 4))
	mock.ExpectCommit()
	page, err := operations.ListEnvironments(context.Background(), identity, project.ID, tenantrepo.PageRequest{Limit: 1})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("environment page=%+v error=%v", page, err)
	}

	updated := environment
	updated.Name, updated.State, updated.Version = "Archived", tenant.LifecycleArchived, 2
	expectMembership(mock, identity, tenant.LifecycleActive)
	expectBinding(mock, identity)
	mock.ExpectQuery(`FROM controlplane_environments WHERE tenant_id = \$1 AND environment_id = \$2`).WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "project_id", "environment_id", "name", "kind", "state", "version"}).AddRow("tenant-a", "project-a", "environment-a", "Production", 3, 1, 1))
	mock.ExpectExec(`UPDATE controlplane_environments`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	change.CorrelationID = "request-environment-update"
	if err := operations.UpdateEnvironment(context.Background(), identity, change, 1, updated); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
