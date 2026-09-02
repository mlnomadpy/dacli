package tenant

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Role uint8

const (
	RoleUnknown Role = iota
	RoleOwner
	RoleAdministrator
	RoleManager
	RoleDeveloper
	RoleReviewer
	RoleBilling
	RoleAuditor
)

var allRoles = [...]Role{RoleOwner, RoleAdministrator, RoleManager, RoleDeveloper, RoleReviewer, RoleBilling, RoleAuditor}

var roleNames = map[Role]string{
	RoleOwner: "owner", RoleAdministrator: "administrator", RoleManager: "manager",
	RoleDeveloper: "developer", RoleReviewer: "reviewer", RoleBilling: "billing", RoleAuditor: "auditor",
}

type RoleSet uint16

func Roles(values ...Role) (RoleSet, error) {
	var set RoleSet
	for _, role := range values {
		if role < RoleOwner || role > RoleAuditor {
			return 0, fmt.Errorf("unknown role %d", role)
		}
		set |= 1 << role
	}
	return set, nil
}

func (s RoleSet) Has(role Role) bool {
	return role >= RoleOwner && role <= RoleAuditor && s&(1<<role) != 0
}
func (s RoleSet) Empty() bool { return s == 0 }
func (s RoleSet) Valid() bool { return s&^RoleSet((1<<(RoleAuditor+1))-2) == 0 }
func (s RoleSet) Roles() []Role {
	var out []Role
	for _, role := range allRoles {
		if s.Has(role) {
			out = append(out, role)
		}
	}
	return out
}

type Permission uint8

const (
	PermissionUnknown Permission = iota
	PermissionOrganizationRead
	PermissionOrganizationManage
	PermissionTeamManage
	PermissionMembershipManage
	PermissionDeviceUse
	PermissionDeviceManage
	PermissionProjectRead
	PermissionProjectWrite
	PermissionProjectManage
	PermissionEnvironmentManage
	PermissionAuditRead
	PermissionBillingManage
)

var allPermissions = [...]Permission{
	PermissionOrganizationRead, PermissionOrganizationManage, PermissionTeamManage,
	PermissionMembershipManage, PermissionDeviceUse, PermissionDeviceManage,
	PermissionProjectRead, PermissionProjectWrite, PermissionProjectManage,
	PermissionEnvironmentManage, PermissionAuditRead, PermissionBillingManage,
}

var permissionNames = map[Permission]string{
	PermissionOrganizationRead: "organization.read", PermissionOrganizationManage: "organization.manage",
	PermissionTeamManage: "team.manage", PermissionMembershipManage: "membership.manage",
	PermissionDeviceUse: "device.use", PermissionDeviceManage: "device.manage",
	PermissionProjectRead: "project.read", PermissionProjectWrite: "project.write",
	PermissionProjectManage: "project.manage", PermissionEnvironmentManage: "environment.manage",
	PermissionAuditRead: "audit.read", PermissionBillingManage: "billing.manage",
}

var grants = map[Role]map[Permission]struct{}{
	RoleOwner: allPermissionSet(),
	RoleAdministrator: permissionSet(
		PermissionOrganizationRead, PermissionOrganizationManage, PermissionTeamManage,
		PermissionMembershipManage, PermissionDeviceUse, PermissionDeviceManage,
		PermissionProjectRead, PermissionProjectWrite, PermissionProjectManage,
		PermissionEnvironmentManage, PermissionAuditRead,
	),
	RoleManager: permissionSet(
		PermissionOrganizationRead, PermissionTeamManage, PermissionMembershipManage,
		PermissionDeviceUse, PermissionProjectRead, PermissionProjectWrite,
		PermissionProjectManage, PermissionEnvironmentManage,
	),
	RoleDeveloper: permissionSet(
		PermissionOrganizationRead, PermissionDeviceUse, PermissionProjectRead, PermissionProjectWrite,
	),
	RoleReviewer: permissionSet(
		PermissionOrganizationRead, PermissionDeviceUse, PermissionProjectRead,
	),
	RoleBilling: permissionSet(
		PermissionOrganizationRead, PermissionBillingManage,
	),
	RoleAuditor: permissionSet(
		PermissionOrganizationRead, PermissionProjectRead, PermissionAuditRead,
	),
}

func permissionSet(values ...Permission) map[Permission]struct{} {
	out := make(map[Permission]struct{}, len(values))
	for _, permission := range values {
		out[permission] = struct{}{}
	}
	return out
}

func allPermissionSet() map[Permission]struct{} {
	out := make(map[Permission]struct{}, len(allPermissions))
	for _, permission := range allPermissions {
		out[permission] = struct{}{}
	}
	return out
}

type MembershipSource interface {
	CurrentMembership(context.Context, Scope, AccountID) (Membership, error)
}

type Authorizer struct {
	memberships MembershipSource
	now         func() time.Time
}

type Authorization struct {
	Scope                     Scope
	Account                   AccountID
	Permission                Permission
	ExpectedMembershipVersion Version
}

var ErrDenied = errors.New("tenant authorization denied")

func NewAuthorizer(source MembershipSource, now func() time.Time) (*Authorizer, error) {
	if source == nil {
		return nil, errors.New("authorization requires a current membership source")
	}
	if now == nil {
		now = time.Now
	}
	return &Authorizer{memberships: source, now: now}, nil
}

func (a *Authorizer) Authorize(ctx context.Context, request Authorization) (Membership, error) {
	if !validID(string(request.Scope.Organization)) || !validID(string(request.Account)) || request.Permission < PermissionOrganizationRead || request.Permission > PermissionBillingManage {
		return Membership{}, ErrDenied
	}
	membership, err := a.memberships.CurrentMembership(ctx, request.Scope, request.Account)
	if err != nil {
		return Membership{}, ErrDenied
	}
	if err := ValidateMembership(request.Scope, membership); err != nil || membership.Account != request.Account || membership.State != LifecycleActive {
		return Membership{}, ErrDenied
	}
	if request.ExpectedMembershipVersion == 0 || membership.Version != request.ExpectedMembershipVersion {
		return Membership{}, ErrDenied
	}
	if membership.ExpiresUnix != 0 && membership.ExpiresUnix <= a.now().Unix() {
		return Membership{}, ErrDenied
	}
	for _, role := range membership.Roles.Roles() {
		if _, allowed := grants[role][request.Permission]; allowed {
			return membership, nil
		}
	}
	return Membership{}, ErrDenied
}

func Allows(role Role, permission Permission) bool {
	_, allowed := grants[role][permission]
	return allowed
}

func (r Role) String() string       { return roleNames[r] }
func (p Permission) String() string { return permissionNames[p] }

func ParseRole(value string) (Role, error) {
	for role, name := range roleNames {
		if name == value {
			return role, nil
		}
	}
	return RoleUnknown, fmt.Errorf("unknown tenant role %q", value)
}

func ParsePermission(value string) (Permission, error) {
	for permission, name := range permissionNames {
		if name == value {
			return permission, nil
		}
	}
	return PermissionUnknown, fmt.Errorf("unknown tenant permission %q", value)
}

// KnownRoles and KnownPermissions return copies so callers cannot mutate the policy registry.
func KnownRoles() []Role { return append([]Role(nil), allRoles[:]...) }

func KnownPermissions() []Permission { return append([]Permission(nil), allPermissions[:]...) }
