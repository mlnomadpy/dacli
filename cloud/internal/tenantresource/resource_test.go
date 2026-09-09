package tenantresource

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

type fixtureStore struct {
	membership     tenant.Membership
	project        tenant.Project
	environment    tenant.Environment
	invitation     tenant.Invitation
	acceptedDigest [32]byte
	calls          []string
}

func (s *fixtureStore) CurrentMembership(context.Context, tenant.Scope, tenant.AccountID) (tenant.Membership, error) {
	return s.membership, nil
}
func (s *fixtureStore) IssueInvitation(_ context.Context, _ tenant.Scope, _ tenant.Mutation, value tenant.Invitation) error {
	s.calls = append(s.calls, "issue-invitation")
	s.invitation = value
	return nil
}
func (s *fixtureStore) AcceptInvitation(_ context.Context, _ tenant.Scope, _ tenant.Mutation, digest [32]byte, account tenant.AccountID, _ tenant.Version) (tenant.Membership, error) {
	s.calls = append(s.calls, "accept-invitation")
	s.acceptedDigest = digest
	return tenant.Membership{Tenant: s.membership.Tenant, Account: account, Roles: s.membership.Roles, State: tenant.LifecycleActive, Version: 1}, nil
}
func (s *fixtureStore) TransitionInvitation(_ context.Context, _ tenant.Scope, _ tenant.Mutation, _ tenant.InvitationID, _ tenant.Version, state tenant.InvitationState) error {
	s.calls = append(s.calls, "transition-invitation:"+string(rune(state+'0')))
	return nil
}
func (s *fixtureStore) Project(context.Context, tenant.Scope, tenant.ProjectID) (tenant.Project, error) {
	return s.project, nil
}
func (s *fixtureStore) CreateProject(context.Context, tenant.Scope, tenant.Mutation, tenant.Project) error {
	s.calls = append(s.calls, "create-project")
	return nil
}
func (s *fixtureStore) UpdateProject(_ context.Context, _ tenant.Scope, _ tenant.Mutation, _ tenant.Version, value tenant.Project) error {
	s.calls = append(s.calls, "update-project")
	s.project = value
	return nil
}
func (s *fixtureStore) Environment(context.Context, tenant.Scope, tenant.EnvironmentID) (tenant.Environment, error) {
	return s.environment, nil
}
func (s *fixtureStore) CreateEnvironment(context.Context, tenant.Scope, tenant.Mutation, tenant.Environment) error {
	s.calls = append(s.calls, "create-environment")
	return nil
}
func (s *fixtureStore) UpdateEnvironment(_ context.Context, _ tenant.Scope, _ tenant.Mutation, _ tenant.Version, value tenant.Environment) error {
	s.calls = append(s.calls, "update-environment")
	s.environment = value
	return nil
}
func (s *fixtureStore) CreateProjectAssignment(context.Context, tenant.Scope, tenant.Mutation, tenant.ProjectAssignment) error {
	s.calls = append(s.calls, "assign-project")
	return nil
}
func (s *fixtureStore) UpdateProjectAssignment(context.Context, tenant.Scope, tenant.Mutation, tenant.Version, tenant.ProjectAssignment) error {
	s.calls = append(s.calls, "update-project-assignment")
	return nil
}
func (s *fixtureStore) CreateEnvironmentAssignment(context.Context, tenant.Scope, tenant.Mutation, tenant.EnvironmentAssignment) error {
	s.calls = append(s.calls, "assign-environment")
	return nil
}
func (s *fixtureStore) UpdateEnvironmentAssignment(context.Context, tenant.Scope, tenant.Mutation, tenant.Version, tenant.EnvironmentAssignment) error {
	s.calls = append(s.calls, "update-environment-assignment")
	return nil
}

func resourceFixture(t *testing.T, role tenant.Role) (*Manager, *fixtureStore, tenant.Scope, Mutation) {
	t.Helper()
	organization, _ := tenant.NewOrganizationID("tenant-a")
	scope, _ := tenant.NewScope(organization)
	account, _ := tenant.NewAccountID("actor-a")
	roles, _ := tenant.Roles(role)
	store := &fixtureStore{membership: tenant.Membership{Tenant: organization, Account: account, Roles: roles, State: tenant.LifecycleActive, Version: 7}}
	manager, err := New(store, store, func() time.Time { return time.Unix(1_000, 0) })
	if err != nil {
		t.Fatal(err)
	}
	mutation := Mutation{Actor: account, ExpectedMembershipVersion: 7, CorrelationID: "corr-1", OccurredAt: time.Unix(1_000, 0)}
	return manager, store, scope, mutation
}

