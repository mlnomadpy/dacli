package agentid

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func newWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "capability-test")
	if err != nil {
		t.Fatal(err)
	}
	// Every test in this package resolves identity from the environment; an
	// inherited DACLI_AGENT (dacli running its own tests through a child agent)
	// would otherwise silently change what Resolve returns.
	t.Setenv(EnvVar, "")
	return w
}

func rootID() *Identity { return &Identity{ID: RootID, Grant: model.GrantRW, Role: "root"} }

// No token means the ergonomic case: a human (or the top-level agent) driving
// directly is the ROOT identity with a read-write grant. If this ever resolved
// to something narrower, every unauthenticated invocation would start failing
// capability checks; if it resolved to a NAMED agent, attribution would lie.
func TestResolveWithNoTokenIsRoot(t *testing.T) {
	w := newWS(t)
	id, err := Resolve(w)
	if err != nil {
		t.Fatal(err)
	}
	if id.ID != RootID || id.Grant != model.GrantRW || id.Role != "root" {
		t.Errorf("Resolve with no token = %+v, want {%s rw root}", id, RootID)
	}
}

// A minted token must resolve back to exactly the child it was minted for —
// with the child's OWN grant and role, never the parent's.
func TestResolveMatchesMintedToken(t *testing.T) {
	w := newWS(t)
	childID, token, err := Spawn(w, rootID(), "reviewer", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvVar, token)
	id, err := Resolve(w)
	if err != nil {
		t.Fatal(err)
	}
	if id.ID != childID {
		t.Errorf("resolved id = %q, want %q", id.ID, childID)
	}
	if id.Grant != model.GrantRO {
		t.Errorf("resolved grant = %q, want ro (the CHILD's grant, not the parent's)", id.Grant)
	}
	if id.Role != "reviewer" {
		t.Errorf("resolved role = %q, want reviewer", id.Role)
	}
}

// An unrecognized token is a hard error, never a silent fall-back to root —
// that fall-back would turn a bad token into an unnoticed privilege escalation.
func TestResolveRejectsUnknownToken(t *testing.T) {
	w := newWS(t)
	if _, _, err := Spawn(w, rootID(), "junior", model.GrantRO); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvVar, "deadbeefdeadbeefdeadbeefdeadbeef")
	id, err := Resolve(w)
	if err != ErrBadToken {
		t.Fatalf("Resolve(bad token) = (%+v, %v); want ErrBadToken", id, err)
	}
}

