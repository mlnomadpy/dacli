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

// Agent ids carry their ROLE, not just entropy (dacli 225).
//
// An id is embedded in places nobody will cross-reference by hand: git author
// names, Dacli-Agent trailers, task logs, event actors, [[wikilinks]]. A pure
// ULID tail like `a-4w4dtttpe8` forces a reader of `git log` to go open
// .dacli/agents/<id>.md to learn anything at all — which is why this repo's own
// history is full of authors patched into `a-j846nahs42 (frontend-engineer)`.
// Minting `a-frontend-engineer-7k3q` instead puts the answer in the string.
//
// The discriminator, not the role, is what makes an id unique, so it keeps a
// full ULID-random-half slice and Spawn re-mints on any file collision. It is
// drawn from a FRESH ulid.New(), never from the token or its hash: the readable
// half of an id must not narrow the search space for a credential.
const (
	// discLen is how many trailing (random-half) ULID characters end an id.
	// 6 Crockford-base32 chars = 30 bits, plus a re-mint on collision.
	discLen = 6
	// maxRoleSlug caps the readable half so an id stays a comfortable filename
	// and git author name even when a role is named at length.
	maxRoleSlug = 24
	// mintAttempts bounds the collision re-mint loop. Reaching it means either
	// crypto/rand is degenerate or the agents dir is unreadable; both deserve an
	// error rather than an id that silently overwrites another agent's file.
	mintAttempts = 8
)

var (
	ErrBadToken    = errors.New("agent token not recognized in this workspace")
	ErrAttenuation = errors.New("cannot grant a capability exceeding your own")
	// ErrEmptyToken is returned when DACLI_AGENT is SET but EMPTY (dacli 288):
	// a token slot that was handed over and then blanked out of the environment.
	// os.Getenv cannot tell this from "unset"; os.LookupEnv can. It is a lost
	// identity and must fail closed rather than resolve to root, or a token loss
	// silently becomes a privilege GAIN.
	ErrEmptyToken = errors.New("DACLI_AGENT is set but empty (lost agent identity); refusing to fall back to root")
)

// Identity is the resolved acting agent for this process.
type Identity struct {
	ID    string
	Grant model.Grant
	Role  string
}

// Resolve determines the acting agent from the DACLI_AGENT environment
// variable. It splits the raw env lookup from the decision (resolveToken) so
// the empty-vs-unset distinction can be tested with explicit inputs, never by
// mutating the environment the test itself runs under (dacli 288).
func Resolve(w *workspace.Workspace) (*Identity, error) {
	tok, present := os.LookupEnv(EnvVar)
	return resolveToken(w, tok, present)
}

