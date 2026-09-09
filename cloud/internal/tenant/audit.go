package tenant

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"
)

// Mutation carries the immutable actor and occurrence identity persisted with
// an audit event. Authorization inputs stay in the service layer.
type Mutation struct {
	Actor         AccountID
	Device        DeviceID
	CorrelationID string
	OccurredAt    time.Time
}

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
	TargetSession
	TargetInvitation
	TargetProjectAssignment
	TargetEnvironmentAssignment
)

type AuditResult uint8

const (
	AuditResultUnknown AuditResult = iota
	AuditResultSucceeded
	AuditResultRefused
	AuditResultConflict
	AuditResultFailed
)

func (r AuditResult) String() string {
	switch r {
	case AuditResultSucceeded:
		return "succeeded"
	case AuditResultRefused:
		return "refused"
	case AuditResultConflict:
		return "conflict"
	case AuditResultFailed:
		return "failed"
	default:
		return ""
	}
}

type AuditReason uint8

const (
	AuditReasonUnknown AuditReason = iota
	AuditReasonCommitted
	AuditReasonAuthorizationDenied
	AuditReasonInvalidState
	AuditReasonResourceUnavailable
	AuditReasonVersionConflict
	AuditReasonPersistenceFailed
)

func (r AuditReason) String() string {
	switch r {
	case AuditReasonCommitted:
		return "committed"
	case AuditReasonAuthorizationDenied:
		return "authorization_denied"
	case AuditReasonInvalidState:
		return "invalid_state"
	case AuditReasonResourceUnavailable:
		return "resource_unavailable"
	case AuditReasonVersionConflict:
		return "version_conflict"
	case AuditReasonPersistenceFailed:
		return "persistence_failed"
	default:
		return ""
	}
}

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
	ActionDigest      [32]byte
	Result            AuditResult
	Reason            [64]byte
	ReasonLength      int
	OccurredUnixMilli int64
}

func NewAuditEvent(scope Scope, actor AccountID, device DeviceID, action AuditAction, kind TargetKind, targetID string, before, after Version, beforeDigest, afterDigest [32]byte, occurredUnixMilli int64) (AuditEvent, error) {
	if !validID(string(scope.Organization)) || !validID(string(actor)) || (device != "" && !validID(string(device))) {
		return AuditEvent{}, errors.New("audit identity is invalid")
	}
	if action < AuditActionCreate || action > AuditActionRemove || kind < TargetOrganization || kind > TargetEnvironmentAssignment || !validID(targetID) {
		return AuditEvent{}, errors.New("audit action or target is invalid")
	}
	if after == 0 || after <= before || occurredUnixMilli <= 0 {
		return AuditEvent{}, errors.New("audit version and timestamp must advance")
	}
	if afterDigest == [32]byte{} {
		return AuditEvent{}, errors.New("audit after digest is required")
	}
	if action == AuditActionCreate || (action == AuditActionAssign && before == 0) {
		if before != 0 || beforeDigest != [32]byte{} {
			return AuditEvent{}, errors.New("create audit must have an empty before state")
		}
	} else if before == 0 || beforeDigest == [32]byte{} {
		return AuditEvent{}, errors.New("non-create audit requires a bound before state")
	}
	var stableTarget [128]byte
	copy(stableTarget[:], targetID)
	actionDigest := NewActionDigest(scope, action, kind, targetID, before, after, beforeDigest, afterDigest)
	return newAuditRecord(scope, actor, device, action, kind, stableTarget, len(targetID), before, after, beforeDigest, afterDigest, actionDigest, AuditResultSucceeded, AuditReasonCommitted, occurredUnixMilli)
}

// NewAuditAttempt records an authenticated mutation that did not commit. Its
// inputs are closed, already-digested action fields, so request bodies and
// credentials cannot be represented in durable audit evidence.
func NewAuditAttempt(scope Scope, actor AccountID, device DeviceID, action AuditAction, kind TargetKind, targetID string, before, after Version, beforeDigest, afterDigest [32]byte, result AuditResult, reason AuditReason, occurredUnixMilli int64) (AuditEvent, error) {
	if result < AuditResultRefused || result > AuditResultFailed || !validAttemptReason(result, reason) {
		return AuditEvent{}, errors.New("audit attempt result is invalid")
	}
	if !validID(string(scope.Organization)) || !validID(string(actor)) || (device != "" && !validID(string(device))) {
		return AuditEvent{}, errors.New("audit identity is invalid")
	}
	if action < AuditActionCreate || action > AuditActionRemove || kind < TargetOrganization || kind > TargetEnvironmentAssignment {
		return AuditEvent{}, errors.New("audit action or target is invalid")
	}
	if !validID(targetID) {
		digest := sha256.Sum256([]byte(targetID))
		targetID = "invalid-" + hex.EncodeToString(digest[:])
	}
	if after == 0 || after <= before || occurredUnixMilli <= 0 || afterDigest == [32]byte{} {
		return AuditEvent{}, errors.New("audit attempt action identity is invalid")
	}
	var stableTarget [128]byte
	copy(stableTarget[:], targetID)
	actionDigest := NewActionDigest(scope, action, kind, targetID, before, after, beforeDigest, afterDigest)
	return newAuditRecord(scope, actor, device, action, kind, stableTarget, len(targetID), before, after, beforeDigest, afterDigest, actionDigest, result, reason, occurredUnixMilli)
}

