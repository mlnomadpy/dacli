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

type Scope struct{ Organization OrganizationID }

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

func validID(value string) bool { return len(value) <= maxIDLength && idPattern.MatchString(value) }
