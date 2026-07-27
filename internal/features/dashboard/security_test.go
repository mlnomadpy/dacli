package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// do drives the handler for one request with an explicit Host header and
// returns the recorded status code — the shared primitive for the hardening
// tests (path traversal, DNS-rebinding, and read-only method gate).
func do(t *testing.T, h http.Handler, method, target, host string) int {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	req.Host = host
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	return rw.Code
}

// TestProjectParamTraversalRejected proves finding 181 issue 1 is closed: an
// unsafe ?project= value (`..`, a separator, or an absolute path) is rejected
// with 400 on every handler that reads it, rather than silently returning an
// empty result. A well-formed slug — and the whole-board empty filter — still
// answers 200.
func TestProjectParamTraversalRejected(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	traversals := []string{
		"../../other-workspace",
		"..",
		"a/b",
		"/etc/passwd",
		"../core",
	}
	for _, endpoint := range []string{"/api/tasks", "/api/graph"} {
		for _, bad := range traversals {
			target := endpoint + "?project=" + bad
			if code := do(t, h, "GET", target, "localhost"); code != http.StatusBadRequest {
				t.Errorf("GET %s = %d, want 400", target, code)
			}
		}
		// A safe slug is accepted (200), even one that names no project — that is a
		// valid "no rows" query, not an attack.
		if code := do(t, h, "GET", endpoint+"?project=core", "localhost"); code != http.StatusOK {
			t.Errorf("GET %s?project=core = %d, want 200", endpoint, code)
		}
		if code := do(t, h, "GET", endpoint+"?project=nope", "localhost"); code != http.StatusOK {
			t.Errorf("GET %s?project=nope = %d, want 200 (safe unknown slug)", endpoint, code)
		}
		// The whole-board query (no filter) is legitimate and must stay 200.
		if code := do(t, h, "GET", endpoint, "localhost"); code != http.StatusOK {
			t.Errorf("GET %s (no project) = %d, want 200", endpoint, code)
		}
	}
}

// TestForeignHostRejected proves finding 181 issue 2 is closed: the loopback-only
// server refuses any request whose Host is not a recognized loopback name (the
// DNS-rebinding defense), while a normal localhost request still works.
func TestForeignHostRejected(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	foreign := []string{
		"evil.example.com",
		"attacker.test:8080",
		"dacli.local",
		"", // a missing Host is not a recognized loopback name
	}
	for _, host := range foreign {
		if code := do(t, h, "GET", "/api/state", host); code != http.StatusForbidden {
			t.Errorf("GET /api/state with Host %q = %d, want 403", host, code)
		}
	}

	// Every loopback form the server binds to is accepted, with and without a port.
	for _, host := range []string{"localhost", "localhost:5173", "127.0.0.1", "127.0.0.1:41234", "[::1]:8080"} {
		if code := do(t, h, "GET", "/api/state", host); code != http.StatusOK {
			t.Errorf("GET /api/state with Host %q = %d, want 200", host, code)
		}
	}
}

// TestAPIIsReadOnly proves the (lower-priority) method gate: the read-only API
// refuses anything but GET with 405, so a cross-origin form POST cannot reach a
// handler even if it somehow forged a loopback Host.
func TestAPIIsReadOnly(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	for _, method := range []string{"POST", "PUT", "DELETE", "PATCH"} {
		if code := do(t, h, method, "/api/state", "localhost"); code != http.StatusMethodNotAllowed {
			t.Errorf("%s /api/state = %d, want 405", method, code)
		}
	}
	if code := do(t, h, "GET", "/api/state", "localhost"); code != http.StatusOK {
		t.Errorf("GET /api/state = %d, want 200", code)
	}
}
