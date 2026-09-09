package tenantrepo

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
	"github.com/mlnomadpy/dacli/cloud/internal/tenantresource"
)

var _ tenantresource.Store = (*Repository)(nil)

func invitationFixture(t *testing.T) (tenant.Scope, Mutation, tenant.Invitation) {
	t.Helper()
	scope := mustScope(t, "tenant-a")
	roles, _ := tenant.Roles(tenant.RoleDeveloper)
	return scope, mutation("corr-invitation"), tenant.Invitation{
		Tenant: scope.Organization, ID: "invitation-1", Account: "account-1", Roles: roles,
		TokenDigest: sha256.Sum256([]byte(t.Name())), State: tenant.InvitationPending, Version: 1, ExpiresUnix: 2_000,
	}
}

func invitationRows(value tenant.Invitation) *sqlmock.Rows {
	roles, _ := encodeRoles(value.Roles)
	return sqlmock.NewRows([]string{"tenant_id", "invitation_id", "account_id", "team_id", "roles", "token_digest", "state", "version", "expires_unix"}).
		AddRow(value.Tenant, value.ID, value.Account, value.Team, roles, value.TokenDigest[:], value.State, value.Version, value.ExpiresUnix)
}

func TestIssueInvitationCommitsDigestAndAuditAtomically(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope, change, value := invitationFixture(t)
	roles, _ := encodeRoles(value.Roles)
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectExec(`INSERT INTO controlplane_invitations`).WithArgs(value.Tenant, value.ID, value.Account, value.Team, roles, value.TokenDigest[:], value.State, value.Version, value.ExpiresUnix).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.IssueInvitation(context.Background(), scope, change, value); err != nil {
		t.Fatal(err)
	}
	assertExpectations(t, mock)
}

func TestAcceptInvitationConsumesOnceAndCreatesMembershipInOneTransaction(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope, change, value := invitationFixture(t)
	change.OccurredAt = time.Unix(1_000, 0)
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_invitations WHERE tenant_id = \$1 AND token_digest = \$2 FOR UPDATE`).WithArgs(scope.Organization, value.TokenDigest[:]).WillReturnRows(invitationRows(value))
	mock.ExpectExec(`UPDATE controlplane_invitations SET state = \$3, version = \$4`).WithArgs(scope.Organization, value.ID, tenant.InvitationAccepted, tenant.Version(2), tenant.InvitationPending, tenant.Version(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_memberships`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WithArgs(scope.Organization, change.CorrelationID+".invitation", sqlmock.AnyArg(), sqlmock.AnyArg(), tenant.AuditActionAssign, tenant.TargetInvitation, string(value.ID), tenant.Version(1), tenant.Version(2), sqlmock.AnyArg(), sqlmock.AnyArg(), change.OccurredAt.UnixMilli()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WithArgs(scope.Organization, change.CorrelationID+".membership", sqlmock.AnyArg(), sqlmock.AnyArg(), tenant.AuditActionCreate, tenant.TargetMembership, string(value.Account), tenant.Version(0), tenant.Version(1), sqlmock.AnyArg(), sqlmock.AnyArg(), change.OccurredAt.UnixMilli()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	membership, err := repo.AcceptInvitation(context.Background(), scope, change, value.TokenDigest, value.Account, 1)
	if err != nil {
		t.Fatal(err)
	}
	if membership.Account != value.Account || membership.Version != 1 || membership.State != tenant.LifecycleActive {
		t.Fatalf("membership = %+v", membership)
	}
	assertExpectations(t, mock)
}

func TestAcceptInvitationExpiryWrongTenantAndReplayFailClosed(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*tenant.Invitation, *Mutation)
	}{
		{"expired", func(value *tenant.Invitation, change *Mutation) { change.OccurredAt = time.Unix(value.ExpiresUnix, 0) }},
		{"wrong account", func(value *tenant.Invitation, _ *Mutation) { value.Account = "different-account" }},
		{"replay", func(value *tenant.Invitation, _ *Mutation) { value.State, value.Version = tenant.InvitationAccepted, 2 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, mock, closeDB := testRepository(t)
			defer closeDB()
			scope, change, value := invitationFixture(t)
			change.OccurredAt = time.Unix(1_000, 0)
			test.change(&value, &change)
			mock.ExpectBegin()
			expectTenantBinding(mock, scope)
			mock.ExpectQuery(`FROM controlplane_invitations`).WillReturnRows(invitationRows(value))
			mock.ExpectRollback()
			_, err := repo.AcceptInvitation(context.Background(), scope, change, value.TokenDigest, "account-1", 1)
			if !errors.Is(err, ErrConflict) {
				t.Fatalf("accept = %v", err)
			}
			assertExpectations(t, mock)
		})
	}
}

