// Package service owns the control-plane process lifecycle and HTTP boundary.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/config"
	"github.com/mlnomadpy/dacli/cloud/internal/domain"
)

const errorSchema = "controlplane-error/v1"

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

type Readiness interface {
	Ready(context.Context) error
}

type ReadinessFunc func(context.Context) error

func (f ReadinessFunc) Ready(ctx context.Context) error { return f(ctx) }

type API struct {
	config    config.Config
	readiness Readiness
	logger    *slog.Logger
}

func NewAPI(cfg config.Config, readiness Readiness, logger *slog.Logger) *API {
	if logger == nil {
		logger = slog.Default()
	}
	if readiness == nil {
		readiness = ReadinessFunc(func(context.Context) error { return nil })
	}
	return &API{config: cfg, readiness: readiness, logger: logger}
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			a.writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "endpoint requires GET", false)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "alive"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			a.writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "endpoint requires GET", false)
			return
		}
		if err := a.readiness.Ready(r.Context()); err != nil {
			a.writeError(w, r, http.StatusServiceUnavailable, "not_ready", "service dependencies are not ready", true)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "contract_version": domain.ContractVersion})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		a.writeError(w, r, http.StatusNotFound, "not_found", "endpoint does not exist", false)
	})
	return a.boundary(mux)
}

func (a *API) boundary(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if !requestIDPattern.MatchString(requestID) {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		r.Header.Set("X-Request-ID", requestID)
		if r.ContentLength > a.config.MaxRequestBytes {
			a.writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large", "request exceeds the configured byte limit", false)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, a.config.MaxRequestBytes)
		ctx, cancel := context.WithTimeout(r.Context(), a.config.RequestTimeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type errorResponse struct {
	Schema    string `json:"schema"`
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func (a *API) writeError(w http.ResponseWriter, r *http.Request, status int, code, message string, retryable bool) {
	requestID := r.Header.Get("X-Request-ID")
	a.logger.Warn("control-plane request refused", "request_id", requestID, "code", code, "status", status)
	writeJSONStatus(w, status, errorResponse{Schema: errorSchema, RequestID: requestID, Code: code, Message: message, Retryable: retryable})
}

func writeJSON(w http.ResponseWriter, status int, value any) { writeJSONStatus(w, status, value) }

func writeJSONStatus(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func newRequestID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(raw[:])
}

// RunAPI serves until ctx is cancelled, then performs a bounded graceful shutdown.
func RunAPI(ctx context.Context, cfg config.Config, api *API) error {
	listener, err := net.Listen("tcp", cfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("listen for API: %w", err)
	}
	return runOnListener(ctx, cfg, api, listener)
}

func runOnListener(ctx context.Context, cfg config.Config, api *API, listener net.Listener) error {
	server := &http.Server{
		Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: cfg.RequestTimeout, WriteTimeout: cfg.RequestTimeout,
		IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10,
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve API: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shut down API: %w", err)
		}
		if err := <-done; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve API during shutdown: %w", err)
		}
		return nil
	}
}
