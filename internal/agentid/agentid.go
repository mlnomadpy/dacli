// Package agentid handles agent identity and capability attenuation.
//
// Read this before trusting it with anything: the capability system here is
// COOPERATIVE, NOT ENFORCED for agents dacli did not spawn. A subagent with
// shell access can edit the workspace markdown directly and bypass dacli
// entirely. What it buys: well-behaved agents going through dacli cannot
// clobber each other, and every write is attributed to a specific agent in a
// lineage. Enforcement becomes real only when dacli launches the child with
// the runtime's own sandbox flags (docs/RUNTIMES.md § 8).
package agentid

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// EnvVar carries the acting agent's token between processes. A parent spawns
// a child and passes the token in the child's environment — never as a
// command argument, which would land in process listings and transcripts.
const EnvVar = "DACLI_AGENT"

// RootID is the agent created by `dacli init`. It holds GrantRW.
const RootID = "a-root"

var (
	ErrBadToken    = errors.New("agent token not recognized in this workspace")
	ErrAttenuation = errors.New("cannot grant a capability exceeding your own")
)

// Identity is the resolved acting agent for this process.
type Identity struct {
	ID    string
	Grant model.Grant
	Role  string
}

// Resolve determines the acting agent. No token means the root agent — the
// ergonomic case where a human or the top-level agent drives directly. A
// token is hashed and matched against the agent files; the token itself is
// never stored anywhere, so a lost token means a new agent, not recovery.
func Resolve(w *workspace.Workspace) (*Identity, error) {
	tok := os.Getenv(EnvVar)
	if tok == "" {
		return &Identity{ID: RootID, Grant: model.GrantRW, Role: "root"}, nil
	}
	want := hashToken(tok)

	entries, err := os.ReadDir(w.AgentsDir())
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		doc, err := mdstore.ReadFile(w.AgentPath(strings.TrimSuffix(e.Name(), ".md")))
		if err != nil {
			continue
		}
		if th, _ := doc.Front.Get("token_hash"); th == want {
			id := &Identity{}
			id.ID, _ = doc.Front.Get("id")
			if g, ok := doc.Front.Get("grant"); ok {
				id.Grant = model.Grant(g)
			}
			id.Role, _ = doc.Front.Get("role")
			return id, nil
		}
	}
	return nil, ErrBadToken
}

// CanMutate reports whether this identity may rewrite an object it owns.
//
// A read-only agent is not mute: it may always append events — claim tasks,
// report findings, propose status changes. It just cannot rewrite objects,
// even ones recorded as its own.
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
// subtree is read-only, however deep it goes. The default grant is ro — the
// safe direction; widening requires saying so.
func Spawn(w *workspace.Workspace, parent *Identity, role string, grant model.Grant) (id, token string, err error) {
	if grant == "" {
		grant = model.GrantRO
	}
	if grant != model.GrantRO && grant != model.GrantRW {
		return "", "", fmt.Errorf("unknown grant %q (ro|rw)", grant)
	}
	if grant.Exceeds(parent.Grant) {
		return "", "", ErrAttenuation
	}

	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(raw[:])

	// The random half of a ULID: unique, and unlike the timestamp half, two
	// spawns in the same millisecond cannot share it.
	u := ulid.New()
	id = "a-" + strings.ToLower(u[16:])

	d := &mdstore.Doc{}
	d.Front.Set("id", id)
	d.Front.Set("kind", string(model.KindAgent))
	d.Front.Set("created", time.Now().UTC().Format(time.RFC3339))
	d.Front.Set("created_by", parent.ID)
	d.Front.Set("parent", "[["+parent.ID+"]]")
	d.Front.Set("grant", string(grant))
	if role != "" {
		d.Front.Set("role", role)
	}
	d.Front.Set("token_hash", hashToken(token))
	title := role
	if title == "" {
		title = id
	}
	d.Sections = []mdstore.Section{{Level: 1, Title: title, Content: fmt.Sprintf("Spawned by %s.\n", parent.ID)}}

	if err := mdstore.WriteFile(w.AgentPath(id), d); err != nil {
		return "", "", err
	}
	return id, token, nil
}

func hashToken(tok string) string {
	h := sha256.Sum256([]byte(tok))
	return "sha256:" + hex.EncodeToString(h[:])
}
