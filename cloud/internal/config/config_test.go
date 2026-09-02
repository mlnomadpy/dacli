package config

import (
	"fmt"
	"strings"
	"testing"
)

const validConfig = `{"mode":"production","listen_address":"127.0.0.1:8080","public_base_url":"https://control.example.test","request_timeout":"10s","shutdown_timeout":"15s","max_request_bytes":65536,"worker_interval":"5s","database_url_env":"TEST_DATABASE_URL","service_secret_env":"TEST_SERVICE_SECRET","contract_minimum":1,"contract_maximum":1}`

func environment(name string) (string, bool) {
	values := map[string]string{
		"TEST_DATABASE_URL":   "postgres://service:p9x7v3m2q8k4z6n1c5b0r7t9w2y4u8i6@db.example.test/dacli?sslmode=require",
		"TEST_SERVICE_SECRET": "a-cryptographically-random-value-with-ample-length-2026",
	}
	v, ok := values[name]
	return v, ok
}

func TestProductionConfigurationIsStrictAndSecretSafe(t *testing.T) {
	cfg, err := Load(strings.NewReader(validConfig), environment)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(mapValues(cfg.SafeSummary()), " "), "p9x7v3") {
		t.Fatal("safe summary disclosed database credentials")
	}
	if address, err := cfg.DatabaseAddress(); err != nil || address != "db.example.test:5432" {
		t.Fatalf("database address = %q, %v", address, err)
	}
	for name, replacement := range map[string]string{
		"unknown fields":        strings.Replace(validConfig, `"mode":`, `"surprise":true,"mode":`, 1),
		"plaintext production":  strings.Replace(validConfig, "https://", "http://", 1),
		"incompatible contract": strings.Replace(validConfig, `"contract_maximum":1`, `"contract_maximum":2`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(strings.NewReader(replacement), environment); err == nil {
				t.Fatal("unsafe configuration was accepted")
			}
		})
	}
}

func TestProductionRefusesDatabaseWithoutExplicitTLS(t *testing.T) {
	lookup := func(name string) (string, bool) {
		if name == "TEST_DATABASE_URL" {
			return "postgres://service:p9x7v3m2q8k4z6n1c5b0r7t9w2y4u8i6@db.example.test/dacli", true
		}
		return environment(name)
	}
	if _, err := Load(strings.NewReader(validConfig), lookup); err == nil {
		t.Fatal("database URL without explicit TLS was accepted")
	}
}

func TestProductionRefusesDefaultSecret(t *testing.T) {
	lookup := func(name string) (string, bool) {
		if name == "TEST_SERVICE_SECRET" {
			return "changeme-this-is-deliberately-long-but-still-weak", true
		}
		return environment(name)
	}
	if _, err := Load(strings.NewReader(validConfig), lookup); err == nil {
		t.Fatal("default secret was accepted")
	}
}

func mapValues(values map[string]any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, fmt.Sprint(value))
	}
	return out
}
