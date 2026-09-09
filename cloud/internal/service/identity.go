package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

var ErrIdentityDenied = errors.New("verified identity required")

type IdentityVerifier interface {
	Verify(context.Context, string) (tenant.VerifiedIdentity, error)
}

type HMACIdentityVerifier struct {
	secret []byte
	now    func() time.Time
}

type identityClaims struct {
	Tenant            string `json:"tenant_id"`
	Account           string `json:"account_id"`
	Device            string `json:"device_id,omitempty"`
	MembershipVersion uint64 `json:"membership_version"`
	PolicyRevision    uint64 `json:"policy_revision"`
	ExpiresUnix       int64  `json:"expires_unix"`
}

func NewHMACIdentityVerifier(secret string) (*HMACIdentityVerifier, error) {
	return newHMACIdentityVerifier(secret, time.Now)
}

func newHMACIdentityVerifier(secret string, now func() time.Time) (*HMACIdentityVerifier, error) {
	if len(secret) < 32 || now == nil {
		return nil, errors.New("identity verifier requires at least 32 secret bytes and a clock")
	}
	return &HMACIdentityVerifier{secret: []byte(secret), now: now}, nil
}

func (v *HMACIdentityVerifier) Verify(_ context.Context, authorization string) (tenant.VerifiedIdentity, error) {
	if v == nil || !strings.HasPrefix(authorization, "Bearer ") {
		return tenant.VerifiedIdentity{}, ErrIdentityDenied
	}
	parts := strings.Split(strings.TrimPrefix(authorization, "Bearer "), ".")
	if len(parts) != 2 || len(parts[0]) > 2048 {
		return tenant.VerifiedIdentity{}, ErrIdentityDenied
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return tenant.VerifiedIdentity{}, ErrIdentityDenied
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, sign(v.secret, "identity", payload)) {
		return tenant.VerifiedIdentity{}, ErrIdentityDenied
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var claims identityClaims
	if err := decoder.Decode(&claims); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return tenant.VerifiedIdentity{}, ErrIdentityDenied
	}
	organization, err := tenant.NewOrganizationID(claims.Tenant)
	if err != nil {
		return tenant.VerifiedIdentity{}, ErrIdentityDenied
	}
	scope, _ := tenant.NewScope(organization)
	identity := tenant.VerifiedIdentity{Scope: scope, Account: tenant.AccountID(claims.Account), Device: tenant.DeviceID(claims.Device), MembershipVersion: tenant.Version(claims.MembershipVersion), PolicyRevision: claims.PolicyRevision}
	if err := tenant.ValidateVerifiedIdentity(identity); err != nil {
		return tenant.VerifiedIdentity{}, ErrIdentityDenied
	}
	if claims.ExpiresUnix <= v.now().Unix() {
		return tenant.VerifiedIdentity{}, ErrIdentityDenied
	}
	return identity, nil
}

// Sign creates the internal bearer representation used until the native
// device-login flow (#984) supplies verified identities through its adapter.
func (v *HMACIdentityVerifier) Sign(identity tenant.VerifiedIdentity) (string, error) {
	if v == nil || tenant.ValidateVerifiedIdentity(identity) != nil {
		return "", ErrIdentityDenied
	}
	payload, err := json.Marshal(identityClaims{Tenant: string(identity.Scope.Organization), Account: string(identity.Account), Device: string(identity.Device), MembershipVersion: uint64(identity.MembershipVersion), PolicyRevision: identity.PolicyRevision, ExpiresUnix: v.now().Add(5 * time.Minute).Unix()})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sign(v.secret, "identity", payload)), nil
}

func sign(secret []byte, domain string, payload []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(domain))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}
