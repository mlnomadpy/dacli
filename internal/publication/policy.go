// Package publication defines the single outbound GitHub disclosure policy.
// Feature slices build their text, JSON, CLI, and MCP projections from this
// typed decision instead of independently deciding which local evidence is
// safe to publish (issue #873).
package publication

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var (
	absolutePath  = regexp.MustCompile(`(?m)(?:/[A-Za-z0-9._-]+){2,}|[A-Za-z]:\\(?:[^\\\s]+\\)+[^\\\s]+`)
	agentIdentity = regexp.MustCompile(`\ba-[A-Za-z0-9][A-Za-z0-9_-]*\b`)
	secretToken   = regexp.MustCompile(`(?i)\b(?:token|api[_-]?key|secret)\s*[:=]\s*[^\s,;]+`)
	webURL        = regexp.MustCompile(`https?://[^\s<>()\[\]{}"']+`)
)

const Schema = "github-publication/v1"

type Visibility string

const (
	VisibilityUnknown Visibility = "unknown"
	VisibilityPrivate Visibility = "private"
	VisibilityPublic  Visibility = "public"
)

type Field string

const (
	FieldTitle       Field = "title"
	FieldSummary     Field = "summary"
	FieldAcceptance  Field = "acceptance"
	FieldIssueRef    Field = "issue_reference"
	FieldIssueClose  Field = "issue_closure"
	FieldFindings    Field = "findings"
	FieldVerdicts    Field = "verdicts"
	FieldDecisions   Field = "decisions"
	FieldTranscripts Field = "transcripts"
	FieldJournals    Field = "journals"
	FieldRecovery    Field = "recovery_details"
	FieldLocalPaths  Field = "local_paths"
	FieldCosts       Field = "tokens_and_costs"
	FieldIdentities  Field = "agent_identities"
)

var publicAllowlist = []Field{FieldTitle, FieldSummary, FieldAcceptance, FieldIssueRef}
var internalFields = []Field{FieldFindings, FieldVerdicts, FieldDecisions, FieldTranscripts, FieldJournals, FieldRecovery, FieldLocalPaths, FieldCosts, FieldIdentities}

type Withheld struct {
	Field  Field  `json:"field"`
	Reason string `json:"reason"`
}

// Policy is the stable machine-readable decision every GitHub publisher uses.
// Unknown visibility is intentionally equivalent to public for disclosure.
type Policy struct {
	Schema            string     `json:"schema"`
	Visibility        Visibility `json:"visibility"`
	Mode              string     `json:"mode"`
	Repository        string     `json:"repository,omitempty"`
	InternalAuthority bool       `json:"internal_authority"`
	TerminalAuthority bool       `json:"terminal_authority"`
	Allowed           []Field    `json:"allowed"`
	Withheld          []Withheld `json:"withheld"`
}

func ParseVisibility(raw string) Visibility {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "PRIVATE", "INTERNAL":
		return VisibilityPrivate
	case "PUBLIC":
		return VisibilityPublic
	default:
		return VisibilityUnknown
	}
}

