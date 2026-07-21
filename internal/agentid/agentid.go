// Package agentid handles agent identity and capability attenuation.
//
// Read this before trusting it with anything: the capability system here is
// COOPERATIVE, NOT ENFORCED. A subagent has shell access — that is what makes
// it useful — so it can edit the workspace markdown directly and bypass dacli
// entirely. Nothing in this package prevents that, and nothing in it should
// ever be described as if it did.
//
// What it does buy: well-behaved agents going through dacli cannot clobber
// each other, and every write is attributed to a specific agent in a lineage.
// That is worth having. It is not a security boundary.
//
// See DESIGN.md § 6 for what an enforced version would require (a daemon
// owning the only writable copy, children on a read-only mount).
package agentid

import (
	"errors"
	"os"

	"github.com/mlnomadpy/dacli/internal/model"
)

// EnvVar carries the acting agent's token between processes. A parent spawns
// a child and passes the token in the child's environment.
const EnvVar = "DACLI_AGENT"

// RootID is the agent created by `dacli init`. It holds GrantRW.
const RootID = "a-root"

var (
	ErrNoIdentity   = errors.New("no agent identity: set " + EnvVar + " or run as the workspace owner")
	ErrBadToken     = errors.New("agent token not recognized")
	ErrNotPermitted = errors.New("this agent holds a read-only grant; append an event instead")
	ErrAttenuation  = errors.New("cannot grant a capability exceeding your own")
)

// Identity is the resolved acting agent for this process.
type Identity struct {
	ID    string
	Grant model.Grant
	Role  string
}

// Current resolves the acting agent from the environment. With no token set,
// it falls back to the root agent — the ergonomic case where a human or the
// top-level agent is driving directly.
func Current() (*Identity, error) {
	tok := os.Getenv(EnvVar)
	if tok == "" {
		return &Identity{ID: RootID, Grant: model.GrantRW}, nil
	}
	// TODO: hash the token, match it against agents/*.md token_hash.
	return nil, ErrBadToken
}

// CanMutate reports whether this identity may rewrite an object it owns.
//
// A read-only agent is not mute. It may always append events — that is how it
// claims tasks, reports findings, and proposes status changes. A read-only
// agent that could not report results would be useless.
func (i *Identity) CanMutate(ownerID string) bool {
	if i.Grant != model.GrantRW {
		return false
	}
	return ownerID == "" || ownerID == i.ID
}

// Spawn mints a child agent under parent. The returned token is displayed
// once and never persisted; only its hash is written to agents/<id>.md.
//
// Attenuation is monotonic and enforced here: a read-only agent's entire
// subtree is read-only, however deep it goes.
func Spawn(parent *Identity, role string, grant model.Grant) (id, token string, err error) {
	if grant.Exceeds(parent.Grant) {
		return "", "", ErrAttenuation
	}
	// TODO: generate a ULID-suffixed id and a random token; write agents/<id>.md
	// with parent, grant, role, and token_hash.
	return "", "", errors.New("not implemented")
}