func TestInvitationCredentialIsHashedAndNeverSerialized(t *testing.T) {
	manager, store, scope, mutation := resourceFixture(t, tenant.RoleManager)
	invited, _ := tenant.NewAccountID("invitee-a")
	invitationID, _ := tenant.NewInvitationID("invite-1")
	roles, _ := tenant.Roles(tenant.RoleDeveloper)
	token := []byte("01234567890123456789012345678901")
	value := tenant.Invitation{Tenant: scope.Organization, ID: invitationID, Account: invited, Roles: roles, State: tenant.InvitationPending, Version: 1, ExpiresUnix: 2_000}
	if err := manager.IssueInvitation(context.Background(), scope, mutation, value, token); err != nil {
		t.Fatal(err)
	}
	if store.invitation.TokenDigest != sha256.Sum256(token) {
		t.Fatal("store did not receive the token digest")
	}
	encoded, err := json.Marshal(store.invitation)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), string(token)) || strings.Contains(string(encoded), "TokenDigest") || strings.Contains(string(encoded), "token_digest") {
		t.Fatalf("serialized invitation leaked credential material: %s", encoded)
	}
}

func TestInvitationAcceptanceBindsAccountAndOneTimeDigest(t *testing.T) {
	manager, store, scope, mutation := resourceFixture(t, tenant.RoleManager)
	account, _ := tenant.NewAccountID("invitee-a")
	mutation.Actor = account
	token := []byte("abcdefghijklmnopqrstuvwxyzABCDEF")
	membership, err := manager.AcceptInvitation(context.Background(), scope, mutation, account, token, 1)
	if err != nil {
		t.Fatal(err)
	}
	if membership.Account != account || store.acceptedDigest != sha256.Sum256(token) {
		t.Fatal("acceptance lost its bound identity or digest")
	}
	other, _ := tenant.NewAccountID("other-a")
	if _, err := manager.AcceptInvitation(context.Background(), scope, mutation, other, token, 1); !errors.Is(err, ErrDenied) {
		t.Fatalf("wrong account = %v", err)
	}
	if len(store.calls) != 1 {
		t.Fatalf("store calls after denied acceptance = %v", store.calls)
	}
}

func TestEveryPrivilegedOperationRequiresItsExactPermission(t *testing.T) {
	manager, store, scope, mutation := resourceFixture(t, tenant.RoleDeveloper)
	projectID, _ := tenant.NewProjectID("project-a")
	environmentID, _ := tenant.NewEnvironmentID("environment-a")
	assignmentID, _ := tenant.NewAssignmentID("assignment-a")
	account, _ := tenant.NewAccountID("member-a")
	invitationID, _ := tenant.NewInvitationID("invitation-a")
	roles, _ := tenant.Roles(tenant.RoleDeveloper)
	invitation := tenant.Invitation{Tenant: scope.Organization, ID: invitationID, Account: account, Roles: roles, State: tenant.InvitationPending, Version: 1, ExpiresUnix: 2_000}
	token := []byte("01234567890123456789012345678901")
	project := tenant.Project{Tenant: scope.Organization, ID: projectID, Name: "Project", State: tenant.LifecycleActive, Version: 1}
	environment := tenant.Environment{Tenant: scope.Organization, Project: projectID, ID: environmentID, Name: "Production", Kind: tenant.EnvironmentProduction, State: tenant.LifecycleActive, Version: 1}
	projectAssignment := tenant.ProjectAssignment{Tenant: scope.Organization, ID: assignmentID, Project: projectID, Account: account, State: tenant.LifecycleActive, Version: 1}
	environmentAssignment := tenant.EnvironmentAssignment{Tenant: scope.Organization, ID: assignmentID, Project: projectID, Environment: environmentID, Account: account, State: tenant.LifecycleActive, Version: 1}
	checks := []func() error{
		func() error { return manager.IssueInvitation(context.Background(), scope, mutation, invitation, token) },
		func() error { return manager.CancelInvitation(context.Background(), scope, mutation, invitationID, 1) },
		func() error { return manager.ExpireInvitation(context.Background(), scope, mutation, invitationID, 1) },
		func() error { return manager.CreateProject(context.Background(), scope, mutation, project) },
		func() error { return manager.CreateEnvironment(context.Background(), scope, mutation, environment) },
		func() error { return manager.AssignProject(context.Background(), scope, mutation, projectAssignment) },
		func() error {
			return manager.AssignEnvironment(context.Background(), scope, mutation, environmentAssignment)
		},
	}
	for _, check := range checks {
		if err := check(); !errors.Is(err, ErrDenied) {
			t.Fatalf("unauthorized operation = %v", err)
		}
	}
	if len(store.calls) != 0 {
		t.Fatalf("denied operations reached persistence: %v", store.calls)
	}
}