// resolveToken is the environment-free core of Resolve. The empty-vs-unset
// distinction is a security boundary, not a nicety (dacli 288). os.Getenv
// collapses both into ""; os.LookupEnv keeps them apart, and they mean opposite
// things:
//
//   - NOT PRESENT — no token at all. The ergonomic, documented case ("unset
//     means the root agent", cli.go): a human or the top-level agent drives
//     directly as root with a read-write grant.
//   - PRESENT BUT EMPTY — a token slot that was handed over and then blanked.
//     A spawn always sets DACLI_AGENT to the child's token (execution.go), so an
//     EMPTY value is a credential a nested subprocess, an env-sanitizing wrapper,
//     or a sandbox stripped out. Falling back to root here would escalate that
//     lost child to a-root RW — turning a token LOSS into a privilege GAIN and
//     undoing the attenuation Spawn enforces. It MUST fail closed.
//   - PRESENT AND NON-EMPTY — hash it and match an agent file, as before; an
//     unrecognized token is ErrBadToken (also fail closed).
func resolveToken(w *workspace.Workspace, tok string, present bool) (*Identity, error) {
	if !present {
		return &Identity{ID: RootID, Grant: model.GrantRW, Role: "root"}, nil
	}
	if tok == "" {
		return nil, ErrEmptyToken
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

// MutateRefusal names WHY CanMutate refused, for messages that must not blame
// an actual read-only grant when the real reason is that an rw agent simply
// isn't this task's owner — those are different situations with different
// remedies, and conflating them under one "(read-only grant)" label misleads
// an rw agent into thinking a grant change would help.
func (i *Identity) MutateRefusal() string {
	if i.Grant != model.GrantRW {
		return "read-only grant"
	}
	return "not the owner"
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

	id, err = mintID(w, role)
	if err != nil {
		return "", "", err
	}

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

// mintID returns a fresh, unused id of the form a-<role-slug>-<disc> (dacli
// 225), falling back to a-<disc> when the role is empty or unslugifiable.
//
// The uniqueness check is the agent FILE, not an in-memory set: ids outlive the
// process that minted them, and an id that collides would silently overwrite
// another agent's token hash — turning a collision into an identity takeover
// rather than a duplicate name.
func mintID(w *workspace.Workspace, role string) (string, error) {
	slug := roleSlug(role)
	for attempt := 0; attempt < mintAttempts; attempt++ {
		id := "a-" + discriminator()
		if slug != "" {
			id = "a-" + slug + "-" + discriminator()
		}
		// Defence in depth: AgentPath joins the id straight onto the agents dir,
		// so an id that is not a single path segment would write outside it.
		// roleSlug already guarantees [a-z0-9-]; this refuses to trust that.
		if !workspace.SafeSegment(id) {
			slug = ""
			continue
		}
		if _, err := os.Stat(w.AgentPath(id)); os.IsNotExist(err) {
			return id, nil
		}
	}
	return "", fmt.Errorf("could not mint an unused agent id for role %q in %d attempts", role, mintAttempts)
}

// discriminator returns discLen lowercase characters from the RANDOM half of a
// fresh ULID. The timestamp half is deliberately excluded: two spawns in the
// same millisecond would share it.
func discriminator() string {
	u := ulid.New()
	return strings.ToLower(u[len(u)-discLen:])
}

// roleSlug reduces a role name to [a-z0-9-]: the character class that is
// simultaneously a safe path segment, a legal git author name, a legal email
// local-part (ids become <id>@agent.dacli), and wikilink-safe. Anything else
// becomes a single dash; leading/trailing dashes are dropped so the id can
// never end up with an empty segment.
func roleSlug(role string) string {
	var b strings.Builder
	dash := true // leading separators are dropped
	for _, r := range strings.ToLower(strings.TrimSpace(role)) {
		if b.Len() >= maxRoleSlug {
			break
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// ParseID splits an agent id into its readable role slug and its discriminator
// (dacli 225), so a bare id read off `git log` or a commit trailer answers
// "which role wrote this" with no workspace lookup.
//
// Parsing stays unambiguous even though role names contain dashes
// (`go-auditor`): the discriminator is always the LAST dash-separated segment,
// so everything between the `a-` prefix and it is the role.
//
// OLD ids (`a-4w4dtttpe8`, minted before readable ids) parse fine and report an
// empty role — they are still valid ids, and nothing in dacli may treat an
// unparseable role as an invalid agent.
func ParseID(id string) (role, disc string, ok bool) {
	if !strings.HasPrefix(id, "a-") || len(id) <= 2 {
		return "", "", false
	}
	if id == RootID {
		return "root", "", true
	}
	rest := id[2:]
	if i := strings.LastIndexByte(rest, '-'); i > 0 {
		return rest[:i], rest[i+1:], true
	}
	return "", rest, true
}

// RoleOf reports the role an id carries, or "" for an old-style or roleless id.
// It is a display aid: the AUTHORITATIVE role is the `role:` field of
// agents/<id>.md, which holds the role's exact name; the id holds its slug.
func RoleOf(id string) string {
	role, _, _ := ParseID(id)
	return role
}

// IsID reports whether s names a dacli agent. Used where attribution has only a
// string to go on — a git author line, a trailer value — and must decide
// whether it is looking at an agent or at a human collaborator.
func IsID(s string) bool {
	_, _, ok := ParseID(s)
	return ok
}
