package publication

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestUnknownVisibilityFailsClosedToPublicSafe(t *testing.T) {
	p := New("acme/widgets", "future-value", true, true, false)
	if p.Mode != "public-safe" || p.Allows(FieldFindings) || p.Allows(FieldIssueClose) {
		t.Fatalf("unknown visibility policy = %+v", p)
	}
	if !p.Allows(FieldAcceptance) || !strings.Contains(p.Reason(FieldFindings), "unknown") {
		t.Fatalf("unknown policy lost safe output or reason: %+v", p)
	}
}

func TestPublicInternalEvidenceNeedsRequestAndRecordedAuthority(t *testing.T) {
	for _, tc := range []struct{ request, recorded, want bool }{{false, false, false}, {true, false, false}, {false, true, false}, {true, true, true}} {
		p := New("acme/widgets", "PUBLIC", tc.request, tc.recorded, false)
		if got := p.Allows(FieldFindings); got != tc.want {
			t.Fatalf("request=%t recorded=%t findings=%t, want %t", tc.request, tc.recorded, got, tc.want)
		}
	}
}

func TestPrivateAndJSONTextUseSameTypedPolicy(t *testing.T) {
	p := New("acme/private", "PRIVATE", false, false, true)
	if !p.Allows(FieldFindings) || !p.Allows(FieldDecisions) || !p.Allows(FieldIssueClose) {
		t.Fatalf("private policy unexpectedly narrowed: %+v", p)
	}
	var text, machine bytes.Buffer
	p.WriteText(&text)
	if err := p.WriteJSON(&machine); err != nil {
		t.Fatal(err)
	}
	var decoded Policy
	if err := json.Unmarshal(machine.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Schema != Schema || decoded.Mode != p.Mode || !strings.Contains(text.String(), p.Mode) {
		t.Fatalf("surfaces drifted: text=%q json=%+v", text.String(), decoded)
	}
}

func TestPublicSafeSanitizesPrivateContentInsideAllowedFields(t *testing.T) {
	p := New("acme/widgets", "PUBLIC", false, false, false)
	got := p.Sanitize("a-worker read /private/operator/key.txt token=abc123")
	for _, leaked := range []string{"a-worker", "/private/operator", "abc123"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("sanitize leaked %q: %q", leaked, got)
		}
	}
	private := New("acme/widgets", "PRIVATE", false, false, false)
	if private.Sanitize("/private/operator/key.txt") != "/private/operator/key.txt" {
		t.Fatal("private projection was redacted")
	}
}

func TestPublicSafePreservesWebLinksWhileRedactingLocalPaths(t *testing.T) {
	p := New("acme/widgets", "PUBLIC", false, false, false)
	const github = "https://github.com/mlnomadpy/dacli/issues/873"
	got := p.Sanitize("See " + github + " from /Users/operator/src/dacli and C:\\Users\\operator\\src\\dacli\\main.go")
	if !strings.Contains(got, github) {
		t.Fatalf("public URL was corrupted: %q", got)
	}
	for _, local := range []string{"/Users/operator", `C:\Users\operator`} {
		if strings.Contains(got, local) {
			t.Fatalf("local path %q leaked: %q", local, got)
		}
	}
	if gotCount := strings.Count(got, "<withheld-local-path>"); gotCount != 2 {
		t.Fatalf("withheld local paths = %d, want 2: %q", gotCount, got)
	}
}

func TestPublicSafeWebLinksDoNotBypassSecretRedaction(t *testing.T) {
	p := New("acme/widgets", "PUBLIC", false, false, false)
	got := p.Sanitize("https://user:password@example.com/path?token=abc123")
	for _, leaked := range []string{"password", "abc123"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("URL secret %q leaked: %q", leaked, got)
		}
	}
	if !strings.HasPrefix(got, "https://user@example.com/path") || !strings.Contains(got, "%3Cwithheld-secret%3E") {
		t.Fatalf("sanitized URL lost its public structure or marker: %q", got)
	}
}
