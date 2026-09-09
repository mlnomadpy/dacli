// Package tenant defines the transport- and persistence-independent tenant domain.
package tenant

import (
	"errors"
	"fmt"
	"regexp"
)

const maxIDLength = 128

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type AccountID string
type OrganizationID string
type TeamID string
type DeviceID string
type ProjectID string
type EnvironmentID string
type InvitationID string
type AssignmentID string

type Scope struct{ Organization OrganizationID }

// VerifiedIdentity is the closed result of an authentication adapter. Route,
// body, and queued payload data must never be used to reconstruct its scope.
type VerifiedIdentity struct {
	Scope             Scope
	Account           AccountID
	Device            DeviceID
	MembershipVersion Version
	PolicyRevision    uint64
}

func NewAccountID(value string) (AccountID, error) { return typedID[AccountID]("account", value) }
func NewOrganizationID(value string) (OrganizationID, error) {
	return typedID[OrganizationID]("organization", value)
}
func NewTeamID(value string) (TeamID, error)       { return typedID[TeamID]("team", value) }
func NewDeviceID(value string) (DeviceID, error)   { return typedID[DeviceID]("device", value) }
func NewProjectID(value string) (ProjectID, error) { return typedID[ProjectID]("project", value) }
func NewEnvironmentID(value string) (EnvironmentID, error) {
	return typedID[EnvironmentID]("environment", value)
}
func NewInvitationID(value string) (InvitationID, error) {
	return typedID[InvitationID]("invitation", value)
}
func NewAssignmentID(value string) (AssignmentID, error) {
	return typedID[AssignmentID]("assignment", value)
}

func typedID[T ~string](kind, value string) (T, error) {
	if len(value) == 0 || len(value) > maxIDLength || !idPattern.MatchString(value) {
		return "", fmt.Errorf("%s id must contain 1..128 opaque ASCII identifier characters", kind)
	}
	return T(value), nil
}

func NewScope(organization OrganizationID) (Scope, error) {
	if !validID(string(organization)) {
		return Scope{}, errors.New("tenant scope requires a valid organization id")
	}
	return Scope{Organization: organization}, nil
}

func ValidateVerifiedIdentity(value VerifiedIdentity) error {
	if _, err := NewScope(value.Scope.Organization); err != nil {
		return errors.New("verified identity has invalid tenant scope")
	}
	if !validID(string(value.Account)) || (value.Device != "" && !validID(string(value.Device))) || value.MembershipVersion == 0 || value.PolicyRevision == 0 {
		return errors.New("verified identity is incomplete")
	}
	return nil
}

func validID(value string) bool { return len(value) <= maxIDLength && idPattern.MatchString(value) }