func TestMembershipRevocationAndStaleVersionTakeEffectBeforePersistence(t *testing.T) {
	manager, store, scope, mutation := resourceFixture(t, tenant.RoleManager)
	projectID, _ := tenant.NewProjectID("project-a")
	project := tenant.Project{Tenant: scope.Organization, ID: projectID, Name: "Project", State: tenant.LifecycleActive, Version: 1}
	store.membership.State = tenant.LifecycleRevoked
	store.membership.Version++
	mutation.ExpectedMembershipVersion++
	if err := manager.CreateProject(context.Background(), scope, mutation, project); !errors.Is(err, ErrDenied) {
		t.Fatalf("revoked membership = %v", err)
	}
	store.membership.State = tenant.LifecycleActive
	mutation.ExpectedMembershipVersion--
	if err := manager.CreateProject(context.Background(), scope, mutation, project); !errors.Is(err, ErrDenied) {
		t.Fatalf("stale membership = %v", err)
	}
	if len(store.calls) != 0 {
		t.Fatalf("denied operations reached persistence: %v", store.calls)
	}
}

func TestArchiveIsARecoverableVersionedTransition(t *testing.T) {
	manager, store, scope, mutation := resourceFixture(t, tenant.RoleManager)
	projectID, _ := tenant.NewProjectID("project-a")
	environmentID, _ := tenant.NewEnvironmentID("environment-a")
	store.project = tenant.Project{Tenant: scope.Organization, ID: projectID, Name: "Project", State: tenant.LifecycleActive, Version: 3}
	store.environment = tenant.Environment{Tenant: scope.Organization, Project: projectID, ID: environmentID, Name: "Prod", Kind: tenant.EnvironmentProduction, State: tenant.LifecycleActive, Version: 5}
	if err := manager.ArchiveProject(context.Background(), scope, mutation, projectID, 3); err != nil {
		t.Fatal(err)
	}
	if store.project.State != tenant.LifecycleArchived || store.project.Version != 4 {
		t.Fatalf("project archive = %+v", store.project)
	}
	if err := manager.ArchiveEnvironment(context.Background(), scope, mutation, environmentID, 5); err != nil {
		t.Fatal(err)
	}
	if store.environment.State != tenant.LifecycleArchived || store.environment.Version != 6 {
		t.Fatalf("environment archive = %+v", store.environment)
	}
	if err := manager.ArchiveProject(context.Background(), scope, mutation, projectID, 3); !errors.Is(err, ErrDenied) {
		t.Fatalf("archive replay = %v", err)
	}
}