func TestInvitationCancelAndExpireAreOptimisticAuditedTransitions(t *testing.T) {
	for _, state := range []tenant.InvitationState{tenant.InvitationCancelled, tenant.InvitationExpired} {
		t.Run(string(rune(state+'0')), func(t *testing.T) {
			repo, mock, closeDB := testRepository(t)
			defer closeDB()
			scope, change, value := invitationFixture(t)
			if state == tenant.InvitationExpired {
				change.OccurredAt = time.Unix(value.ExpiresUnix, 0)
			}
			mock.ExpectBegin()
			expectTenantBinding(mock, scope)
			mock.ExpectQuery(`FROM controlplane_invitations WHERE tenant_id = \$1 AND invitation_id = \$2`).WithArgs(scope.Organization, value.ID).WillReturnRows(invitationRows(value))
			mock.ExpectExec(`UPDATE controlplane_invitations`).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
			if err := repo.TransitionInvitation(context.Background(), scope, change, value.ID, 1, state); err != nil {
				t.Fatal(err)
			}
			assertExpectations(t, mock)
		})
	}
}

func TestInvitationCannotExpireBeforeItsDeadline(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope, change, value := invitationFixture(t)
	change.OccurredAt = time.Unix(value.ExpiresUnix-1, 0)
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_invitations`).WillReturnRows(invitationRows(value))
	mock.ExpectRollback()
	if err := repo.TransitionInvitation(context.Background(), scope, change, value.ID, 1, tenant.InvitationExpired); !errors.Is(err, ErrConflict) {
		t.Fatalf("early expiry = %v", err)
	}
	assertExpectations(t, mock)
}

func TestEnvironmentLifecycleIsTenantScopedOptimisticAndAudited(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	value := tenant.Environment{Tenant: scope.Organization, Project: "project-1", ID: "environment-1", Name: "Production", Kind: tenant.EnvironmentProduction, State: tenant.LifecycleActive, Version: 1}
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectExec(`INSERT INTO controlplane_environments`).WithArgs(value.Tenant, value.Project, value.ID, value.Name, value.Kind, value.State, value.Version).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.CreateEnvironment(context.Background(), scope, mutation("env-create"), value); err != nil {
		t.Fatal(err)
	}

	archived := value
	archived.State, archived.Version = tenant.LifecycleArchived, 2
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_environments WHERE tenant_id = \$1 AND environment_id = \$2`).WithArgs(scope.Organization, value.ID).WillReturnRows(environmentRows(value))
	mock.ExpectExec(`UPDATE controlplane_environments`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WithArgs(scope.Organization, "env-archive", sqlmock.AnyArg(), sqlmock.AnyArg(), tenant.AuditActionArchive, tenant.TargetEnvironment, string(value.ID), tenant.Version(1), tenant.Version(2), sqlmock.AnyArg(), sqlmock.AnyArg(), int64(1234)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.UpdateEnvironment(context.Background(), scope, mutation("env-archive"), 1, archived); err != nil {
		t.Fatal(err)
	}
	assertExpectations(t, mock)
}

func TestEnvironmentIdentifierCollisionDoesNotCrossTenant(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-b")
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_environments WHERE tenant_id = \$1 AND environment_id = \$2`).WithArgs(scope.Organization, tenant.EnvironmentID("same-id")).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
	if _, err := repo.Environment(context.Background(), scope, "same-id"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant environment = %v", err)
	}
	assertExpectations(t, mock)
}

func TestAssignmentsUseTenantRelationshipsAndOptimisticArchive(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	project := tenant.ProjectAssignment{Tenant: scope.Organization, ID: "project-assignment-1", Project: "project-1", Account: "account-1", State: tenant.LifecycleActive, Version: 1}
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectExec(`INSERT INTO controlplane_project_assignments`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.CreateProjectAssignment(context.Background(), scope, mutation("assign-project"), project); err != nil {
		t.Fatal(err)
	}

	archived := project
	archived.State, archived.Version = tenant.LifecycleArchived, 2
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_project_assignments WHERE tenant_id = \$1 AND assignment_id = \$2`).WithArgs(scope.Organization, project.ID).WillReturnRows(projectAssignmentRows(project))
	mock.ExpectExec(`UPDATE controlplane_project_assignments`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.UpdateProjectAssignment(context.Background(), scope, mutation("archive-project-assignment"), 1, archived); err != nil {
		t.Fatal(err)
	}

	environment := tenant.EnvironmentAssignment{Tenant: scope.Organization, ID: "environment-assignment-1", Project: "project-1", Environment: "environment-1", Account: "account-1", State: tenant.LifecycleActive, Version: 1}
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectExec(`INSERT INTO controlplane_environment_assignments`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.CreateEnvironmentAssignment(context.Background(), scope, mutation("assign-environment"), environment); err != nil {
		t.Fatal(err)
	}
	assertExpectations(t, mock)
}

func TestAssignmentConcurrentUpdateRollsBackWithoutAudit(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	value := tenant.EnvironmentAssignment{Tenant: scope.Organization, ID: "assignment-1", Project: "project-1", Environment: "environment-1", Account: "account-1", State: tenant.LifecycleArchived, Version: 2}
	current := value
	current.State, current.Version = tenant.LifecycleActive, 1
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_environment_assignments`).WillReturnRows(environmentAssignmentRows(current))
	mock.ExpectExec(`UPDATE controlplane_environment_assignments`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()
	if err := repo.UpdateEnvironmentAssignment(context.Background(), scope, mutation("assignment-race"), 1, value); !errors.Is(err, ErrConflict) {
		t.Fatalf("lost update = %v", err)
	}
	assertExpectations(t, mock)
}

func TestEnvironmentAssignmentUpdateCommitsAuditAndRestore(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	before := tenant.EnvironmentAssignment{Tenant: scope.Organization, ID: "assignment-1", Project: "project-1", Environment: "environment-1", Account: "account-1", State: tenant.LifecycleArchived, Version: 2}
	after := before
	after.State, after.Version = tenant.LifecycleActive, 3
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_environment_assignments`).WillReturnRows(environmentAssignmentRows(before))
	mock.ExpectExec(`UPDATE controlplane_environment_assignments`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WithArgs(scope.Organization, "assignment-restore", sqlmock.AnyArg(), sqlmock.AnyArg(), tenant.AuditActionRestore, tenant.TargetEnvironmentAssignment, string(after.ID), tenant.Version(2), tenant.Version(3), sqlmock.AnyArg(), sqlmock.AnyArg(), int64(1234)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.UpdateEnvironmentAssignment(context.Background(), scope, mutation("assignment-restore"), 2, after); err != nil {
		t.Fatal(err)
	}
	assertExpectations(t, mock)
}

func TestWorkflowValidationRejectsUnknownTransitionsBeforeTransaction(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope, change, invitation := invitationFixture(t)
	if err := repo.TransitionInvitation(context.Background(), scope, change, invitation.ID, 1, tenant.InvitationAccepted); err == nil {
		t.Fatal("direct accepted transition was allowed")
	}
	badEnvironment := tenant.Environment{Tenant: scope.Organization, ID: "environment-1", Name: "Environment", State: tenant.LifecycleActive, Version: 1}
	if err := repo.CreateEnvironment(context.Background(), scope, change, badEnvironment); err == nil {
		t.Fatal("environment without a project was accepted")
	}
	badAssignment := tenant.ProjectAssignment{Tenant: scope.Organization, ID: "assignment-1", Project: "project-1", State: tenant.LifecycleActive, Version: 1}
	if err := repo.CreateProjectAssignment(context.Background(), scope, change, badAssignment); err == nil {
		t.Fatal("assignment without an account was accepted")
	}
	assertExpectations(t, mock)
}

func environmentRows(value tenant.Environment) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"tenant_id", "project_id", "environment_id", "name", "kind", "state", "version"}).AddRow(value.Tenant, value.Project, value.ID, value.Name, value.Kind, value.State, value.Version)
}

func projectAssignmentRows(value tenant.ProjectAssignment) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"tenant_id", "assignment_id", "project_id", "account_id", "state", "version"}).AddRow(value.Tenant, value.ID, value.Project, value.Account, value.State, value.Version)
}

func environmentAssignmentRows(value tenant.EnvironmentAssignment) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"tenant_id", "assignment_id", "project_id", "environment_id", "account_id", "state", "version"}).AddRow(value.Tenant, value.ID, value.Project, value.Environment, value.Account, value.State, value.Version)
}
