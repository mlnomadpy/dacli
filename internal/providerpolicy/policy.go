// Package providerpolicy classifies provider failures and applies retry,
// cooldown, and explicit fallback policy without coupling orchestration to a
// vendor SDK.
package providerpolicy

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Kind string

const (
	RateLimited    Kind = "rate_limit"
	QuotaExhausted Kind = "quota_exhausted"
	Authentication Kind = "authentication"
	Unavailable    Kind = "unavailable"
	PermanentInput Kind = "permanent_input"
	PolicyRefusal  Kind = "policy_refusal"
	Unknown        Kind = "unknown"
)

type Outcome struct {
	Kind       Kind
	Reason     string
	ResetAfter time.Duration
}

// Transition is the one printable and recordable representation of a pause or
// fallback, keeping operator output and the durable event body identical.
type Transition struct {
	Source, Destination string
	Reason              string
	Cooldown            time.Duration
}

func (t Transition) String() string {
	destination := t.Destination
	if destination == "" {
		destination = "none"
	}
	return fmt.Sprintf("source=%s destination=%s reason=%s cooldown=%s", t.Source, destination, t.Reason, t.Cooldown)
}

func (o Outcome) Retryable() bool { return o.Kind == RateLimited || o.Kind == Unavailable }
func (o Outcome) Fallbackable() bool {
	return o.Kind == RateLimited || o.Kind == QuotaExhausted || o.Kind == Unavailable
}

var retryAfterRE = regexp.MustCompile(`(?i)(?:retry[-_ ]after|reset[-_ ]after)["':= ]+([0-9]+)`)

// Classify maps the stable signals emitted by the supported JSON adapters and
// their text fallbacks. Specific permanent outcomes precede broad transient
// matches so an invalid request mentioning a quota cannot become retryable.
func Classify(exitCode int, output string) Outcome {
	lower := strings.ToLower(output)
	kind := Unknown
	switch {
	case exitCode == 3 || containsAny(lower, "policy refusal", "permission denied by policy", "content policy"):
		kind = PolicyRefusal
	case containsAny(lower, "invalid_request", "invalid request", "bad request", "invalid argument", "context_length_exceeded"):
		kind = PermanentInput
	case containsAny(lower, "authentication_error", "unauthorized", "invalid api key", "authentication failed"):
		kind = Authentication
	case containsAny(lower, "insufficient_quota", "quota exhausted", "quota_exceeded", "billing hard limit"):
		kind = QuotaExhausted
	case containsAny(lower, "rate_limit", "rate limit", "too many requests", "http 429"):
		kind = RateLimited
	case containsAny(lower, "service_unavailable", "temporarily unavailable", "http 503", "overloaded"):
		kind = Unavailable
	}
	return Outcome{Kind: kind, Reason: firstLine(output), ResetAfter: resetAfter(output)}
}

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 240 {
		s = s[:240]
	}
	return s
}

func resetAfter(output string) time.Duration {
	var object map[string]any
	if json.Unmarshal([]byte(output), &object) == nil {
		for _, key := range []string{"retry_after", "retry_after_seconds", "reset_after"} {
			if n, ok := number(object[key]); ok && n >= 0 {
				return time.Duration(n * float64(time.Second))
			}
		}
	}
	m := retryAfterRE.FindStringSubmatch(output)
	if len(m) == 2 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return 0
}

func number(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

type RetryPolicy struct {
	Base, Max time.Duration
	Jitter    float64
	Rand      *rand.Rand
}

func (p RetryPolicy) Delay(attempt int, outcome Outcome) (time.Duration, bool) {
	if !outcome.Retryable() {
		return 0, false
	}
	if outcome.ResetAfter > 0 {
		return outcome.ResetAfter, true
	}
	base, max := p.Base, p.Max
	if base <= 0 {
		base = time.Second
	}
	if max <= 0 {
		max = time.Minute
	}
	if attempt < 0 {
		attempt = 0
	}
	d := time.Duration(float64(base) * math.Pow(2, float64(attempt)))
	if d > max || d < 0 {
		d = max
	}
	if p.Jitter > 0 {
		r := rand.Float64()
		if p.Rand != nil {
			r = p.Rand.Float64()
		}
		d = time.Duration(float64(d) * (1 - p.Jitter + 2*p.Jitter*r))
		if d > max {
			d = max
		}
	}
	return d, true
}

type Cooldown struct {
	Runtime string    `json:"runtime"`
	Kind    Kind      `json:"kind"`
	Reason  string    `json:"reason"`
	Until   time.Time `json:"until"`
}

type Breaker struct {
	Dir string
	Now func() time.Time
}

func (b Breaker) Open(runtime string) (Cooldown, bool, error) {
	var c Cooldown
	raw, err := os.ReadFile(b.path(runtime))
	if errors.Is(err, os.ErrNotExist) {
		return c, false, nil
	}
	if err != nil {
		return c, false, err
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, false, fmt.Errorf("read runtime cooldown: %w", err)
	}
	now := time.Now()
	if b.Now != nil {
		now = b.Now()
	}
	return c, now.Before(c.Until), nil
}

func (b Breaker) Record(runtime string, outcome Outcome, cooldown time.Duration) (Cooldown, error) {
	now := time.Now()
	if b.Now != nil {
		now = b.Now()
	}
	c := Cooldown{Runtime: runtime, Kind: outcome.Kind, Reason: outcome.Reason, Until: now.Add(cooldown)}
	if err := os.MkdirAll(b.Dir, 0o755); err != nil {
		return c, err
	}
	raw, _ := json.MarshalIndent(c, "", "  ")
	return c, os.WriteFile(b.path(runtime), append(raw, '\n'), 0o644)
}

func (b Breaker) path(runtime string) string {
	runtime = regexp.MustCompile(`[^a-zA-Z0-9._-]`).ReplaceAllString(runtime, "_")
	return filepath.Join(b.Dir, runtime+".json")
}

// EligibleFallback preserves the source security floor: grants cannot weaken
// and every required capability must remain declared by the destination.
func EligibleFallback(sourceGrant, destinationGrant string, required, destination []string) bool {
	strength := map[string]int{"ro": 1, "rw": 2}
	if strength[destinationGrant] < strength[sourceGrant] {
		return false
	}
	have := map[string]bool{}
	for _, c := range destination {
		have[c] = true
	}
	for _, c := range required {
		if !have[c] {
			return false
		}
	}
	return true
}
