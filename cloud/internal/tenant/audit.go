package tenant

import "errors"

type AuditAction uint8

const (
	AuditActionUnknown AuditAction = iota
	AuditActionCreate
	AuditActionUpdate
	AuditActionArchive
	AuditActionRestore
	AuditActionRevoke
	AuditActionAssign
	AuditActionRemove
)

type TargetKind uint8

const (
	TargetUnknown TargetKind = iota
	TargetOrganization
	TargetTeam
	TargetMembership
	TargetDevice
	TargetProject
	TargetEnvironment
)

// AuditEvent is pointer-free: callers cannot mutate a shared digest or nested
// collection after persistence accepts the value.
type AuditEvent struct {
	Tenant            OrganizationID
	Actor             AccountID
	ActorDevice       DeviceID
	Action            AuditAction
	TargetKind        TargetKind
	TargetID          [128]byte
	TargetIDLength    int
	VersionBefore     Version
	VersionAfter      Version
	BeforeDigest      [32]byte
	AfterDigest       [32]byte
	OccurredUnixMilli int64
}

func NewAuditEvent(scope Scope, actor AccountID, device DeviceID, action AuditAction, kind TargetKind, targetID string, before, after Version, beforeDigest, afterDigest [32]byte, occurredUnixMilli int64) (AuditEvent, error) {
	if !validID(string(scope.Organization)) || !validID(string(actor)) || (device != "" && !validID(string(device))) {
		return AuditEvent{}, errors.New("audit identity is invalid")
	}
	if action < AuditActionCreate || action > AuditActionRemove || kind < TargetOrganization || kind > TargetEnvironment || !validID(targetID) {
		return AuditEvent{}, errors.New("audit action or target is invalid")
	}
	if after == 0 || after <= before || occurredUnixMilli <= 0 {
		return AuditEvent{}, errors.New("audit version and timestamp must advance")
	}
	if afterDigest == [32]byte{} {
		return AuditEvent{}, errors.New("audit after digest is required")
	}
	if action == AuditActionCreate {
		if before != 0 || beforeDigest != [32]byte{} {
			return AuditEvent{}, errors.New("create audit must have an empty before state")
		}
	} else if before == 0 || beforeDigest == [32]byte{} {
		return AuditEvent{}, errors.New("non-create audit requires a bound before state")
	}
	var stableTarget [128]byte
	copy(stableTarget[:], targetID)
	return AuditEvent{
		Tenant: scope.Organization, Actor: actor, ActorDevice: device,
		Action: action, TargetKind: kind, TargetID: stableTarget, TargetIDLength: len(targetID),
		VersionBefore: before, VersionAfter: after, BeforeDigest: beforeDigest,
		AfterDigest: afterDigest, OccurredUnixMilli: occurredUnixMilli,
	}, nil
}

func (e AuditEvent) Target() string { return string(e.TargetID[:e.TargetIDLength]) }
