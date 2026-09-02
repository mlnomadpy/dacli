package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/config"
)

func testConfig() config.Config {
	return config.Config{ListenAddress: "127.0.0.1:0", RequestTimeout: time.Second, ShutdownTimeout: time.Second, MaxRequestBytes: 1024}
}

func TestHealthReadinessRequestIDsAndRedactedErrors(t *testing.T) {
	var logs bytes.Buffer
	api := NewAPI(testConfig(), ReadinessFunc(func(context.Context) error { return errors.New("postgres://user:secret@private.example/db") }), slog.New(slog.NewJSONHandler(&logs, nil)))
	healthRequest := httptest.NewRequest("GET", "/healthz", nil)
	health := httptest.NewRecorder()
	api.Handler().ServeHTTP(health, healthRequest)
	if health.Code != 200 || health.Header().Get("X-Request-ID") == "" {
		t.Fatalf("health response = %d id=%q", health.Code, health.Header().Get("X-Request-ID"))
	}

	request := httptest.NewRequest("GET", "/readyz", nil)
	request.Header.Set("X-Request-ID", "caller-123")
	ready := httptest.NewRecorder()
	api.Handler().ServeHTTP(ready, request)
	body := ready.Body.Bytes()
	if ready.Code != 503 || ready.Header().Get("X-Request-ID") != "caller-123" || !bytes.Contains(body, []byte(`"code":"not_ready"`)) {
		t.Fatalf("readiness response = %d %s", ready.Code, body)
	}
	if strings.Contains(logs.String(), "secret") || strings.Contains(string(body), "secret") {
		t.Fatalf("diagnostic disclosed dependency error: log=%s body=%s", logs.String(), body)
	}
}

func TestRequestBoundaryRejectsOversizeAndInvalidRequestID(t *testing.T) {
	api := NewAPI(testConfig(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest("POST", "/healthz", strings.NewReader(strings.Repeat("x", 1025)))
	request.Header.Set("X-Request-ID", "unsafe\nheader")
	recorder := httptest.NewRecorder()
	api.Handler().ServeHTTP(recorder, request)
	if recorder.Code != 413 {
		t.Fatalf("status = %d", recorder.Code)
	}
	if id := recorder.Header().Get("X-Request-ID"); id == "" || strings.Contains(id, "unsafe") {
		t.Fatalf("request id = %q", id)
	}
}

func TestAllHTTPRefusalsUseStructuredSafeEnvelope(t *testing.T) {
	api := NewAPI(testConfig(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	fixtures := []struct{ name, method, path, code string }{
		{"not found", "GET", "/missing", "not_found"},
		{"wrong method", "POST", "/healthz", "method_not_allowed"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			request := httptest.NewRequest(fixture.method, fixture.path, nil)
			recorder := httptest.NewRecorder()
			api.Handler().ServeHTTP(recorder, request)
			body := recorder.Body.String()
			if recorder.Header().Get("Content-Type") != "application/json" || !strings.Contains(body, `"schema":"controlplane-error/v1"`) || !strings.Contains(body, `"code":"`+fixture.code+`"`) {
				t.Fatalf("refusal = %d %s", recorder.Code, body)
			}
		})
	}
}

func TestRunOnListenerShutsDownGracefully(t *testing.T) {
	listener := newBlockingListener()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runOnListener(ctx, testConfig(), NewAPI(testConfig(), nil, nil), listener) }()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down")
	}
}

type blockingListener struct {
	closed chan struct{}
}

func newBlockingListener() *blockingListener { return &blockingListener{closed: make(chan struct{})} }
func (l *blockingListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, net.ErrClosed
}
func (l *blockingListener) Close() error {
	select {
	case <-l.closed:
	default:
		close(l.closed)
	}
	return nil
}
func (l *blockingListener) Addr() net.Addr { return testAddr("memory") }

type testAddr string

func (a testAddr) Network() string { return string(a) }
func (a testAddr) String() string  { return string(a) }
