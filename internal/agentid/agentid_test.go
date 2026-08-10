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
	// would otherwise silently change what Resolve returns. UNSET it, not blank
	// it: after dacli 288 a set-but-empty DACLI_AGENT is a lost token that fails
	// closed, so only unsetting resolves cleanly to root.
	unsetAgentEnv(t)
	return w
}

// unsetAgentEnv removes DACLI_AGENT for the duration of the test and restores
// whatever the process started with. t.Setenv cannot unset, and after dacli 288
// setting it empty no longer means "no token" — it is a lost identity that fails
// closed — so tests that want the root identity must clear the variable
// entirely, exactly as internal/cli's TestMain does with os.Unsetenv.
func unsetAgentEnv(t *testing.T) {
	t.Helper()
	if v, ok := os.LookupEnv(EnvVar); ok {
		t.Setenv(EnvVar, v) // registers the restore, then we remove it
		_ = os.Unsetenv(EnvVar)
	}
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

// The security boundary of dacli 288, tested at the pure core so the outcome
// depends only on the explicit inputs — never on whatever DACLI_AGENT the
// process that runs this test happens to carry (the acceptance's "testable
// without relying on the environment the test runs under"). An empty
// DACLI_AGENT (present but blank) is a spawned child that LOST its token and
// must fail closed; an UNSET DACLI_AGENT is no token at all and resolves to
// root. os.Getenv cannot tell these apart; this is why Resolve uses LookupEnv.
func TestResolveTokenDistinguishesEmptyFromUnset(t *testing.T) {
	w := newWS(t)

	// Present but EMPTY: a lost identity. Must fail closed — resolving to root
	// here would escalate the child to the most privileged actor in the tree,
	// the exact fail-open this task closes.
	if id, err := resolveToken(w, "", true); err != ErrEmptyToken {
		t.Errorf("resolveToken(present, empty) = (%+v, %v); want ErrEmptyToken (fail closed)", id, err)
	}

	// NOT present: no token at all — the ergonomic direct-drive case, root.
	id, err := resolveToken(w, "", false)
	if err != nil {
		t.Fatalf("resolveToken(absent) errored: %v", err)
	}
	if id.ID != RootID || id.Grant != model.GrantRW {
		t.Errorf("resolveToken(absent) = %+v, want root rw", id)
	}
}

// Resolve wires the LookupEnv distinction through end to end: a DACLI_AGENT that
// is present but empty fails closed rather than falling back to root. This is
// the runtime path a spawned child hits when a wrapper or sandbox blanks its
// token, and it must be tested via the real env, not just the pure core.
func TestResolveFailsClosedOnEmptyToken(t *testing.T) {
	w := newWS(t) // unsets DACLI_AGENT
	t.Setenv(EnvVar, "")
	if id, err := Resolve(w); err != ErrEmptyToken {
		t.Fatalf("Resolve(empty token) = (%+v, %v); want ErrEmptyToken", id, err)
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
			//nolint:nilerr // fs.WalkDirFunc: nil skips this entry and keeps walking
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

// A minted id must READ as its role (dacli 225). `a-4w4dtttpe8` in a git log or
// a commit trailer told an operator nothing without opening the agent file; the
// role belongs in the string itself.
func TestSpawnIDsCarryTheRole(t *testing.T) {
	w := newWS(t)
	for _, role := range []string{"fixer", "go-auditor", "frontend-engineer"} {
		id, _, err := Spawn(w, rootID(), role, model.GrantRO)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(id, "a-"+role+"-") {
			t.Errorf("Spawn(role %q) minted %q; want a-%s-<disc>", role, id, role)
		}
		gotRole, disc, ok := ParseID(id)
		if !ok || gotRole != role {
			t.Errorf("ParseID(%q) = (%q, %q, %v); want role %q — a dash in the role must not break parsing", id, gotRole, disc, ok, role)
		}
		if len(disc) != discLen {
			t.Errorf("ParseID(%q) discriminator %q is %d chars, want %d", id, disc, len(disc), discLen)
		}
		if RoleOf(id) != role {
			t.Errorf("RoleOf(%q) = %q, want %q", id, RoleOf(id), role)
		}
	}
}

// An id becomes a FILENAME (agents/<id>.md), a git author name, an email
// local-part (<id>@agent.dacli) and a [[wikilink]]. A role the operator typed
// freely must not be able to take any of those out — least of all the filename,
// where a separator would write outside the agents dir.
func TestSpawnIDsAreSafePathSegments(t *testing.T) {
	w := newWS(t)
	for _, role := range []string{
		"fixer", "Go Auditor", "../../etc/passwd", "a/b", "..", "", "   ",
		"UPPER_snake.Case", "rôle-àccentué", "🙂", "-----",
		"a-really-long-role-name-that-goes-on-and-on-forever",
	} {
		id, _, err := Spawn(w, rootID(), role, model.GrantRO)
		if err != nil {
			t.Fatalf("Spawn(role %q): %v", role, err)
		}
		if !workspace.SafeSegment(id) {
			t.Errorf("Spawn(role %q) minted %q, which is not a safe path segment", role, id)
		}
		if !strings.HasPrefix(id, "a-") {
			t.Errorf("Spawn(role %q) minted %q, which lacks the a- prefix", role, id)
		}
		for _, c := range id {
			if !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') && c != '-' {
				t.Errorf("Spawn(role %q) minted %q, containing %q — ids must stay [a-z0-9-] to survive git authors, emails and wikilinks", role, id, c)
			}
		}
		// The file must actually be where AgentPath says it is.
		if _, serr := os.Stat(w.AgentPath(id)); serr != nil {
			t.Errorf("Spawn(role %q) wrote no file at AgentPath(%q): %v", role, id, serr)
		}
	}
}

// BACKWARDS COMPATIBILITY (dacli 225). Existing workspaces are full of
// old-format ids in agent files, task logs, event bodies and commit trailers.
// A resolve/parse path that only understood the new format would orphan every
// agent minted before this change.
func TestOldFormatIDsStillResolve(t *testing.T) {
	w := newWS(t)
	const old = "a-4w4dtttpe8" // verbatim from this repo's own history
	token := "0123456789abcdef0123456789abcdef"

	d := &mdstore.Doc{}
	d.Front.Set("id", old)
	d.Front.Set("kind", string(model.KindAgent))
	d.Front.Set("parent", "[["+RootID+"]]")
	d.Front.Set("grant", string(model.GrantRW))
	d.Front.Set("role", "frontend-engineer")
	sum := sha256.Sum256([]byte(token))
	d.Front.Set("token_hash", "sha256:"+hex.EncodeToString(sum[:]))
	if err := mdstore.WriteFile(w.AgentPath(old), d); err != nil {
		t.Fatal(err)
	}

	t.Setenv(EnvVar, token)
	id, err := Resolve(w)
	if err != nil {
		t.Fatalf("Resolve rejected an old-format agent: %v", err)
	}
	if id.ID != old || id.Grant != model.GrantRW || id.Role != "frontend-engineer" {
		t.Errorf("old-format agent resolved as %+v, want {%s rw frontend-engineer}", id, old)
	}
	// CanMutate keys on the id string, so it must not care about the format.
	if !id.CanMutate(old) || id.CanMutate("a-someone-else") {
		t.Error("CanMutate misjudged an old-format id")
	}
	// And the old id is still parseable and still recognised as an agent id —
	// it simply carries no role, which is exactly why 225 exists.
	role, disc, ok := ParseID(old)
	if !ok || role != "" || disc != "4w4dtttpe8" {
		t.Errorf("ParseID(%q) = (%q, %q, %v); an old id must parse with an empty role", old, role, disc, ok)
	}
	if !IsID(old) {
		t.Errorf("IsID(%q) = false; old ids are still agent ids", old)
	}
	// An old-format child still spawns, and its parent link still points at it.
	kidID, _, err := Spawn(w, id, "reviewer", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	kid, err := mdstore.ReadFile(w.AgentPath(kidID))
	if err != nil {
		t.Fatal(err)
	}
	if p, _ := kid.Front.Get("parent"); p != "[["+old+"]]" {
		t.Errorf("child of an old-format agent recorded parent %q, want [[%s]]", p, old)
	}
}

// ParseID must stay unambiguous for every shape an id takes, including the root
// (whose role is implicit in its name) and non-ids handed to it by a blame or
// trailer reader that only has a string.
func TestParseID(t *testing.T) {
	cases := []struct {
		id, role, disc string
		ok             bool
	}{
		{"a-fixer-7k3q4b", "fixer", "7k3q4b", true},
		{"a-go-auditor-2m8x9v", "go-auditor", "2m8x9v", true},
		{"a-4w4dtttpe8", "", "4w4dtttpe8", true}, // old format
		{RootID, "root", "", true},
		{"a-x", "", "x", true},
		{"", "", "", false},
		{"a-", "", "", false},
		{"Taha Bouhsine", "", "", false},
		{"dependabot[bot]", "", "", false},
	}
	for _, tc := range cases {
		role, disc, ok := ParseID(tc.id)
		if role != tc.role || disc != tc.disc || ok != tc.ok {
			t.Errorf("ParseID(%q) = (%q, %q, %v); want (%q, %q, %v)", tc.id, role, disc, ok, tc.role, tc.disc, tc.ok)
		}
	}
}

// The readable half must not leak the credential. The discriminator comes from
// a fresh ULID, never from the token or its hash — an id printed in every commit
// must not narrow the search space for the token it was minted with.
func TestIDDoesNotLeakTheToken(t *testing.T) {
	w := newWS(t)
	for i := 0; i < 25; i++ {
		id, token, err := Spawn(w, rootID(), "fixer", model.GrantRO)
		if err != nil {
			t.Fatal(err)
		}
		// The whole token, obviously, must never appear.
		if strings.Contains(id, token) {
			t.Fatalf("id %q contains the raw token", id)
		}
		// And no window as long as the discriminator itself: any scheme that
		// derived the discriminator from the token — a prefix, a suffix, an
		// encoded hash slice — produces a full-length match and trips here on
		// the first spawn.
		//
		// The window length has to be the discriminator length, not something
		// shorter. A shorter window tests nothing about derivation and is a
		// coincidence detector instead: with ~45 windows over the token and a
		// handful of positions in the id, a 4-character match lands by chance
		// on roughly one run in twenty, which is how this test used to fail on
		// perfectly good ids.
		_, disc, ok := ParseID(id)
		if !ok || disc == "" {
			t.Fatalf("ParseID(%q) gave no discriminator", id)
		}
		for j := 0; j+len(disc) <= len(token); j++ {
			if strings.Contains(id, token[j:j+len(disc)]) {
				t.Fatalf("id %q contains token fragment %q — the discriminator is derived from the token", id, token[j:j+len(disc)])
			}
		}
		// And nothing in the workspace holds the raw token — re-asserted here
		// because the id change touched what Spawn writes.
		if err := filepath.WalkDir(filepath.Join(w.Root, ".dacli"), func(path string, d fs.DirEntry, werr error) error {
			if werr != nil || d.IsDir() {
				return werr
			}
			raw, rerr := os.ReadFile(path)
			if rerr == nil && strings.Contains(string(raw), token) {
				t.Errorf("raw token leaked into %s", path)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
}

// Collision safety under load: readability must not cost uniqueness. Many
// agents in the SAME role share a slug, so the discriminator plus the re-mint
// on an existing file is all that separates them — and a collision would not
// merely duplicate a name, it would overwrite another agent's token hash.
func TestSpawnManyInOneRoleNeverCollides(t *testing.T) {
	w := newWS(t)
	seen := map[string]bool{}
	for i := 0; i < 300; i++ {
		id, _, err := Spawn(w, rootID(), "fixer", model.GrantRO)
		if err != nil {
			t.Fatalf("spawn %d: %v", i, err)
		}
		if seen[id] {
			t.Fatalf("duplicate agent id %q on spawn %d", id, i)
		}
		seen[id] = true
	}
	files, err := os.ReadDir(w.AgentsDir())
	if err != nil {
		t.Fatal(err)
	}
	// 300 spawns + the root agent: one file each, none overwritten.
	if len(files) != 301 {
		t.Errorf("%d agent files after 300 spawns; a collision silently overwrote one", len(files))
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
