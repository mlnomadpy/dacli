// Package config loads the control-plane process configuration.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

const maxConfigBytes = 64 << 10

// File is the non-secret configuration persisted by operators.
type File struct {
	Mode             string `json:"mode"`
	ListenAddress    string `json:"listen_address"`
	PublicBaseURL    string `json:"public_base_url"`
	RequestTimeout   string `json:"request_timeout"`
	ShutdownTimeout  string `json:"shutdown_timeout"`
	MaxRequestBytes  int64  `json:"max_request_bytes"`
	WorkerInterval   string `json:"worker_interval"`
	DatabaseURLEnv   string `json:"database_url_env"`
	ServiceSecretEnv string `json:"service_secret_env"`
	ContractMinimum  int    `json:"contract_minimum"`
	ContractMaximum  int    `json:"contract_maximum"`
}

// Config is the validated runtime configuration. Secrets remain private.
type Config struct {
	Mode            string
	ListenAddress   string
	PublicBaseURL   *url.URL
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
	MaxRequestBytes int64
	WorkerInterval  time.Duration
	ContractMinimum int
	ContractMaximum int
	databaseURL     string
	serviceSecret   string
}

// LookupEnv allows tests and embedding processes to provide a controlled environment.
type LookupEnv func(string) (string, bool)

func Load(r io.Reader, lookup LookupEnv) (Config, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	limited := io.LimitReader(r, maxConfigBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return Config{}, fmt.Errorf("read configuration: %w", err)
	}
	if len(raw) > maxConfigBytes {
		return Config{}, errors.New("configuration exceeds 64 KiB")
	}
	var f File
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&f); err != nil {
		return Config{}, fmt.Errorf("decode configuration: %w", err)
	}
	if err := requireEOF(decoder); err != nil {
		return Config{}, err
	}
	return validate(f, lookup)
}

func LoadFile(path string, lookup LookupEnv) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open configuration: %w", err)
	}
	defer func() { _ = f.Close() }()
	return Load(f, lookup)
}

func requireEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("configuration contains multiple JSON values")
		}
		return fmt.Errorf("decode trailing configuration: %w", err)
	}
	return nil
}

func validate(f File, lookup LookupEnv) (Config, error) {
	if f.Mode != "development" && f.Mode != "production" {
		return Config{}, errors.New("mode must be development or production")
	}
	if _, port, err := net.SplitHostPort(f.ListenAddress); err != nil || port == "" {
		return Config{}, errors.New("listen_address must include a host and port")
	}
	base, err := url.Parse(f.PublicBaseURL)
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return Config{}, errors.New("public_base_url must be an absolute HTTP(S) URL")
	}
	if base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return Config{}, errors.New("public_base_url must not contain credentials, query, or fragment")
	}
	if f.Mode == "production" && base.Scheme != "https" {
		return Config{}, errors.New("production public_base_url must use HTTPS")
	}
	requestTimeout, err := boundedDuration("request_timeout", f.RequestTimeout, time.Second, 2*time.Minute)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := boundedDuration("shutdown_timeout", f.ShutdownTimeout, time.Second, 2*time.Minute)
	if err != nil {
		return Config{}, err
	}
	workerInterval, err := boundedDuration("worker_interval", f.WorkerInterval, time.Second, time.Hour)
	if err != nil {
		return Config{}, err
	}
	if f.MaxRequestBytes < 1024 || f.MaxRequestBytes > 1<<20 {
		return Config{}, errors.New("max_request_bytes must be between 1024 and 1048576")
	}
	if f.ContractMinimum != 1 || f.ContractMaximum != 1 {
		return Config{}, errors.New("this binary supports exactly control-plane contract version 1")
	}
	databaseURL, err := secretFromEnv("database_url_env", f.DatabaseURLEnv, lookup)
	if err != nil {
		return Config{}, err
	}
	serviceSecret, err := secretFromEnv("service_secret_env", f.ServiceSecretEnv, lookup)
	if err != nil {
		return Config{}, err
	}
	if f.Mode == "production" {
		if weakSecret(serviceSecret) {
			return Config{}, errors.New("production service secret is missing, too short, or a known default")
		}
		dbURL, parseErr := url.Parse(databaseURL)
		if parseErr != nil || (dbURL.Scheme != "postgres" && dbURL.Scheme != "postgresql") || dbURL.Host == "" {
			return Config{}, errors.New("production database URL must be an absolute PostgreSQL URL")
		}
		switch dbURL.Query().Get("sslmode") {
		case "require", "verify-ca", "verify-full":
		default:
			return Config{}, errors.New("production database URL must explicitly require or verify TLS")
		}
		if dbURL.User == nil {
			return Config{}, errors.New("production database credential is missing, too short, or a known default")
		}
		password, present := dbURL.User.Password()
		if !present || weakSecret(password) {
			return Config{}, errors.New("production database credential is missing, too short, or a known default")
		}
	}
	return Config{
		Mode: f.Mode, ListenAddress: f.ListenAddress, PublicBaseURL: base,
		RequestTimeout: requestTimeout, ShutdownTimeout: shutdownTimeout,
		MaxRequestBytes: f.MaxRequestBytes, WorkerInterval: workerInterval,
		ContractMinimum: f.ContractMinimum, ContractMaximum: f.ContractMaximum,
		databaseURL: databaseURL, serviceSecret: serviceSecret,
	}, nil
}

func boundedDuration(name, value string, minimum, maximum time.Duration) (time.Duration, error) {
	d, err := time.ParseDuration(value)
	if err != nil || d < minimum || d > maximum {
		return 0, fmt.Errorf("%s must be a duration between %s and %s", name, minimum, maximum)
	}
	return d, nil
}

func secretFromEnv(field, name string, lookup LookupEnv) (string, error) {
	if name == "" || strings.ContainsAny(name, "= \t\r\n") {
		return "", fmt.Errorf("%s must name one environment variable", field)
	}
	value, ok := lookup(name)
	if !ok || value == "" {
		return "", fmt.Errorf("%s environment variable is not set", field)
	}
	return value, nil
}

func weakSecret(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if len(value) < 32 {
		return true
	}
	for _, marker := range []string{"changeme", "change-me", "password", "default", "example", "secret"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

// DatabaseURL returns the resolved database credential to infrastructure adapters only.
func (c Config) DatabaseURL() string { return c.databaseURL }

// ServiceSecret returns the resolved signing credential to infrastructure adapters only.
func (c Config) ServiceSecret() string { return c.serviceSecret }

// DatabaseAddress returns the credential-free PostgreSQL network address.
func (c Config) DatabaseAddress() (string, error) {
	database, err := url.Parse(c.databaseURL)
	if err != nil || database.Hostname() == "" {
		return "", errors.New("database URL has no network host")
	}
	port := database.Port()
	if port == "" {
		port = "5432"
	}
	return net.JoinHostPort(database.Hostname(), port), nil
}

// SafeSummary is deliberately safe for structured logs and diagnostics.
func (c Config) SafeSummary() map[string]any {
	return map[string]any{"mode": c.Mode, "listen_address": c.ListenAddress, "public_base_url": c.PublicBaseURL.String(), "contract_minimum": c.ContractMinimum, "contract_maximum": c.ContractMaximum}
}