// New derives the complete decision. includeInternal is an operator request;
// recordedInternal is durable authority scoped by the caller to the exact
// repository. A request without authority remains withheld.
func New(repository, rawVisibility string, includeInternal, recordedInternal, terminal bool) Policy {
	visibility := ParseVisibility(rawVisibility)
	p := Policy{Schema: Schema, Visibility: visibility, Repository: repository, TerminalAuthority: terminal, Withheld: []Withheld{}}
	if visibility == VisibilityPrivate {
		p.Mode = "private-explicit"
		p.InternalAuthority = true
		p.Allowed = append([]Field{}, publicAllowlist...)
		for _, field := range internalFields {
			// Verdict publication has always been an explicit --with-verdicts
			// action, including on private repositories.
			if field != FieldVerdicts || includeInternal {
				p.Allowed = append(p.Allowed, field)
			}
		}
		if !includeInternal {
			p.Withheld = append(p.Withheld, Withheld{Field: FieldVerdicts, Reason: "verdict publication requires the explicit --with-verdicts request"})
		}
	} else {
		p.Mode = "public-safe"
		p.Allowed = append([]Field{}, publicAllowlist...)
		// An unrecognized visibility can be public. No prior grant is evidence
		// about that unknown state, so it can never enable internal fields.
		p.InternalAuthority = visibility == VisibilityPublic && includeInternal && recordedInternal
		if p.InternalAuthority {
			p.Mode = "public-disclosed"
			p.Allowed = append(p.Allowed, internalFields...)
		} else {
			reason := "public-safe allowlist: separate recorded internal-disclosure authority is required"
			if visibility == VisibilityUnknown {
				reason = "repository visibility is unknown; disclosure fails closed to the public-safe allowlist"
			}
			for _, field := range internalFields {
				p.Withheld = append(p.Withheld, Withheld{Field: field, Reason: reason})
			}
		}
	}
	if terminal {
		p.Allowed = append(p.Allowed, FieldIssueClose)
	} else {
		p.Withheld = append(p.Withheld, Withheld{Field: FieldIssueClose, Reason: "nonterminal delivery uses a non-closing issue reference"})
	}
	sort.Slice(p.Allowed, func(i, j int) bool { return p.Allowed[i] < p.Allowed[j] })
	sort.Slice(p.Withheld, func(i, j int) bool { return p.Withheld[i].Field < p.Withheld[j].Field })
	return p
}

func (p Policy) Allows(field Field) bool {
	for _, allowed := range p.Allowed {
		if allowed == field {
			return true
		}
	}
	return false
}

func (p Policy) Reason(field Field) string {
	for _, withheld := range p.Withheld {
		if withheld.Field == field {
			return withheld.Reason
		}
	}
	return "field is outside the selected publication policy"
}

// Sanitize applies the content-level half of public-safe projection. The
// allowlist decides which sections exist; this pass prevents an allowed task
// title/acceptance sentence from smuggling private paths, secrets, or internal
// agent identities inside it.
func (p Policy) Sanitize(value string) string {
	if p.Mode != "public-safe" {
		return value
	}
	var out strings.Builder
	last := 0
	for _, span := range webURL.FindAllStringIndex(value, -1) {
		out.WriteString(sanitizePublicText(value[last:span[0]]))
		out.WriteString(sanitizeWebURL(value[span[0]:span[1]]))
		last = span[1]
	}
	out.WriteString(sanitizePublicText(value[last:]))
	return out.String()
}

func sanitizePublicText(value string) string {
	value = secretToken.ReplaceAllString(value, "<withheld-secret>")
	value = absolutePath.ReplaceAllString(value, "<withheld-local-path>")
	return agentIdentity.ReplaceAllString(value, "<withheld-agent-identity>")
}

// sanitizeWebURL preserves ordinary public links as links. Applying the local
// path regexp to the entire string changed https://github.com/org/repo into
// https:<withheld-local-path>. Credentials and secret-named query parameters
// remain withheld, so URL preservation is not a secret-disclosure bypass.
func sanitizeWebURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return sanitizePublicText(raw)
	}
	if u.User != nil {
		if _, hasPassword := u.User.Password(); hasPassword {
			u.User = url.User(u.User.Username())
		}
	}
	query := u.Query()
	changed := false
	for key := range query {
		switch strings.ToLower(strings.ReplaceAll(key, "-", "_")) {
		case "token", "api_key", "secret":
			query.Set(key, "<withheld-secret>")
			changed = true
		}
	}
	if changed {
		u.RawQuery = query.Encode()
	}
	return u.String()
}

func (p Policy) WriteText(w io.Writer) {
	fmt.Fprintf(w, "publication policy: %s (visibility=%s, repository=%s)\n", p.Mode, p.Visibility, p.Repository)
	for _, item := range p.Withheld {
		fmt.Fprintf(w, "withheld %s: %s\n", item.Field, item.Reason)
	}
}

func (p Policy) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(p)
}
