package tenant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
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
	if event.ActionDigest == [32]byte{} || event.Result != AuditResultSucceeded || event.ResultReason() != "committed" {
		t.Fatalf("audit event lost action result: %+v", event)
	}
	assertNoMutableReferences(t, reflect.TypeOf(event))
	if _, err := NewAuditEvent(scope, account, device, AuditActionUpdate, TargetProject, string(project), 9, 9, before, after, 1); err == nil {
		t.Fatal("non-advancing audit version was accepted")
	}
}

func TestActionDigestIsStableAndBindsEveryClosedMutationField(t *testing.T) {
	scope, _, _, _, project, _ := mustIDs(t)
	before := sha256.Sum256([]byte("before"))
	after := sha256.Sum256([]byte("after"))
	base := NewActionDigest(scope, AuditActionUpdate, TargetProject, string(project), 8, 9, before, after)
	if got, want := hex.EncodeToString(base[:]), "25305457b9bd18f796078bfc763987208fe2278c85825a052b5f2a344310ce01"; got != want {
		t.Fatalf("action digest fixture=%s", got)
	}
	otherScope, _ := NewScope("tenant-other")
	mutations := [][32]byte{
		NewActionDigest(otherScope, AuditActionUpdate, TargetProject, string(project), 8, 9, before, after),
		NewActionDigest(scope, AuditActionArchive, TargetProject, string(project), 8, 9, before, after),
		NewActionDigest(scope, AuditActionUpdate, TargetEnvironment, string(project), 8, 9, before, after),
		NewActionDigest(scope, AuditActionUpdate, TargetProject, "other", 8, 9, before, after),
		NewActionDigest(scope, AuditActionUpdate, TargetProject, string(project), 7, 9, before, after),
		NewActionDigest(scope, AuditActionUpdate, TargetProject, string(project), 8, 10, before, after),
		NewActionDigest(scope, AuditActionUpdate, TargetProject, string(project), 8, 9, sha256.Sum256([]byte("changed")), after),
		NewActionDigest(scope, AuditActionUpdate, TargetProject, string(project), 8, 9, before, sha256.Sum256([]byte("changed"))),
	}
	for index, changed := range mutations {
		if changed == base {
			t.Errorf("field mutation %d did not change action digest", index)
		}
	}
}

func TestActionDigestSeparatesEveryMutationFamily(t *testing.T) {
	scope, _, _, _, _, _ := mustIDs(t)
	state := sha256.Sum256([]byte("closed-state"))
	families := []struct {
		action AuditAction
		target TargetKind
	}{
		{AuditActionCreate, TargetInvitation},
		{AuditActionAssign, TargetMembership},
		{AuditActionCreate, TargetProject},
		{AuditActionCreate, TargetEnvironment},
		{AuditActionAssign, TargetProjectAssignment},
		{AuditActionAssign, TargetEnvironmentAssignment},
		{AuditActionCreate, TargetSession},
		{AuditActionRevoke, TargetSession},
	}
	seen := make(map[[32]byte]bool, len(families))
	for _, family := range families {
		digest := NewActionDigest(scope, family.action, family.target, "same-target", 1, 2, state, state)
		if digest == [32]byte{} || seen[digest] {
			t.Fatalf("mutation family action=%d target=%d has empty or colliding digest", family.action, family.target)
		}
		seen[digest] = true
	}
	for result := AuditResultSucceeded; result <= AuditResultFailed; result++ {
		if result.String() == "" {
			t.Fatalf("audit result %d has no closed name", result)
		}
	}
	if AuditResultUnknown.String() != "" {
		t.Fatal("unknown audit result acquired a persisted name")
	}
	for reason := AuditReasonCommitted; reason <= AuditReasonPersistenceFailed; reason++ {
		if reason.String() == "" {
			t.Fatalf("audit reason %d has no closed name", reason)
		}
	}
	if AuditReasonUnknown.String() != "" {
		t.Fatal("unknown audit reason acquired a persisted name")
	}
}

