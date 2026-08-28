package providerpolicy

import (
	"math/rand"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClassifyProviderOutcomes(t *testing.T) {
	cases := []struct {
		text string
		kind Kind
	}{
		{`{"error":{"type":"rate_limit_error"},"retry_after":17}`, RateLimited},
		{`insufficient_quota: billing hard limit reached`, QuotaExhausted},
		{`authentication_error: invalid API key`, Authentication},
		{`{"type":"system","subtype":"init"}
Not logged in · Please run /login`, Authentication},
		{`HTTP 503 service_unavailable`, Unavailable},
		{`invalid_request: context_length_exceeded`, PermanentInput},
	}
	for _, tc := range cases {
		if got := Classify(1, tc.text); got.Kind != tc.kind {
			t.Errorf("%q: got %s want %s", tc.text, got.Kind, tc.kind)
		}
	}
}

func TestClaudeAuthenticationFailureIsNeverRetriedOrFallbackedAsTransient(t *testing.T) {
	auth := Classify(1, `{"type":"system","subtype":"init"}
Not logged in · Please run /login`)
	if auth.Kind != Authentication || auth.Retryable() || auth.Fallbackable() {
		t.Fatalf("post-init authentication outcome = %+v, retryable=%v fallbackable=%v", auth, auth.Retryable(), auth.Fallbackable())
	}
	transient := Classify(1, `{"type":"system","subtype":"init"}
HTTP 503 service_unavailable`)
	if transient.Kind != Unavailable || !transient.Retryable() || !transient.Fallbackable() {
		t.Fatalf("post-init transient outcome = %+v, retryable=%v fallbackable=%v", transient, transient.Retryable(), transient.Fallbackable())
	}
}

func TestRetryPolicyHonorsResetAndBoundsBackoff(t *testing.T) {
	p := RetryPolicy{Base: time.Second, Max: 8 * time.Second, Jitter: .2, Rand: rand.New(rand.NewSource(1))}
	if got, ok := p.Delay(9, Outcome{Kind: RateLimited, ResetAfter: 23 * time.Second}); !ok || got != 23*time.Second {
		t.Fatalf("metadata delay = %s, %v", got, ok)
	}
	if got, ok := p.Delay(9, Outcome{Kind: Unavailable}); !ok || got <= 0 || got > 8*time.Second {
		t.Fatalf("bounded delay = %s, %v", got, ok)
	}
	for _, kind := range []Kind{PermanentInput, PolicyRefusal, Authentication} {
		if _, ok := p.Delay(0, Outcome{Kind: kind}); ok {
			t.Errorf("%s was retryable", kind)
		}
	}
}

func TestBreakerSurvivesNewInstance(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cooldowns")
	now := time.Unix(100, 0)
	b := Breaker{Dir: dir, Now: func() time.Time { return now }}
	if _, err := b.Record("codex", Outcome{Kind: QuotaExhausted, Reason: "quota"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	got, open, err := (Breaker{Dir: dir, Now: func() time.Time { return now.Add(time.Second) }}).Open("codex")
	if err != nil || !open || got.Kind != QuotaExhausted {
		t.Fatalf("persisted breaker = %+v, %v, %v", got, open, err)
	}
}

func TestFallbackCannotWeakenPolicy(t *testing.T) {
	if EligibleFallback("rw", "ro", []string{"code"}, []string{"code"}) {
		t.Fatal("weaker grant accepted")
	}
	if EligibleFallback("ro", "rw", []string{"code", "vision"}, []string{"code"}) {
		t.Fatal("missing capability accepted")
	}
	if !EligibleFallback("ro", "rw", []string{"code"}, []string{"code", "vision"}) {
		t.Fatal("stronger fallback rejected")
	}
	for _, kind := range []Kind{PermanentInput, PolicyRefusal} {
		if (Outcome{Kind: kind}).Fallbackable() {
			t.Errorf("%s triggered fallback", kind)
		}
	}
}

func TestTransitionNamesBothRuntimesReasonAndCooldown(t *testing.T) {
	got := (Transition{Source: "codex", Destination: "claude", Reason: "quota_exhausted", Cooldown: time.Hour}).String()
	for _, want := range []string{"source=codex", "destination=claude", "reason=quota_exhausted", "cooldown=1h0m0s"} {
		if !strings.Contains(got, want) {
			t.Errorf("transition %q missing %q", got, want)
		}
	}
}