// The token is displayed once and never persisted: only its sha256 lands on
// disk. A raw token anywhere in the workspace would make `git log` a credential
// store.
func TestSpawnPersistsOnlyTheTokenHash(t *testing.T) {
	w := newWS(t)
	childID, token, err := Spawn(w, rootID(), "junior", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(token))
	wantHash := "sha256:" + hex.EncodeToString(sum[:])

	doc, err := mdstore.ReadFile(w.AgentPath(childID))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := doc.Front.Get("token_hash"); got != wantHash {
		t.Errorf("token_hash = %q, want %q", got, wantHash)
	}
	if got, _ := doc.Front.Get("parent"); got != "[["+RootID+"]]" {
		t.Errorf("parent = %q, want a wikilink to %s", got, RootID)
	}

	err = filepath.WalkDir(filepath.Join(w.Root, ".dacli"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		if strings.Contains(string(raw), token) {
			t.Errorf("raw token leaked into %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Attenuation is the whole capability model: a child may never hold a grant its
// parent does not. Enumerated rather than sampled — every (parent, requested)
// pair, so a new grant value cannot quietly slip past one example.
func TestSpawnAttenuation(t *testing.T) {
	cases := []struct {
		name      string
		parent    model.Grant
		requested model.Grant
		want      model.Grant // "" = expect a refusal
		wantErr   error
	}{
		{"rw parent grants rw", model.GrantRW, model.GrantRW, model.GrantRW, nil},
		{"rw parent grants ro", model.GrantRW, model.GrantRO, model.GrantRO, nil},
		{"rw parent defaults to ro", model.GrantRW, "", model.GrantRO, nil},
		{"ro parent grants ro", model.GrantRO, model.GrantRO, model.GrantRO, nil},
		{"ro parent defaults to ro", model.GrantRO, "", model.GrantRO, nil},
		{"ro parent CANNOT grant rw", model.GrantRO, model.GrantRW, "", ErrAttenuation},
		{"empty parent grant cannot grant rw", "", model.GrantRW, "", ErrAttenuation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newWS(t)
			before, _ := os.ReadDir(w.AgentsDir())
			parent := &Identity{ID: "a-parent", Grant: tc.parent}
			childID, token, err := Spawn(w, parent, "worker", tc.requested)
			if tc.wantErr != nil {
				if err != tc.wantErr {
					t.Fatalf("Spawn = (%q, err %v); want %v", childID, err, tc.wantErr)
				}
				if childID != "" || token != "" {
					t.Errorf("a refused spawn must mint nothing; got id %q token %q", childID, token)
				}
				if after, _ := os.ReadDir(w.AgentsDir()); len(after) != len(before) {
					t.Errorf("a refused spawn wrote %d agent file(s)", len(after)-len(before))
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			doc, derr := mdstore.ReadFile(w.AgentPath(childID))
			if derr != nil {
				t.Fatal(derr)
			}
			if got, _ := doc.Front.Get("grant"); model.Grant(got) != tc.want {
				t.Errorf("recorded grant = %q, want %q", got, tc.want)
			}
		})
	}
}

// An unknown grant string is rejected outright rather than stored verbatim: a
// typo'd grant that is neither ro nor rw would fail every == comparison and so
// behave like read-only by accident instead of by decision.
func TestSpawnRejectsUnknownGrant(t *testing.T) {
	w := newWS(t)
	for _, bad := range []model.Grant{"admin", "RW", "read-write", "rwx"} {
		if _, _, err := Spawn(w, rootID(), "worker", bad); err == nil {
			t.Errorf("Spawn accepted unknown grant %q", bad)
		}
	}
}

// Monotonicity: a read-only agent's ENTIRE subtree is read-only, however deep.
// Walking the lineage rather than checking one hop is the point — a single-hop
// check would miss a grandchild re-widening its grant.
func TestAttenuationIsMonotonicDownTheSubtree(t *testing.T) {
	w := newWS(t)
	childID, childTok, err := Spawn(w, rootID(), "lead", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvVar, childTok)
	child, err := Resolve(w)
	if err != nil {
		t.Fatal(err)
	}
	if child.ID != childID || child.Grant != model.GrantRO {
		t.Fatalf("child resolved as %+v", child)
	}
	// Depth 2: the read-only child cannot widen for its own child...
	if _, _, err := Spawn(w, child, "junior", model.GrantRW); err != ErrAttenuation {
		t.Fatalf("ro child spawning rw grandchild = %v; want ErrAttenuation", err)
	}
	// ...and the ro grandchild it CAN mint cannot widen either.
	gcID, gcTok, err := Spawn(w, child, "junior", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvVar, gcTok)
	gc, err := Resolve(w)
	if err != nil {
		t.Fatal(err)
	}
	if gc.ID != gcID || gc.Grant != model.GrantRO {
		t.Fatalf("grandchild resolved as %+v, want %s/ro", gc, gcID)
	}
	if _, _, err := Spawn(w, gc, "intern", model.GrantRW); err != ErrAttenuation {
		t.Errorf("ro grandchild spawning rw = %v; want ErrAttenuation", err)
	}
}

// Two spawns inside the same millisecond must not collide: the id is built from
// the RANDOM half of a ULID precisely because the timestamp half would.
func TestSpawnIDsAreUniqueWithinAMillisecond(t *testing.T) {
	w := newWS(t)
	seen := map[string]bool{}
	tokens := map[string]bool{}
	for i := 0; i < 50; i++ {
		id, tok, err := Spawn(w, rootID(), "worker", model.GrantRO)
		if err != nil {
			t.Fatal(err)
		}
		if seen[id] {
			t.Fatalf("duplicate agent id %q on spawn %d", id, i)
		}
		if tokens[tok] {
			t.Fatalf("duplicate token on spawn %d", i)
		}
		if !strings.HasPrefix(id, "a-") {
			t.Fatalf("agent id %q lacks the a- prefix", id)
		}
		seen[id], tokens[tok] = true, true
	}
}

// CanMutate is the write gate. Two independent conditions: an rw grant, AND
// ownership of the object. Enumerated because every call site depends on both.
func TestCanMutate(t *testing.T) {
	cases := []struct {
		name  string
		grant model.Grant
		owner string
		want  bool
	}{
		{"rw owns it", model.GrantRW, "a-me", true},
		{"rw, unowned object", model.GrantRW, "", true},
		{"rw but someone else owns it", model.GrantRW, "a-other", false},
		{"ro owns it — still cannot rewrite", model.GrantRO, "a-me", false},
		{"ro, unowned object", model.GrantRO, "", false},
		{"ro, another owner", model.GrantRO, "a-other", false},
		{"empty grant is not rw", "", "a-me", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := &Identity{ID: "a-me", Grant: tc.grant}
			if got := id.CanMutate(tc.owner); got != tc.want {
				t.Errorf("CanMutate(%q) with grant %q = %v, want %v", tc.owner, tc.grant, got, tc.want)
			}
		})
	}
}

// MutateRefusal must not blame the grant when the real reason is ownership: an
// rw agent told "read-only grant" would go chase a grant change that cannot
// help. The two situations have different remedies.
func TestMutateRefusalNamesTheActualReason(t *testing.T) {
	if got := (&Identity{ID: "a-me", Grant: model.GrantRW}).MutateRefusal(); got != "not the owner" {
		t.Errorf("rw non-owner refusal = %q, want %q", got, "not the owner")
	}
	if got := (&Identity{ID: "a-me", Grant: model.GrantRO}).MutateRefusal(); got != "read-only grant" {
		t.Errorf("ro refusal = %q, want %q", got, "read-only grant")
	}
}

// The token travels in the ENVIRONMENT, never argv — an argv token would land
// in `ps` output and in every transcript that records the invocation.
func TestEnvVarName(t *testing.T) {
	if EnvVar != "DACLI_AGENT" {
		t.Errorf("EnvVar = %q; renaming it breaks every already-spawned child", EnvVar)
	}
}
