package tenant

import (
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"testing"
	"time"
)

func mustIDs(t *testing.T) (Scope, AccountID, TeamID, DeviceID, ProjectID, EnvironmentID) {
	t.Helper()
	organization, err := NewOrganizationID("org_same-opaque-id")
	if err != nil {
		t.Fatal(err)
	}
	scope, err := NewScope(organization)
	if err != nil {
		t.Fatal(err)
	}
	account, _ := NewAccountID("acct_01")
	team, _ := NewTeamID("team_01")
	device, _ := NewDeviceID("device_01")
	project, _ := NewProjectID("project_collision")
	environment, _ := NewEnvironmentID("environment_01")
	return scope, account, team, device, project, environment
}

func TestTypedIdentifiersRejectUnboundedOrStructuredInput(t *testing.T) {
	for _, value := range []string{"", " leading", "path/to/source", "line\nbreak", string(make([]byte, 129))} {
		if _, err := NewProjectID(value); err == nil {
			t.Fatalf("project id %q was accepted", value)
		}
	}
	account, err := NewAccountID("same")
	if err != nil {
		t.Fatal(err)
	}
	project, err := NewProjectID("same")
	if err != nil {
		t.Fatal(err)
	}
	if reflect.TypeOf(account) == reflect.TypeOf(project) {
		t.Fatal("account and project identifiers are not distinct types")
	}
}

func TestEntityValidationFailsClosedAcrossTenantCollision(t *testing.T) {
	scope, account, teamID, deviceID, projectID, environmentID := mustIDs(t)
	otherOrganization, _ := NewOrganizationID(string(scope.Organization) + "-other")
	otherScope, _ := NewScope(otherOrganization)
	roles, _ := Roles(RoleDeveloper)
	publicKey := sha256.Sum256([]byte("public key fixture"))
	fixtures := []struct {
		name     string
		validate func(Scope) error
	}{
		{"team", func(s Scope) error {
			return ValidateTeam(s, Team{Tenant: scope.Organization, ID: teamID, Name: "Platform", State: LifecycleActive, Version: 1})
		}},
		{"membership", func(s Scope) error {
			return ValidateMembership(s, Membership{Tenant: scope.Organization, Account: account, Team: teamID, Roles: roles, State: LifecycleActive, Version: 1})
		}},
		{"device", func(s Scope) error {
			return ValidateDevice(s, Device{Tenant: scope.Organization, ID: deviceID, Account: account, Name: "Build host", Platform: PlatformLinux, PublicKey: publicKey, State: LifecycleActive, Version: 1})
		}},
		{"project", func(s Scope) error {
			return ValidateProject(s, Project{Tenant: scope.Organization, ID: projectID, Name: "Same ID in each tenant", State: LifecycleActive, Version: 1})
		}},
		{"environment", func(s Scope) error {
			return ValidateEnvironment(s, Environment{Tenant: scope.Organization, Project: projectID, ID: environmentID, Name: "Production", Kind: EnvironmentProduction, State: LifecycleActive, Version: 1})
		}},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			if err := fixture.validate(scope); err != nil {
				t.Fatalf("own tenant refused: %v", err)
			}
			if err := fixture.validate(otherScope); err == nil {
				t.Fatal("same identifier was accepted through another tenant scope")
			}
		})
	}
}

func TestRolePermissionMatrixIsExhaustiveAndDenyByDefault(t *testing.T) {
	expected := map[Role]map[Permission]bool{
		RoleOwner:         allow(allPermissions[:]...),
		RoleAdministrator: allow(PermissionOrganizationRead, PermissionOrganizationManage, PermissionTeamManage, PermissionMembershipManage, PermissionDeviceUse, PermissionDeviceManage, PermissionProjectRead, PermissionProjectWrite, PermissionProjectManage, PermissionEnvironmentManage, PermissionAuditRead),
		RoleManager:       allow(PermissionOrganizationRead, PermissionTeamManage, PermissionMembershipManage, PermissionDeviceUse, PermissionProjectRead, PermissionProjectWrite, PermissionProjectManage, PermissionEnvironmentManage),
		RoleDeveloper:     allow(PermissionOrganizationRead, PermissionDeviceUse, PermissionProjectRead, PermissionProjectWrite),
		RoleReviewer:      allow(PermissionOrganizationRead, PermissionDeviceUse, PermissionProjectRead),
		RoleBilling:       allow(PermissionOrganizationRead, PermissionBillingManage),
		RoleAuditor:       allow(PermissionOrganizationRead, PermissionProjectRead, PermissionAuditRead),
	}
	if len(expected) != len(allRoles) || len(grants) != len(allRoles) {
		t.Fatalf("role registry expected=%d implementation=%d all=%d", len(expected), len(grants), len(allRoles))
	}
	for _, role := range allRoles {
		if role.String() == "" {
			t.Errorf("role %d has no stable name", role)
		}
		if parsed, err := ParseRole(role.String()); err != nil || parsed != role {
			t.Errorf("role %d round trip = %d, %v", role, parsed, err)
		}
		for _, permission := range allPermissions {
			if permission.String() == "" {
				t.Errorf("permission %d has no stable name", permission)
			}
			if parsed, err := ParsePermission(permission.String()); err != nil || parsed != permission {
				t.Errorf("permission %d round trip = %d, %v", permission, parsed, err)
			}
			if got, want := Allows(role, permission), expected[role][permission]; got != want {
				t.Errorf("role=%d permission=%d allows=%v want=%v", role, permission, got, want)
			}
		}
	}
	if Allows(RoleUnknown, PermissionProjectRead) || Allows(RoleOwner, PermissionUnknown) || Allows(255, PermissionProjectRead) {
		t.Fatal("unknown role or permission was allowed")
	}
	if _, err := ParseRole("superuser"); err == nil {
		t.Fatal("unknown role name was parsed")
	}
	copyOfRoles := KnownRoles()
	copyOfRoles[0] = RoleUnknown
	if KnownRoles()[0] != RoleOwner {
		t.Fatal("caller mutated the role registry")
	}
}