func TestAuthenticatedAttemptsAreImmutableAndResultReasonBound(t *testing.T) {
	scope, account, _, device, _, _ := mustIDs(t)
	before := sha256.Sum256([]byte("expected-version"))
	after := sha256.Sum256([]byte("requested-state"))
	families := []struct {
		action AuditAction
		kind   TargetKind
	}{
		{AuditActionCreate, TargetInvitation},
		{AuditActionAssign, TargetMembership},
		{AuditActionCreate, TargetProject},
		{AuditActionCreate, TargetEnvironment},
		{AuditActionAssign, TargetProjectAssignment},
		{AuditActionAssign, TargetEnvironmentAssignment},
		{AuditActionCreate, TargetSession},
		{AuditActionRevoke, TargetSession},
	}
	for _, family := range families {
		event, err := NewAuditAttempt(scope, account, device, family.action, family.kind, "opaque-target", 4, 5, before, after, AuditResultRefused, AuditReasonAuthorizationDenied, 1234)
		if err != nil {
			t.Fatalf("action=%d target=%d: %v", family.action, family.kind, err)
		}
		if event.Result != AuditResultRefused || event.ResultReason() != "authorization_denied" || event.ActionDigest == [32]byte{} {
			t.Fatalf("attempt lost closed evidence: %+v", event)
		}
		assertNoMutableReferences(t, reflect.TypeOf(event))
	}
	if _, err := NewAuditAttempt(scope, account, device, AuditActionUpdate, TargetProject, "project-a", 4, 5, before, after, AuditResultSucceeded, AuditReasonCommitted, 1234); err == nil {
		t.Fatal("success was accepted by the failure-only constructor")
	}
	if _, err := NewAuditAttempt(scope, account, device, AuditActionUpdate, TargetProject, "project-a", 4, 5, before, after, AuditResultConflict, AuditReasonAuthorizationDenied, 1234); err == nil {
		t.Fatal("mismatched result and reason were accepted")
	}
	event, _ := NewAuditAttempt(scope, account, device, AuditActionUpdate, TargetProject, "project-a", 4, 5, before, after, AuditResultConflict, AuditReasonVersionConflict, 1234)
	event.ActionDigest[0]++
	if err := ValidateAuditEvent(event); err == nil {
		t.Fatal("changed action digest was accepted by persistence validation")
	}
	event.TargetIDLength = len(event.TargetID) + 1
	if err := ValidateAuditEvent(event); err == nil {
		t.Fatal("unsafe target length was accepted")
	}
	invalidTarget := "unsafe target text"
	redacted, err := NewAuditAttempt(scope, account, device, AuditActionUpdate, TargetProject, invalidTarget, 4, 5, before, after, AuditResultRefused, AuditReasonInvalidState, 1234)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(redacted.Target(), invalidTarget) || !strings.HasPrefix(redacted.Target(), "invalid-") {
		t.Fatalf("invalid target was not reduced to an opaque identity: %q", redacted.Target())
	}
}

func TestInvitationAndAssignmentsAreClosedTenantScopedValues(t *testing.T) {
	scope, account, _, _, project, environment := mustIDs(t)
	roles, _ := Roles(RoleDeveloper)
	invitationID, _ := NewInvitationID("invitation_01")
	assignmentID, _ := NewAssignmentID("assignment_01")
	digest := sha256.Sum256([]byte("one-time invitation credential"))
	values := []struct {
		name     string
		validate func(Scope) error
	}{
		{"invitation", func(s Scope) error {
			return ValidateInvitation(s, Invitation{Tenant: scope.Organization, ID: invitationID, Account: account, Roles: roles, TokenDigest: digest, State: InvitationPending, Version: 1, ExpiresUnix: 100})
		}},
		{"project assignment", func(s Scope) error {
			return ValidateProjectAssignment(s, ProjectAssignment{Tenant: scope.Organization, ID: assignmentID, Project: project, Account: account, State: LifecycleActive, Version: 1})
		}},
		{"environment assignment", func(s Scope) error {
			return ValidateEnvironmentAssignment(s, EnvironmentAssignment{Tenant: scope.Organization, ID: assignmentID, Project: project, Environment: environment, Account: account, State: LifecycleActive, Version: 1})
		}},
	}
	otherOrganization, _ := NewOrganizationID("org_other")
	otherScope, _ := NewScope(otherOrganization)
	for _, value := range values {
		if err := value.validate(scope); err != nil {
			t.Errorf("%s valid = %v", value.name, err)
		}
		if err := value.validate(otherScope); err == nil {
			t.Errorf("%s accepted a colliding identifier from another tenant", value.name)
		}
	}
}

func TestAssignmentAuditMayStartAtVersionOne(t *testing.T) {
	scope, account, _, device, _, _ := mustIDs(t)
	after := sha256.Sum256([]byte("assignment"))
	if _, err := NewAuditEvent(scope, account, device, AuditActionAssign, TargetProjectAssignment, "assignment-1", 0, 1, [32]byte{}, after, 1); err != nil {
		t.Fatalf("new assignment audit = %v", err)
	}
}

func TestVerifiedIdentityIsClosedAndComplete(t *testing.T) {
	scope, account, _, device, _, _ := mustIDs(t)
	base := VerifiedIdentity{Scope: scope, Account: account, Device: device, MembershipVersion: 2, PolicyRevision: 3}
	if err := ValidateVerifiedIdentity(base); err != nil {
		t.Fatal(err)
	}
	invalid := []VerifiedIdentity{base, base, base, base}
	invalid[0].Scope = Scope{}
	invalid[1].Account = ""
	invalid[2].MembershipVersion = 0
	invalid[3].PolicyRevision = 0
	for _, value := range invalid {
		if err := ValidateVerifiedIdentity(value); err == nil {
			t.Fatalf("incomplete identity accepted: %+v", value)
		}
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