func newAuditRecord(scope Scope, actor AccountID, device DeviceID, action AuditAction, kind TargetKind, target [128]byte, targetLength int, before, after Version, beforeDigest, afterDigest, actionDigest [32]byte, result AuditResult, reason AuditReason, occurredUnixMilli int64) (AuditEvent, error) {
	reasonName := reason.String()
	if reasonName == "" {
		return AuditEvent{}, errors.New("audit reason is invalid")
	}
	var stableReason [64]byte
	copy(stableReason[:], reasonName)
	return AuditEvent{
		Tenant: scope.Organization, Actor: actor, ActorDevice: device,
		Action: action, TargetKind: kind, TargetID: target, TargetIDLength: targetLength,
		VersionBefore: before, VersionAfter: after, BeforeDigest: beforeDigest,
		AfterDigest: afterDigest, ActionDigest: actionDigest, Result: result,
		Reason: stableReason, ReasonLength: len(reasonName), OccurredUnixMilli: occurredUnixMilli,
	}, nil
}

func validAttemptReason(result AuditResult, reason AuditReason) bool {
	switch result {
	case AuditResultRefused:
		return reason == AuditReasonAuthorizationDenied || reason == AuditReasonInvalidState || reason == AuditReasonResourceUnavailable
	case AuditResultConflict:
		return reason == AuditReasonVersionConflict
	case AuditResultFailed:
		return reason == AuditReasonPersistenceFailed
	default:
		return false
	}
}

func (e AuditEvent) Target() string       { return string(e.TargetID[:e.TargetIDLength]) }
func (e AuditEvent) ResultReason() string { return string(e.Reason[:e.ReasonLength]) }

// ValidateAuditEvent protects provider-neutral persistence adapters from
// forged or partially initialized values. Constructors remain the normal path.
func ValidateAuditEvent(event AuditEvent) error {
	if event.TargetIDLength < 1 || event.TargetIDLength > len(event.TargetID) || event.ReasonLength < 1 || event.ReasonLength > len(event.Reason) {
		return errors.New("audit event contains invalid bounded lengths")
	}
	target, reason := event.Target(), event.ResultReason()
	if !validID(string(event.Tenant)) || !validID(string(event.Actor)) || (event.ActorDevice != "" && !validID(string(event.ActorDevice))) || !validID(target) {
		return errors.New("audit event identity is invalid")
	}
	if event.Action < AuditActionCreate || event.Action > AuditActionRemove || event.TargetKind < TargetOrganization || event.TargetKind > TargetEnvironmentAssignment || event.VersionAfter == 0 || event.VersionAfter <= event.VersionBefore || event.AfterDigest == [32]byte{} || event.OccurredUnixMilli <= 0 {
		return errors.New("audit event action identity is invalid")
	}
	validResultReason := event.Result == AuditResultSucceeded && reason == AuditReasonCommitted.String()
	if event.Result != AuditResultSucceeded {
		for candidate := AuditReasonAuthorizationDenied; candidate <= AuditReasonPersistenceFailed; candidate++ {
			if candidate.String() == reason && validAttemptReason(event.Result, candidate) {
				validResultReason = true
				break
			}
		}
	}
	if !validResultReason {
		return errors.New("audit event result and reason are invalid")
	}
	want := NewActionDigest(Scope{Organization: event.Tenant}, event.Action, event.TargetKind, target, event.VersionBefore, event.VersionAfter, event.BeforeDigest, event.AfterDigest)
	if event.ActionDigest != want {
		return errors.New("audit event action digest does not match its closed fields")
	}
	return nil
}

// NewActionDigest canonicalizes only closed mutation fields. Credentials and
// arbitrary metadata cannot enter this API; credential-bearing domain values
// are represented only by their already-required one-way state digests.
func NewActionDigest(scope Scope, action AuditAction, kind TargetKind, targetID string, before, after Version, beforeDigest, afterDigest [32]byte) [32]byte {
	hash := sha256.New()
	writeDigestPart(hash, []byte("dacli-control-plane-action/v1"))
	writeDigestPart(hash, []byte(scope.Organization))
	writeDigestPart(hash, []byte{byte(action), byte(kind)})
	writeDigestPart(hash, []byte(targetID))
	var versions [16]byte
	binary.BigEndian.PutUint64(versions[:8], uint64(before))
	binary.BigEndian.PutUint64(versions[8:], uint64(after))
	writeDigestPart(hash, versions[:])
	writeDigestPart(hash, beforeDigest[:])
	writeDigestPart(hash, afterDigest[:])
	var out [32]byte
	copy(out[:], hash.Sum(nil))
	return out
}

type digestWriter interface{ Write([]byte) (int, error) }

func writeDigestPart(hash digestWriter, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = hash.Write(length[:])
	_, _ = hash.Write(value)
}