func allow(values ...Permission) map[Permission]bool {
	out := make(map[Permission]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

type changingMembershipSource struct {
	memberships []Membership
	calls       int
	err         error
}

func (s *changingMembershipSource) CurrentMembership(context.Context, Scope, AccountID) (Membership, error) {
	s.calls++
	if s.err != nil {
		return Membership{}, s.err
	}
	index := s.calls - 1
	if index >= len(s.memberships) {
		index = len(s.memberships) - 1
	}
	return s.memberships[index], nil
}

func TestAuthorizerReloadsMembershipAndRevocationTakesEffectNextOperation(t *testing.T) {
	scope, account, team, _, _, _ := mustIDs(t)
	roles, _ := Roles(RoleDeveloper)
	active := Membership{Tenant: scope.Organization, Account: account, Team: team, Roles: roles, State: LifecycleActive, Version: 4}
	revoked := active
	revoked.State = LifecycleRevoked
	revoked.Version = 5
	source := &changingMembershipSource{memberships: []Membership{active, revoked}}
	authorizer, err := NewAuthorizer(source, func() time.Time { return time.Unix(2_000, 0) })
	if err != nil {
		t.Fatal(err)
	}
	request := Authorization{Scope: scope, Account: account, Permission: PermissionProjectWrite, ExpectedMembershipVersion: 4}
	if _, err := authorizer.Authorize(context.Background(), request); err != nil {
		t.Fatalf("active membership refused: %v", err)
	}
	request.ExpectedMembershipVersion = 5
	if _, err := authorizer.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
		t.Fatalf("revoked membership = %v, want denial", err)
	}
	if source.calls != 2 {
		t.Fatalf("membership reads = %d, want one per operation", source.calls)
	}
}

func TestAuthorizerRefusesWrongTenantExpiredStaleAndMissingMembership(t *testing.T) {
	scope, account, team, _, _, _ := mustIDs(t)
	roles, _ := Roles(RoleOwner)
	base := Membership{Tenant: scope.Organization, Account: account, Team: team, Roles: roles, State: LifecycleActive, Version: 7}
	otherOrganization, _ := NewOrganizationID("org_other")
	otherScope, _ := NewScope(otherOrganization)
	fixtures := []struct {
		name       string
		membership Membership
		scope      Scope
		expected   Version
		sourceErr  error
	}{
		{"wrong tenant", base, otherScope, 7, nil},
		{"expired", func() Membership { value := base; value.ExpiresUnix = 999; return value }(), scope, 7, nil},
		{"stale version", base, scope, 6, nil},
		{"missing version", base, scope, 0, nil},
		{"removed", base, scope, 7, errors.New("not found")},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			source := &changingMembershipSource{memberships: []Membership{fixture.membership}, err: fixture.sourceErr}
			authorizer, _ := NewAuthorizer(source, func() time.Time { return time.Unix(1_000, 0) })
			_, err := authorizer.Authorize(context.Background(), Authorization{Scope: fixture.scope, Account: account, Permission: PermissionOrganizationRead, ExpectedMembershipVersion: fixture.expected})
			if !errors.Is(err, ErrDenied) {
				t.Fatalf("authorization = %v, want generic denial", err)
			}
		})
	}
}

func TestAuditEventIsImmutablePointerFreeAndVersionBound(t *testing.T) {
	scope, account, _, device, project, _ := mustIDs(t)
	before := sha256.Sum256([]byte("before"))
	after := sha256.Sum256([]byte("after"))
	event, err := NewAuditEvent(scope, account, device, AuditActionUpdate, TargetProject, string(project), 8, 9, before, after, 1_700_000_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if event.Target() != string(project) || event.BeforeDigest != before || event.AfterDigest != after {
		t.Fatalf("audit event lost bound values: %+v", event)
	}
	assertNoMutableReferences(t, reflect.TypeOf(event))
	if _, err := NewAuditEvent(scope, account, device, AuditActionUpdate, TargetProject, string(project), 9, 9, before, after, 1); err == nil {
		t.Fatal("non-advancing audit version was accepted")
	}
}

func assertNoMutableReferences(t *testing.T, value reflect.Type) {
	t.Helper()
	for index := 0; index < value.NumField(); index++ {
		kind := value.Field(index).Type.Kind()
		if kind == reflect.Pointer || kind == reflect.Map || kind == reflect.Slice || kind == reflect.Interface || kind == reflect.Func {
			t.Fatalf("audit field %s contains mutable reference kind %s", value.Field(index).Name, kind)
		}
	}
}
