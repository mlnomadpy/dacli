package service

import (
	"bytes"
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
	"github.com/mlnomadpy/dacli/cloud/internal/tenantrepo"
)

var ErrCursorDenied = errors.New("page cursor is invalid")

type cursorCodec struct{ secret []byte }

type cursorClaims struct {
	Tenant   string `json:"tenant_id"`
	Kind     string `json:"resource_kind"`
	Parent   string `json:"parent_id,omitempty"`
	Snapshot int64  `json:"snapshot"`
	After    int64  `json:"after"`
}

func newCursorCodec(secret string) (*cursorCodec, error) {
	if len(secret) < 32 {
		return nil, errors.New("cursor codec requires at least 32 secret bytes")
	}
	return &cursorCodec{secret: []byte(secret)}, nil
}

func (c *cursorCodec) encode(scope tenant.Scope, kind, parent string, pageSnapshot, next int64) (string, error) {
	if next == 0 {
		return "", nil
	}
	payload, err := json.Marshal(cursorClaims{Tenant: string(scope.Organization), Kind: kind, Parent: parent, Snapshot: pageSnapshot, After: next})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sign(c.secret, "cursor", payload)), nil
}

func (c *cursorCodec) decode(raw string, scope tenant.Scope, kind, parent string, limit int) (tenantrepo.PageRequest, error) {
	request := tenantrepo.PageRequest{Limit: limit}
	parts := bytes.Split([]byte(raw), []byte("."))
	if len(parts) != 2 || len(parts[0]) > 2048 {
		return request, ErrCursorDenied
	}
	payload, err := base64.RawURLEncoding.DecodeString(string(parts[0]))
	if err != nil {
		return request, ErrCursorDenied
	}
	signature, err := base64.RawURLEncoding.DecodeString(string(parts[1]))
	if err != nil || !hmac.Equal(signature, sign(c.secret, "cursor", payload)) {
		return request, ErrCursorDenied
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var claims cursorClaims
	if err := decoder.Decode(&claims); err != nil || decoder.Decode(&struct{}{}) != io.EOF || claims.Tenant != string(scope.Organization) || claims.Kind != kind || claims.Parent != parent || claims.Snapshot <= 0 || claims.After <= 0 || claims.After > claims.Snapshot {
		return request, ErrCursorDenied
	}
	request.Snapshot, request.After = claims.Snapshot, claims.After
	return request, nil
}
