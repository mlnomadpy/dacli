package tenant

import (
	"errors"
	"fmt"
	"sort"
)

type Lifecycle uint8

const (
	LifecycleUnknown Lifecycle = iota
	LifecycleActive
	LifecycleSuspended
	LifecycleArchived
	LifecycleRevoked
)

type Version uint64

type Account struct {
	ID      AccountID
	Name    string
	State   Lifecycle
	Version Version
}

type Organization struct {
	ID      OrganizationID
	Name    string
	State   Lifecycle
	Version Version
}

type Team struct {
	Tenant  OrganizationID
	ID      TeamID
	Name    string
	State   Lifecycle
	Version Version
}

type Membership struct {
	Tenant      OrganizationID
	Account     AccountID
	Team        TeamID
	Roles       RoleSet
	State       Lifecycle
	Version     Version
	ExpiresUnix int64
}

type Device struct {
	Tenant    OrganizationID
	ID        DeviceID
	Account   AccountID
	Name      string
	Platform  Platform
	PublicKey [32]byte
	State     Lifecycle
	Version   Version
}

type Platform uint8

const (
	PlatformUnknown Platform = iota
	PlatformDarwin
	PlatformLinux
	PlatformWindows
)

type Project struct {
	Tenant  OrganizationID
	ID      ProjectID
	Name    string
	State   Lifecycle
	Version Version
}

type Environment struct {
	Tenant  OrganizationID
	Project ProjectID
	ID      EnvironmentID
	Name    string
	Kind    EnvironmentKind
	State   Lifecycle
	Version Version
}

type EnvironmentKind uint8

const (
	EnvironmentUnknown EnvironmentKind = iota
	EnvironmentDevelopment
	EnvironmentStaging
	EnvironmentProduction
)

func ValidateAccount(value Account) error {
	return validateRoot("account", string(value.ID), value.Name, value.State, value.Version)
}

func ValidateOrganization(value Organization) error {
	return validateRoot("organization", string(value.ID), value.Name, value.State, value.Version)
}

func ValidateTeam(scope Scope, value Team) error {
	if err := validateScoped(scope, value.Tenant, "team", string(value.ID), value.Name, value.State, value.Version); err != nil {
		return err
	}
	return nil
}

func ValidateMembership(scope Scope, value Membership) error {
	if err := validateTenant(scope, value.Tenant); err != nil {
		return err
	}
	if !validID(string(value.Account)) {
		return errors.New("membership requires a valid account id")
	}
	if value.Team != "" && !validID(string(value.Team)) {
		return errors.New("membership team id is invalid")
	}
	if !value.Roles.Valid() || value.Roles.Empty() {
		return errors.New("membership requires only known roles and at least one role")
	}
	if value.ExpiresUnix < 0 {
		return errors.New("membership expiry cannot be negative")
	}
	return validateStateVersion("membership", value.State, value.Version)
}

func ValidateDevice(scope Scope, value Device) error {
	if err := validateScoped(scope, value.Tenant, "device", string(value.ID), value.Name, value.State, value.Version); err != nil {
		return err
	}
	if !validID(string(value.Account)) {
		return errors.New("device requires a valid account id")
	}
	if value.Platform < PlatformDarwin || value.Platform > PlatformWindows {
		return errors.New("device platform is unknown")
	}
	if value.PublicKey == [32]byte{} {
		return errors.New("device public key is empty")
	}
	return nil
}

func ValidateProject(scope Scope, value Project) error {
	return validateScoped(scope, value.Tenant, "project", string(value.ID), value.Name, value.State, value.Version)
}

func ValidateEnvironment(scope Scope, value Environment) error {
	if err := validateScoped(scope, value.Tenant, "environment", string(value.ID), value.Name, value.State, value.Version); err != nil {
		return err
	}
	if !validID(string(value.Project)) {
		return errors.New("environment requires a valid project id")
	}
	if value.Kind < EnvironmentDevelopment || value.Kind > EnvironmentProduction {
		return errors.New("environment kind is unknown")
	}
	return nil
}

func validateScoped(scope Scope, tenant OrganizationID, kind, id, name string, state Lifecycle, version Version) error {
	if err := validateTenant(scope, tenant); err != nil {
		return err
	}
	return validateRoot(kind, id, name, state, version)
}

func validateTenant(scope Scope, tenant OrganizationID) error {
	if !validID(string(scope.Organization)) {
		return errors.New("operation requires an explicit valid tenant scope")
	}
	if tenant != scope.Organization {
		return errors.New("resource is outside the tenant scope")
	}
	return nil
}

func validateRoot(kind, id, name string, state Lifecycle, version Version) error {
	if !validID(id) {
		return fmt.Errorf("%s id is invalid", kind)
	}
	if len(name) == 0 || len(name) > 128 {
		return fmt.Errorf("%s name must contain 1..128 bytes", kind)
	}
	return validateStateVersion(kind, state, version)
}

func validateStateVersion(kind string, state Lifecycle, version Version) error {
	if state < LifecycleActive || state > LifecycleRevoked {
		return fmt.Errorf("%s lifecycle is unknown", kind)
	}
	if version == 0 {
		return fmt.Errorf("%s version must be positive", kind)
	}
	return nil
}

// StableRoles returns the role set in deterministic order for serialization.
func (m Membership) StableRoles() []Role {
	roles := m.Roles.Roles()
	sort.Slice(roles, func(i, j int) bool { return roles[i] < roles[j] })
	return roles
}