func TestManagerRoutesEverySupportedWorkflowAfterAuthorization(t *testing.T) {
	manager, store, scope, mutation := resourceFixture(t, tenant.RoleManager)
	projectID, _ := tenant.NewProjectID("project-a")
	environmentID, _ := tenant.NewEnvironmentID("environment-a")
	projectAssignmentID, _ := tenant.NewAssignmentID("project-assignment-a")
	environmentAssignmentID, _ := tenant.NewAssignmentID("environment-assignment-a")
	account, _ := tenant.NewAccountID("member-a")
	project := tenant.Project{Tenant: scope.Organization, ID: projectID, Name: "Project", State: tenant.LifecycleActive, Version: 1}
	environment := tenant.Environment{Tenant: scope.Organization, Project: projectID, ID: environmentID, Name: "Production", Kind: tenant.EnvironmentProduction, State: tenant.LifecycleActive, Version: 1}
	projectAssignment := tenant.ProjectAssignment{Tenant: scope.Organization, ID: projectAssignmentID, Project: projectID, Account: account, State: tenant.LifecycleActive, Version: 1}
	environmentAssignment := tenant.EnvironmentAssignment{Tenant: scope.Organization, ID: environmentAssignmentID, Project: projectID, Environment: environmentID, Account: account, State: tenant.LifecycleActive, Version: 1}
	operations := []func() error{
		func() error { return manager.CreateProject(context.Background(), scope, mutation, project) },
		func() error {
			project.Version = 2
			project.Name = "Updated"
			return manager.UpdateProject(context.Background(), scope, mutation, 1, project)
		},
		func() error { return manager.CreateEnvironment(context.Background(), scope, mutation, environment) },
		func() error {
			environment.Version = 2
			environment.Name = "Updated"
			return manager.UpdateEnvironment(context.Background(), scope, mutation, 1, environment)
		},
		func() error { return manager.AssignProject(context.Background(), scope, mutation, projectAssignment) },
		func() error {
			projectAssignment.Version = 2
			projectAssignment.State = tenant.LifecycleArchived
			return manager.UpdateProjectAssignment(context.Background(), scope, mutation, 1, projectAssignment)
		},
		func() error {
			return manager.AssignEnvironment(context.Background(), scope, mutation, environmentAssignment)
		},
		func() error {
			environmentAssignment.Version = 2
			environmentAssignment.State = tenant.LifecycleArchived
			return manager.UpdateEnvironmentAssignment(context.Background(), scope, mutation, 1, environmentAssignment)
		},
		func() error {
			return manager.CancelInvitation(context.Background(), scope, mutation, "invitation-a", 1)
		},
		func() error {
			return manager.ExpireInvitation(context.Background(), scope, mutation, "invitation-b", 2)
		},
	}
	for index, operation := range operations {
		if err := operation(); err != nil {
			t.Fatalf("operation %d = %v", index, err)
		}
	}
	if len(store.calls) != len(operations) {
		t.Fatalf("store calls = %v", store.calls)
	}
}

func TestManagerRejectsInvalidConstructionAndInputs(t *testing.T) {
	if _, err := New(nil, nil, nil); err == nil {
		t.Fatal("nil store was accepted")
	}
	manager, store, scope, mutation := resourceFixture(t, tenant.RoleManager)
	project := tenant.Project{Tenant: scope.Organization, ID: "project-a", Name: "Project", State: tenant.LifecycleActive, Version: 2}
	if err := manager.CreateProject(context.Background(), scope, mutation, project); !errors.Is(err, ErrDenied) {
		t.Fatalf("version-two create = %v", err)
	}
	if err := manager.UpdateProject(context.Background(), scope, mutation, 0, project); !errors.Is(err, ErrDenied) {
		t.Fatalf("zero-version update = %v", err)
	}
	if err := manager.CancelInvitation(context.Background(), scope, mutation, "invitation-a", 0); !errors.Is(err, ErrDenied) {
		t.Fatalf("zero-version cancellation = %v", err)
	}
	if err := manager.IssueInvitation(context.Background(), scope, mutation, tenant.Invitation{}, []byte("short")); !errors.Is(err, ErrDenied) {
		t.Fatalf("short invitation credential = %v", err)
	}
	if len(store.calls) != 0 {
		t.Fatalf("invalid inputs reached persistence: %v", store.calls)
	}
}

func TestTenantResourceRecordsHaveClosedPrivacyFields(t *testing.T) {
	allowed := map[reflect.Type]map[string]bool{
		reflect.TypeOf(tenant.Invitation{}):            {"Tenant": true, "ID": true, "Account": true, "Team": true, "Roles": true, "TokenDigest": true, "State": true, "Version": true, "ExpiresUnix": true},
		reflect.TypeOf(tenant.Project{}):               {"Tenant": true, "ID": true, "Name": true, "State": true, "Version": true},
		reflect.TypeOf(tenant.Environment{}):           {"Tenant": true, "Project": true, "ID": true, "Name": true, "Kind": true, "State": true, "Version": true},
		reflect.TypeOf(tenant.ProjectAssignment{}):     {"Tenant": true, "ID": true, "Project": true, "Account": true, "State": true, "Version": true},
		reflect.TypeOf(tenant.EnvironmentAssignment{}): {"Tenant": true, "ID": true, "Project": true, "Environment": true, "Account": true, "State": true, "Version": true},
	}
	for typ, fields := range allowed {
		if typ.NumField() != len(fields) {
			t.Fatalf("%s fields changed without a privacy review", typ)
		}
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if !fields[field.Name] {
				t.Errorf("%s contains unreviewed field %s", typ, field.Name)
			}
			lower := strings.ToLower(field.Name)
			for _, forbidden := range []string{"source", "prompt", "transcript", "secret", "metadata", "path", "output"} {
				if strings.Contains(lower, forbidden) {
					t.Errorf("%s contains forbidden field %s", typ, field.Name)
				}
			}
		}
	}
}
